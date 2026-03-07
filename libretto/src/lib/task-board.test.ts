import test from 'node:test'
import assert from 'node:assert/strict'
import { appendCreatedTask, applyTaskUpdate, observabilityStatus, selectPreferredIssue } from './task-board.ts'
import type { DeliveryInsights, RuntimeState, TaskListResponse, TaskRecord } from './api.ts'

test('selectPreferredIssue keeps the selected issue when refreshed data still contains it', () => {
  const selected = selectPreferredIssue(sampleRuntimeState(), sampleTaskList(), 'SYM-2')
  assert.equal(selected, 'SYM-2')
})

test('appendCreatedTask adds the task and updates counts by state', () => {
  const next = appendCreatedTask(sampleTaskList(), sampleTask({ id: 'task-3', identifier: 'SYM-3', state: 'Review' }))
  assert.equal(next.counts.total, 3)
  assert.equal(next.counts.by_state.review, 1)
  assert.equal(next.tasks.at(-1)?.identifier, 'SYM-3')
})

test('applyTaskUpdate replaces the task and moves it across state counts', () => {
  const next = applyTaskUpdate(sampleTaskList(), sampleTask({ id: 'task-2', identifier: 'SYM-2', state: 'Done' }))
  assert.equal(next.counts.by_state.todo, 1)
  assert.equal(next.counts.by_state.done, 1)
  assert.equal(next.tasks.find((task) => task.identifier === 'SYM-2')?.state, 'Done')
})

test('observabilityStatus distinguishes healthy, degraded, and unavailable states', () => {
  assert.equal(observabilityStatus(sampleDeliveryReport(), null), 'degraded')
  assert.equal(observabilityStatus({ ...sampleDeliveryReport(), warnings: [] }, null), 'healthy')
  assert.equal(observabilityStatus(null, 'delivery unavailable'), 'unavailable')
})

function sampleRuntimeState(): RuntimeState {
  return {
    generated_at: '2026-03-07T12:00:00Z',
    counts: { running: 1, retrying: 1 },
    running: [
      {
        kind: 'running',
        issue_id: 'issue-1',
        issue_identifier: 'SYM-1',
        state: 'In Progress',
        session_id: 'thread-1-turn-1',
        turn_count: 1,
        last_event: 'turn.started',
        last_message: '',
        started_at: '2026-03-07T12:00:00Z',
        last_event_at: null,
        tokens: { input_tokens: 1, output_tokens: 1, total_tokens: 2 },
      },
    ],
    retrying: [
      {
        kind: 'retrying',
        issue_id: 'issue-2',
        issue_identifier: 'SYM-2',
        attempt: 1,
        due_at: '2026-03-07T12:05:00Z',
        error: 'retry scheduled',
      },
    ],
    codex_totals: {
      input_tokens: 1,
      output_tokens: 1,
      total_tokens: 2,
      seconds_running: 5,
    },
    rate_limits: null,
  }
}

function sampleTaskList(): TaskListResponse {
  return {
    tasks: [
      sampleTask({ id: 'task-1', identifier: 'SYM-1', state: 'Todo' }),
      sampleTask({ id: 'task-2', identifier: 'SYM-2', state: 'In Progress' }),
    ],
    counts: {
      total: 2,
      by_state: {
        todo: 1,
        'in progress': 1,
      },
    },
  }
}

function sampleTask(partial: Partial<TaskRecord>): TaskRecord {
  return {
    id: partial.id ?? 'task-1',
    identifier: partial.identifier ?? 'SYM-1',
    title: partial.title ?? 'Task',
    description: partial.description ?? null,
    priority: partial.priority ?? null,
    state: partial.state ?? 'Todo',
    branch_name: partial.branch_name ?? null,
    url: partial.url ?? null,
    labels: partial.labels ?? [],
    blocked_by: partial.blocked_by ?? [],
    created_at: partial.created_at ?? null,
    updated_at: partial.updated_at ?? null,
  }
}

function sampleDeliveryReport(): DeliveryInsights {
  return {
    generated_at: '2026-03-07T12:00:00Z',
    summary: {
      delivery_health: { key: 'delivery_health', label: 'Delivery health', score: 78, status: 'watch', detail: '' },
      flow_efficiency: { key: 'flow_efficiency', label: 'Flow efficiency', score: 72, status: 'watch', detail: '' },
      merge_readiness: { key: 'merge_readiness', label: 'Merge readiness', score: 64, status: 'watch', detail: '' },
      predictability: { key: 'predictability', label: 'Predictability', score: 71, status: 'watch', detail: '' },
    },
    tracker: {
      total_tasks: 2,
      active_tasks: 1,
      blocked_tasks: 0,
      review_tasks: 0,
      done_last_window: 0,
      avg_active_age_hours: 4,
      backlog_pressure: 1,
      runtime: {
        running_sessions: 1,
        retrying_sessions: 1,
        active_tokens: 2,
      },
      agile: {
        throughput_last_window: 0,
        completion_ratio: 0,
        review_load: 0,
      },
      kanban: {
        wip_count: 1,
        blocked_ratio: 0,
        aging_work_ratio: 0.1,
        flow_load: 0.5,
      },
    },
    scm: {
      active_sources: 0,
      totals: {
        branches: 0,
        unmerged_branches: 0,
        stale_branches: 0,
        drift_commits: 0,
        ahead_commits: 0,
        max_age_hours: 0,
        open_change_requests: 0,
        approved_change_requests: 0,
        failing_change_requests: 0,
        stale_change_requests: 0,
      },
      sources: [],
    },
    warnings: ['scm metrics degraded'],
  }
}
