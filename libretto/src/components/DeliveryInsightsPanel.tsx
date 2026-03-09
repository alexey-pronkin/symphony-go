import type { DeliveryInsights, DeliveryTrendReport } from '../lib/api'
import { deliveryObservabilityState, hasDeliveryWarnings, orderedDeliveryCards } from '../lib/delivery-insights'

type DeliveryInsightsPanelProps = {
  report: DeliveryInsights | null
  trends: DeliveryTrendReport | null
  loading: boolean
  trendsLoading: boolean
  error: string | null
  trendsError: string | null
}

export function DeliveryInsightsPanel({ report, trends, loading, trendsLoading, error, trendsError }: DeliveryInsightsPanelProps) {
  if (loading && !report) {
    return (
      <section className="panel delivery-panel">
        <div className="panel-heading">
          <div>
            <p className="panel-kicker">Delivery metrics</p>
            <h2>Operational signals</h2>
          </div>
        </div>
        <p className="status-message">Loading delivery metrics…</p>
      </section>
    )
  }

  if (error && !report) {
    return (
      <section className="panel delivery-panel">
        <div className="panel-heading">
          <div>
            <p className="panel-kicker">Delivery metrics</p>
            <h2>Operational signals</h2>
          </div>
        </div>
        <p className="status-message">{error}</p>
      </section>
    )
  }

  if (!report) {
    return null
  }

  const cards = orderedDeliveryCards(report)
  const status = deliveryObservabilityState(report, error)

  return (
    <section className={`panel delivery-panel delivery-panel-${status}`}>
      <div className="panel-heading">
        <div>
          <p className="panel-kicker">Delivery metrics</p>
          <h2>Operational signals</h2>
        </div>
      </div>

      <div className="delivery-card-grid">
        {cards.map((card) => (
          <article className={`delivery-card delivery-${card.status}`} key={card.key}>
            <div className="delivery-card-top">
              <span>{card.label}</span>
              <strong>{card.score}</strong>
            </div>
            <p>{card.detail}</p>
          </article>
        ))}
      </div>

      <div className="delivery-trend-grid">
        {metricTrendCard('Delivery health', trends, trendsLoading, trendsError, (point) => point.delivery_health)}
        {metricTrendCard('Flow efficiency', trends, trendsLoading, trendsError, (point) => point.flow_efficiency)}
        {metricTrendCard('Merge readiness', trends, trendsLoading, trendsError, (point) => point.merge_readiness)}
        {metricTrendCard('Predictability', trends, trendsLoading, trendsError, (point) => point.predictability)}
      </div>

      <div className="delivery-rollup-grid">
        <article className="delivery-breakdown">
          <h3>Trend rollups</h3>
          <dl>
            <div>
              <dt>Health average</dt>
              <dd>{trends?.rollups.health_average ?? '—'}</dd>
            </div>
            <div>
              <dt>Health delta</dt>
              <dd>{formatSigned(trends?.rollups.health_delta)}</dd>
            </div>
            <div>
              <dt>Health slope</dt>
              <dd>{formatSigned(trends?.rollups.health_slope)}</dd>
            </div>
            <div>
              <dt>Warning pressure</dt>
              <dd>{trends ? trends.rollups.warning_pressure.toFixed(2) : '—'}</dd>
            </div>
          </dl>
        </article>

        <article className="delivery-breakdown">
          <h3>Alerts</h3>
          <div className="delivery-alert-list">
            {trends?.alerts?.length ? (
              trends.alerts.map((alert) => (
                <div className={`delivery-alert delivery-${alert.severity}`} key={alert.key}>
                  <strong>{alert.label}</strong>
                  <p>{alert.detail}</p>
                </div>
              ))
            ) : (
              <p className="status-message">No trend alerts in the current window.</p>
            )}
          </div>
        </article>
      </div>

      {hasDeliveryWarnings(report) ? (
        <div className="delivery-warning-list">
          {report.warnings.map((warning) => (
            <p key={warning}>{warning}</p>
          ))}
        </div>
      ) : null}

      <div className="delivery-breakdown-grid">
        <article className="delivery-breakdown">
          <h3>Agile</h3>
          <dl>
            <div>
              <dt>Throughput</dt>
              <dd>{report.tracker.agile.throughput_last_window}</dd>
            </div>
            <div>
              <dt>Completion ratio</dt>
              <dd>{percent(report.tracker.agile.completion_ratio)}</dd>
            </div>
            <div>
              <dt>Review load</dt>
              <dd>{percent(report.tracker.agile.review_load)}</dd>
            </div>
          </dl>
        </article>

        <article className="delivery-breakdown">
          <h3>Kanban</h3>
          <dl>
            <div>
              <dt>WIP</dt>
              <dd>{report.tracker.kanban.wip_count}</dd>
            </div>
            <div>
              <dt>Blocked ratio</dt>
              <dd>{percent(report.tracker.kanban.blocked_ratio)}</dd>
            </div>
            <div>
              <dt>Aging work</dt>
              <dd>{percent(report.tracker.kanban.aging_work_ratio)}</dd>
            </div>
          </dl>
        </article>

        <article className="delivery-breakdown">
          <h3>Gitflow</h3>
          <dl>
            <div>
              <dt>Sources</dt>
              <dd>{report.scm.active_sources}</dd>
            </div>
            <div>
              <dt>Unmerged</dt>
              <dd>{report.scm.totals.unmerged_branches}</dd>
            </div>
            <div>
              <dt>Drift commits</dt>
              <dd>{report.scm.totals.drift_commits}</dd>
            </div>
            <div>
              <dt>Open changes</dt>
              <dd>{report.scm.totals.open_change_requests}</dd>
            </div>
            <div>
              <dt>Failing checks</dt>
              <dd>{report.scm.totals.failing_change_requests}</dd>
            </div>
          </dl>
        </article>
      </div>

      <div className="delivery-source-list">
        {report.scm.sources.map((source) => (
          <article className="delivery-source" key={`${source.kind}-${source.name}-${source.repo_path}`}>
            <header>
              <strong>{source.name}</strong>
              <span>
                {source.kind} · {source.main_branch}
              </span>
            </header>
            <p>{source.repo_path}</p>
            <div className="delivery-source-metrics">
              <span>{source.branches} branches</span>
              <span>{source.unmerged_branches} unmerged</span>
              <span>{source.stale_branches} stale</span>
              <span>{source.open_change_requests} open changes</span>
              <span>{source.failing_change_requests} failing</span>
              <span>{source.merge_readiness} readiness</span>
            </div>
            {source.warnings?.length ? (
              <div className="delivery-source-warnings">
                {source.warnings.map((warning) => (
                  <p key={warning}>{warning}</p>
                ))}
              </div>
            ) : null}
          </article>
        ))}
      </div>
    </section>
  )
}

function percent(value: number): string {
  return `${Math.round(value * 100)}%`
}

function formatSigned(value: number | undefined): string {
  if (value == null) {
    return '—'
  }
  if (Number.isInteger(value)) {
    return `${value >= 0 ? '+' : ''}${value}`
  }
  return `${value >= 0 ? '+' : ''}${value.toFixed(2)}`
}

function metricTrendCard(
  label: string,
  trends: DeliveryTrendReport | null,
  loading: boolean,
  error: string | null,
  select: (point: DeliveryTrendReport['points'][number]) => number
) {
  const points = trends?.points ?? []
  const values = points.map(select)
  const current = values.at(-1)
  const previous = values.length > 1 ? values.at(-2) : undefined
  const delta = current != null && previous != null ? current - previous : null

  return (
    <article className="delivery-trend-card" key={label}>
      <div className="delivery-trend-top">
        <span>{label}</span>
        <strong>{current ?? '—'}</strong>
      </div>
      <div className="delivery-sparkline" aria-hidden="true">
        {values.length > 0 ? (
          values.map((value, index) => (
            <span
              className="delivery-spark"
              key={`${label}-${index}`}
              style={{ height: `${Math.max(16, Math.round((value / 100) * 64))}px` }}
            />
          ))
        ) : (
          <span className="delivery-trend-empty">No trend points yet</span>
        )}
      </div>
      <p>
        {loading ? 'Loading trend history…' : null}
        {!loading && error ? error : null}
        {!loading && !error && delta != null ? `${delta >= 0 ? '+' : ''}${delta} from previous sample` : null}
        {!loading && !error && delta == null && points.length > 0 ? `${points.length} samples in ${trends?.window ?? 'window'}` : null}
        {!loading && !error && points.length === 0 ? 'No historical samples captured yet' : null}
      </p>
    </article>
  )
}
