import type { DeliveryInsights, RuntimeState, TaskListResponse, TaskRecord } from './api'
import { hasDeliveryWarnings } from './delivery-insights.ts'

export function selectPreferredIssue(
  runtime: RuntimeState | null,
  tasks: TaskListResponse | null,
  currentSelected: string | null
): string | null {
  const available = new Set<string>()
  for (const item of runtime?.running ?? []) {
    available.add(item.issue_identifier)
  }
  for (const item of runtime?.retrying ?? []) {
    available.add(item.issue_identifier)
  }
  for (const task of tasks?.tasks ?? []) {
    available.add(task.identifier)
  }
  if (currentSelected && available.has(currentSelected)) {
    return currentSelected
  }
  for (const item of runtime?.running ?? []) {
    return item.issue_identifier
  }
  for (const item of runtime?.retrying ?? []) {
    return item.issue_identifier
  }
  for (const task of tasks?.tasks ?? []) {
    return task.identifier
  }
  return null
}

export function appendCreatedTask(tasks: TaskListResponse | null, task: TaskRecord): TaskListResponse {
  const next = tasks?.tasks ? [...tasks.tasks, task] : [task]
  return buildTaskListResponse(next)
}

export function applyTaskUpdate(tasks: TaskListResponse | null, task: TaskRecord): TaskListResponse {
  const existing = tasks?.tasks ?? []
  const next = existing.some((candidate) => candidate.identifier === task.identifier)
    ? existing.map((candidate) => (candidate.identifier === task.identifier ? task : candidate))
    : [...existing, task]
  return buildTaskListResponse(next)
}

export function observabilityStatus(report: DeliveryInsights | null, error: string | null): 'healthy' | 'degraded' | 'unavailable' {
  if (error && !report) {
    return 'unavailable'
  }
  if (hasDeliveryWarnings(report)) {
    return 'degraded'
  }
  return 'healthy'
}

function buildTaskListResponse(tasks: TaskRecord[]): TaskListResponse {
  const sorted = [...tasks].sort((left, right) => left.identifier.localeCompare(right.identifier))
  const byState: Record<string, number> = {}
  for (const task of sorted) {
    const key = task.state.trim().toLowerCase()
    byState[key] = (byState[key] ?? 0) + 1
  }
  return {
    tasks: sorted,
    counts: {
      total: sorted.length,
      by_state: byState,
    },
  }
}
