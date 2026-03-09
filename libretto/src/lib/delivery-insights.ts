import type { DeliveryInsights, DeliveryMetricCard } from './api'

export type DeliveryRollupAlert = {
  severity: 'critical' | 'warning'
  title: string
  detail: string
}

export function filterDeliveryRollupAlerts(
  alerts: DeliveryRollupAlert[],
  severity: 'all' | DeliveryRollupAlert['severity']
): DeliveryRollupAlert[] {
  if (severity === 'all') {
    return alerts
  }
  return alerts.filter((alert) => alert.severity === severity)
}

export function orderedDeliveryCards(report: DeliveryInsights): DeliveryMetricCard[] {
  return [report.summary.delivery_health, report.summary.flow_efficiency, report.summary.merge_readiness, report.summary.predictability]
}

export function hasDeliveryWarnings(report: DeliveryInsights | null): boolean {
  return Boolean(report && report.warnings.length > 0)
}

export function deliveryObservabilityState(report: DeliveryInsights | null, error: string | null): 'healthy' | 'degraded' | 'unavailable' {
  if (error && !report) {
    return 'unavailable'
  }
  if (hasDeliveryWarnings(report)) {
    return 'degraded'
  }
  return 'healthy'
}

export function buildDeliveryRollupAlerts(report: DeliveryInsights): DeliveryRollupAlert[] {
  const alerts: DeliveryRollupAlert[] = []

  for (const card of orderedDeliveryCards(report)) {
    if (card.status === 'risk') {
      alerts.push({
        severity: 'critical',
        title: `${card.label} is at risk`,
        detail: card.detail,
      })
    } else if (card.status === 'watch') {
      alerts.push({
        severity: 'warning',
        title: `${card.label} needs attention`,
        detail: card.detail,
      })
    }
  }

  if (report.scm.totals.failing_change_requests > 0) {
    alerts.push({
      severity: 'critical',
      title: 'Failing change requests detected',
      detail: `${report.scm.totals.failing_change_requests} change request(s) currently have failing checks.`,
    })
  }

  if (report.scm.totals.stale_change_requests > 0) {
    alerts.push({
      severity: 'warning',
      title: 'Stale change requests need review',
      detail: `${report.scm.totals.stale_change_requests} change request(s) are stale.`,
    })
  }

  if (report.scm.totals.stale_branches > 0) {
    alerts.push({
      severity: 'warning',
      title: 'Stale branches are accumulating',
      detail: `${report.scm.totals.stale_branches} branch(es) exceed the stale threshold.`,
    })
  }

  report.warnings.forEach((warning) => {
    alerts.push({
      severity: 'warning',
      title: 'Delivery metrics warning',
      detail: warning,
    })
  })

  report.scm.sources.forEach((source) => {
    source.warnings?.forEach((warning) => {
      alerts.push({
        severity: 'warning',
        title: `${source.name} source warning`,
        detail: warning,
      })
    })
  })

  const unique = dedupeAlerts(alerts)
  unique.sort((left, right) => severityRank(left.severity) - severityRank(right.severity))
  return unique.slice(0, 6)
}

function dedupeAlerts(alerts: DeliveryRollupAlert[]): DeliveryRollupAlert[] {
  const seen = new Set<string>()
  return alerts.filter((alert) => {
    const key = `${alert.severity}:${alert.title}:${alert.detail}`
    if (seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  })
}

function severityRank(severity: DeliveryRollupAlert['severity']): number {
  return severity === 'critical' ? 0 : 1
}
