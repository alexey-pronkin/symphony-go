import type { DeliveryInsights, DeliveryMetricCard } from './api'

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
