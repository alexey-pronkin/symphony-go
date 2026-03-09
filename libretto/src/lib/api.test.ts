import test from 'node:test'
import assert from 'node:assert/strict'
import { createSymphonyClient } from './api.ts'

test('fetchState uses same-origin path by default', async () => {
  let requestUrl = ''
  const client = createSymphonyClient({
    fetcher: async (input) => {
      requestUrl = String(input)
      return jsonResponse({
        generated_at: '2026-03-07T12:00:00Z',
        counts: { running: 1, retrying: 0 },
        running: [
          {
            issue_id: 'issue-1',
            issue_identifier: 'MT-1',
            state: 'In Progress',
            session_id: 'thread-1-turn-1',
            turn_count: 1,
            last_event: 'turn/started',
            last_message: '',
            started_at: '2026-03-07T12:00:00Z',
            last_event_at: null,
            tokens: { input_tokens: 1, output_tokens: 2, total_tokens: 3 },
          },
        ],
        retrying: [],
        codex_totals: {
          input_tokens: 1,
          output_tokens: 2,
          total_tokens: 3,
          seconds_running: 5,
        },
        rate_limits: null,
      })
    },
  })

  const state = await client.fetchState()
  assert.equal(requestUrl, '/api/v1/state')
  assert.equal(state.running[0]?.kind, 'running')
})

test('fetchIssue uses configured base URL', async () => {
  let requestUrl = ''
  const client = createSymphonyClient({
    baseUrl: 'http://127.0.0.1:18080/',
    fetcher: async (input) => {
      requestUrl = String(input)
      return jsonResponse({
        issue_identifier: 'MT-1',
        issue_id: 'issue-1',
        status: 'running',
        workspace: { path: '/tmp/MT-1' },
        attempts: { restart_count: 0, current_retry_attempt: 0 },
        running: null,
        retry: null,
        last_error: null,
        tracked: {},
      })
    },
  })

  await client.fetchIssue('MT-1')
  assert.equal(requestUrl, 'http://127.0.0.1:18080/api/v1/MT-1')
})

test('refresh posts to refresh endpoint', async () => {
  let method = ''
  const client = createSymphonyClient({
    fetcher: async (_input, init) => {
      method = String(init?.method)
      return jsonResponse({
        queued: true,
        coalesced: false,
        requested_at: '2026-03-07T12:00:00Z',
        operations: ['poll', 'reconcile'],
      })
    },
  })

  const response = await client.refresh()
  assert.equal(method, 'POST')
  assert.equal(response.queued, true)
})

test('task endpoints use expected routes and payloads', async () => {
  const requests: Array<{ url: string; method: string; body: string }> = []
  const client = createSymphonyClient({
    fetcher: async (input, init) => {
      requests.push({
        url: String(input),
        method: String(init?.method ?? 'GET'),
        body: String(init?.body ?? ''),
      })
      if (String(input).endsWith('/api/v1/tasks') && String(init?.method ?? 'GET') === 'GET') {
        return jsonResponse({
          tasks: [],
          counts: {
            total: 0,
            by_state: {},
          },
        })
      }
      return jsonResponse({
        id: 'task-1',
        identifier: 'SYM-1',
        title: 'Created',
        state: 'Todo',
        labels: [],
        blocked_by: [],
      })
    },
  })

  await client.fetchTasks()
  await client.createTask({ title: 'Created', state: 'Todo' })
  await client.updateTask('SYM-1', { state: 'Done' })

  assert.equal(requests[0]?.url, '/api/v1/tasks')
  assert.equal(requests[0]?.method, 'GET')
  assert.equal(requests[1]?.method, 'POST')
  assert.match(requests[1]?.body ?? '', /"title":"Created"/)
  assert.equal(requests[2]?.url, '/api/v1/tasks/SYM-1')
  assert.equal(requests[2]?.method, 'PATCH')
  assert.match(requests[2]?.body ?? '', /"state":"Done"/)
})

test('delivery insights endpoint uses expected route', async () => {
  let requestUrl = ''
  const client = createSymphonyClient({
    fetcher: async (input) => {
      requestUrl = String(input)
      return jsonResponse({
        generated_at: '2026-03-07T12:00:00Z',
        summary: {
          delivery_health: { key: 'delivery_health', label: 'Delivery health', score: 80, status: 'strong', detail: '' },
          flow_efficiency: { key: 'flow_efficiency', label: 'Flow efficiency', score: 70, status: 'watch', detail: '' },
          merge_readiness: { key: 'merge_readiness', label: 'Merge readiness', score: 60, status: 'watch', detail: '' },
          predictability: { key: 'predictability', label: 'Predictability', score: 75, status: 'watch', detail: '' },
        },
        tracker: {
          total_tasks: 1,
          active_tasks: 1,
          blocked_tasks: 0,
          review_tasks: 0,
          done_last_window: 0,
          avg_active_age_hours: 5,
          backlog_pressure: 1,
          runtime: {
            running_sessions: 1,
            retrying_sessions: 0,
            active_tokens: 3,
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
      })
    },
  })

  const payload = await client.fetchDeliveryInsights()
  assert.equal(requestUrl, '/api/v1/insights/delivery')
  assert.equal(payload.summary.delivery_health.score, 80)
})

test('delivery trends endpoint uses expected route', async () => {
  let requestUrl = ''
  const client = createSymphonyClient({
    fetcher: async (input) => {
      requestUrl = String(input)
      return jsonResponse({
        generated_at: '2026-03-08T12:00:00Z',
        window: '7d',
        limit: 12,
        available: true,
        rollups: {
          health_average: 77,
          health_delta: 5,
          health_slope: 2.5,
          flow_average: 71,
          merge_average: 69,
          predictability_trend: 4,
          warning_pressure: 0,
          insufficient_samples: false,
        },
        alerts: [
          {
            key: 'blocked_work',
            label: 'Blocked work',
            severity: 'watch',
            detail: '3 blocked tasks in the latest sample.',
          },
        ],
        points: [
          {
            captured_at: '2026-03-07T12:00:00Z',
            delivery_health: 77,
            flow_efficiency: 71,
            merge_readiness: 69,
            predictability: 73,
            active_tasks: 5,
            blocked_tasks: 1,
            done_last_window: 4,
            wip_count: 3,
            open_change_requests: 2,
            failing_change_checks: 0,
            warning_count: 0,
          },
        ],
        warnings: [],
      })
    },
  })

  const payload = await client.fetchDeliveryTrends()
  assert.equal(requestUrl, '/api/v1/insights/delivery/trends?window=7d&limit=12')
  assert.equal(payload.points[0]?.delivery_health, 77)
  assert.equal(payload.rollups.health_average, 77)
})

test('request surfaces structured API errors', async () => {
  const client = createSymphonyClient({
    fetcher: async () =>
      jsonResponse(
        {
          error: {
            code: 'issue_not_found',
            message: 'issue not found in current runtime state',
          },
        },
        404
      ),
  })

  await assert.rejects(() => client.fetchIssue('MT-404'), /issue not found in current runtime state/)
})

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}
