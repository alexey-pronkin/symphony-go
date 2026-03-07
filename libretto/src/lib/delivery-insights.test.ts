import test from 'node:test'
import assert from 'node:assert/strict'
import { hasDeliveryWarnings, orderedDeliveryCards } from './delivery-insights.ts'
import type { DeliveryInsights } from './api.ts'

test('orderedDeliveryCards keeps the dashboard metric order stable', () => {
  const cards = orderedDeliveryCards(sampleReport())
  assert.deepEqual(
    cards.map((card) => card.key),
    ['delivery_health', 'flow_efficiency', 'merge_readiness', 'predictability']
  )
})

test('hasDeliveryWarnings reports degraded delivery metrics state', () => {
  assert.equal(hasDeliveryWarnings(sampleReport()), true)
  assert.equal(hasDeliveryWarnings({ ...sampleReport(), warnings: [] }), false)
  assert.equal(hasDeliveryWarnings(null), false)
})

function sampleReport(): DeliveryInsights {
  return {
    generated_at: '2026-03-07T12:00:00Z',
    summary: {
      delivery_health: {
        key: 'delivery_health',
        label: 'Delivery health',
        score: 81,
        status: 'strong',
        detail: '2 active tasks, 1 blocked, 0 retrying sessions.',
      },
      flow_efficiency: {
        key: 'flow_efficiency',
        label: 'Flow efficiency',
        score: 76,
        status: 'watch',
        detail: '3 completed in window, review load 20%.',
      },
      merge_readiness: {
        key: 'merge_readiness',
        label: 'Merge readiness',
        score: 69,
        status: 'watch',
        detail: '2 unmerged branches, 4 drift commits.',
      },
      predictability: {
        key: 'predictability',
        label: 'Predictability',
        score: 73,
        status: 'watch',
        detail: 'completion ratio 42%, backlog pressure 1.2x.',
      },
    },
    tracker: {
      total_tasks: 8,
      active_tasks: 2,
      blocked_tasks: 1,
      review_tasks: 1,
      done_last_window: 3,
      avg_active_age_hours: 18,
      backlog_pressure: 1.2,
      runtime: {
        running_sessions: 1,
        retrying_sessions: 0,
        active_tokens: 42,
      },
      agile: {
        throughput_last_window: 3,
        completion_ratio: 0.42,
        review_load: 0.2,
      },
      kanban: {
        wip_count: 2,
        blocked_ratio: 0.12,
        aging_work_ratio: 0.28,
        flow_load: 0.38,
      },
    },
    scm: {
      active_sources: 1,
      totals: {
        branches: 3,
        unmerged_branches: 2,
        stale_branches: 1,
        drift_commits: 4,
        ahead_commits: 3,
        max_age_hours: 96,
      },
      sources: [],
    },
    warnings: ['scm metrics degraded'],
  }
}
