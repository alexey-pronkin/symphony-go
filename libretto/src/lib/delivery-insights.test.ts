import test from 'node:test'
import assert from 'node:assert/strict'
import {
  buildDeliveryRollupAlerts,
  deliveryObservabilityState,
  filterDeliveryRollupAlerts,
  deliverySourceKey,
  hasDeliveryWarnings,
  orderedDeliveryCards,
} from './delivery-insights.ts'
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

test('deliveryObservabilityState distinguishes degraded and unavailable delivery metrics', () => {
  assert.equal(deliveryObservabilityState(sampleReport(), null), 'degraded')
  assert.equal(deliveryObservabilityState({ ...sampleReport(), warnings: [] }, null), 'healthy')
  assert.equal(deliveryObservabilityState(null, 'delivery unavailable'), 'unavailable')
})

test('buildDeliveryRollupAlerts prioritizes critical issues before warnings', () => {
  const alerts = buildDeliveryRollupAlerts(sampleReport())
  assert.equal(alerts[0]?.severity, 'critical')
  assert.equal(alerts[0]?.title, 'Failing change requests detected')
  assert.match(alerts[0]?.detail ?? '', /1 change request/)
})

test('buildDeliveryRollupAlerts deduplicates repeated warning messages', () => {
  const alerts = buildDeliveryRollupAlerts({
    ...sampleReport(),
    summary: {
      delivery_health: { ...sampleReport().summary.delivery_health, status: 'strong', detail: 'healthy' },
      flow_efficiency: { ...sampleReport().summary.flow_efficiency, status: 'strong', detail: 'healthy' },
      merge_readiness: { ...sampleReport().summary.merge_readiness, status: 'strong', detail: 'healthy' },
      predictability: { ...sampleReport().summary.predictability, status: 'strong', detail: 'healthy' },
    },
    warnings: ['scm metrics degraded', 'scm metrics degraded'],
    scm: {
      ...sampleReport().scm,
      totals: {
        ...sampleReport().scm.totals,
        stale_branches: 0,
        stale_change_requests: 0,
        failing_change_requests: 0,
      },
      sources: [
        {
          ...sampleReport().scm.sources[0],
          failing_change_requests: 0,
          stale_change_requests: 0,
          warnings: ['provider timeout', 'provider timeout'],
        },
      ],
    },
  })
  const duplicateWarnings = alerts.filter((alert) => alert.detail === 'scm metrics degraded')
  const sourceWarnings = alerts.filter((alert) => alert.detail === 'provider timeout')
  assert.equal(duplicateWarnings.length, 1)
  assert.equal(sourceWarnings.length, 1)
})

test('filterDeliveryRollupAlerts narrows the rollup by severity', () => {
  const alerts = buildDeliveryRollupAlerts(sampleReport())
  const critical = filterDeliveryRollupAlerts(alerts, 'critical')
  const warnings = filterDeliveryRollupAlerts(alerts, 'warning')
  assert.ok(critical.length > 0)
  assert.ok(warnings.length > 0)
  assert.ok(critical.every((alert) => alert.severity === 'critical'))
  assert.ok(warnings.every((alert) => alert.severity === 'warning'))
})

test('buildDeliveryRollupAlerts tags source-backed alerts with source keys', () => {
  const report = sampleReport()
  const alerts = buildDeliveryRollupAlerts(report)
  const sourceAlert = alerts.find((alert) => alert.sourceKey)
  assert.equal(sourceAlert?.sourceKey, deliverySourceKey(report.scm.sources[0]))
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
        open_change_requests: 2,
        approved_change_requests: 1,
        failing_change_requests: 1,
        stale_change_requests: 1,
      },
      sources: [
        {
          kind: 'github',
          name: 'platform',
          repo_path: '/tmp/platform',
          main_branch: 'main',
          branches: 3,
          unmerged_branches: 2,
          stale_branches: 1,
          drift_commits: 4,
          ahead_commits: 3,
          max_age_hours: 96,
          open_change_requests: 2,
          approved_change_requests: 1,
          failing_change_requests: 1,
          stale_change_requests: 1,
          merge_readiness: 69,
          warnings: ['provider timeout'],
        },
      ],
    },
    warnings: ['scm metrics degraded'],
  }
}
