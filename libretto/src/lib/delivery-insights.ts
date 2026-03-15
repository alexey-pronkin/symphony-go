import type { DeliveryInsights, DeliveryMetricCard } from './api'

export type DeliveryRollupAlert = {
  severity: 'critical' | 'warning'
  title: string
  detail: string
  sourceKey?: string
}

export type DeliveryAlertFilterCounts = {
  all: number
  critical: number
  warning: number
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

export function countDeliveryRollupAlerts(alerts: DeliveryRollupAlert[]): DeliveryAlertFilterCounts {
  return alerts.reduce<DeliveryAlertFilterCounts>(
    (counts, alert) => {
      counts.all += 1
      counts[alert.severity] += 1
      return counts
    },
    { all: 0, critical: 0, warning: 0 }
  )
}

export function toggleDeliverySourceFocus(currentFocusedSourceKey: string | null, requestedSourceKey: string): string | null {
  return currentFocusedSourceKey === requestedSourceKey ? null : requestedSourceKey
}

export function resolveDeliverySourceFocus(
  focusedSourceKey: string | null,
  sources: Array<Pick<DeliveryInsights['scm']['sources'][number], 'kind' | 'name' | 'repo_path'>>
): string | null {
  return focusedSourceKey && sources.some((source) => deliverySourceKey(source) === focusedSourceKey) ? focusedSourceKey : null
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
    const sourceKey = deliverySourceKey(source)
    if (source.failing_change_requests > 0) {
      alerts.push({
        severity: 'critical',
        title: `${source.name} has failing change requests`,
        detail: `${source.failing_change_requests} failing change request(s) are blocking merge readiness.`,
        sourceKey,
      })
    }

    if (source.stale_change_requests > 0) {
      alerts.push({
        severity: 'warning',
        title: `${source.name} has stale change requests`,
        detail: `${source.stale_change_requests} stale change request(s) need follow-up.`,
        sourceKey,
      })
    }

    source.warnings?.forEach((warning) => {
      alerts.push({
        severity: 'warning',
        title: `${source.name} source warning`,
        detail: warning,
        sourceKey,
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
    const key = `${alert.severity}:${alert.title}:${alert.detail}:${alert.sourceKey ?? ''}`
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

export function deliverySourceKey(source: Pick<DeliveryInsights['scm']['sources'][number], 'kind' | 'name' | 'repo_path'>): string {
  return `${source.kind}:${source.name}:${source.repo_path}`
}
