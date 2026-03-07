export type RuntimeState = {
  generated_at: string
  counts: {
    running: number
    retrying: number
  }
  running: RunningIssue[]
  retrying: RetryingIssue[]
  codex_totals: {
    input_tokens: number
    output_tokens: number
    total_tokens: number
    seconds_running: number
  }
  rate_limits: Record<string, unknown> | null
}

export type RunningIssue = {
  kind: 'running'
  issue_id: string
  issue_identifier: string
  state: string
  session_id: string
  turn_count: number
  last_event: string
  last_message: string
  started_at: string
  last_event_at: string | null
  tokens: TokenUsage
}

export type RetryingIssue = {
  kind: 'retrying'
  issue_id: string
  issue_identifier: string
  attempt: number
  due_at: string
  error: string
}

export type RuntimeIssue = RunningIssue | RetryingIssue

export type IssueDetail = {
  issue_identifier: string
  issue_id: string
  status: string
  workspace: {
    path: string
  }
  attempts: {
    restart_count: number
    current_retry_attempt: number
  }
  running: {
    session_id: string
    turn_count: number
    state: string
    started_at: string
    last_event: string
    last_message: string
    last_event_at: string | null
    tokens: TokenUsage
  } | null
  retry: {
    issue_id: string
    issue_identifier: string
    attempt: number
    due_at: string
    error: string
  } | null
  logs: {
    codex_session_logs: Array<{
      label: string
      path: string
      url?: string | null
    }>
  }
  recent_events: Array<{
    at: string
    event: string
    message: string
  }>
  last_error: string | null
  tracked: Record<string, unknown>
}

export type RefreshResponse = {
  queued: boolean
  coalesced: boolean
  requested_at: string
  operations: string[]
}

export type TokenUsage = {
  input_tokens: number
  output_tokens: number
  total_tokens: number
}

export type TaskRecord = {
  id: string
  identifier: string
  title: string
  description?: string | null
  priority?: number | null
  state: string
  branch_name?: string | null
  url?: string | null
  labels: string[]
  blocked_by: Array<{
    id?: string | null
    identifier?: string | null
    state?: string | null
  }>
  created_at?: string | null
  updated_at?: string | null
}

export type TaskListResponse = {
  tasks: TaskRecord[]
  counts: {
    total: number
    by_state: Record<string, number>
  }
}

export type CreateTaskInput = {
  title: string
  description?: string
  state?: string
  priority?: number
  labels?: string[]
}

export type UpdateTaskInput = {
  title?: string
  description?: string
  state?: string
  priority?: number
  labels?: string[]
}

type FetchLike = typeof fetch

export function createSymphonyClient(options?: { baseUrl?: string; fetcher?: FetchLike }) {
  const envBaseUrl = (import.meta as ImportMeta & { env?: { VITE_SYMPHONY_API_BASE_URL?: string } }).env?.VITE_SYMPHONY_API_BASE_URL
  const baseUrl = normalizeBaseUrl(options?.baseUrl ?? envBaseUrl ?? '')
  const fetcher = options?.fetcher ?? fetch

  return {
    async fetchState(): Promise<RuntimeState> {
      const payload = await request<RuntimeState>(fetcher, `${baseUrl}/api/v1/state`)
      return {
        ...payload,
        running: payload.running.map((item) => ({ ...item, kind: 'running' as const })),
        retrying: payload.retrying.map((item) => ({
          ...item,
          kind: 'retrying' as const,
        })),
      }
    },
    fetchIssue(issueIdentifier: string): Promise<IssueDetail> {
      return request<IssueDetail>(fetcher, `${baseUrl}/api/v1/${encodeURIComponent(issueIdentifier)}`)
    },
    fetchTasks(): Promise<TaskListResponse> {
      return request<TaskListResponse>(fetcher, `${baseUrl}/api/v1/tasks`)
    },
    createTask(input: CreateTaskInput): Promise<TaskRecord> {
      return request<TaskRecord>(fetcher, `${baseUrl}/api/v1/tasks`, {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateTask(identifier: string, input: UpdateTaskInput): Promise<TaskRecord> {
      return request<TaskRecord>(fetcher, `${baseUrl}/api/v1/tasks/${encodeURIComponent(identifier)}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      })
    },
    refresh(): Promise<RefreshResponse> {
      return request<RefreshResponse>(fetcher, `${baseUrl}/api/v1/refresh`, {
        method: 'POST',
      })
    },
  }
}

export async function request<T>(fetcher: FetchLike, input: string, init?: RequestInit): Promise<T> {
  const response = await fetcher(input, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  })

  let payload: unknown = null
  try {
    payload = await response.json()
  } catch {
    payload = null
  }

  if (!response.ok) {
    throw new Error(readError(payload) ?? `Request failed with status ${response.status}`)
  }

  return payload as T
}

function readError(payload: unknown): string | null {
  if (!payload || typeof payload !== 'object') {
    return null
  }
  const error = (payload as { error?: { message?: unknown } }).error
  if (!error || typeof error.message !== 'string') {
    return null
  }
  return error.message
}

function normalizeBaseUrl(value: string): string {
  return value.trim().replace(/\/+$/, '')
}
