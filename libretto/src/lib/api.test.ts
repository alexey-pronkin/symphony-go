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
