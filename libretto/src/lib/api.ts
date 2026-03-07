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
