import { startTransition, useEffect, useEffectEvent, useState } from 'react'
import './App.css'
import { createSymphonyClient, type IssueDetail, type RuntimeIssue, type RuntimeState } from './lib/api'

const client = createSymphonyClient()
const POLL_INTERVAL_MS = 15000

function App() {
  const [state, setState] = useState<RuntimeState | null>(null)
  const [stateError, setStateError] = useState<string | null>(null)
  const [loadingState, setLoadingState] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [selectedIssue, setSelectedIssue] = useState<string | null>(null)
  const [detail, setDetail] = useState<IssueDetail | null>(null)
  const [detailError, setDetailError] = useState<string | null>(null)
  const [loadingDetail, setLoadingDetail] = useState(false)

  async function performLoadState(mode: 'initial' | 'refresh' = 'refresh', currentSelectedIssue = selectedIssue) {
    if (mode === 'initial') {
      setLoadingState(true)
    } else {
      setRefreshing(true)
    }

    try {
      const nextState = await client.fetchState()
      startTransition(() => {
        setState(nextState)
        setStateError(null)
        const available = allIssues(nextState)
        if (available.length === 0) {
          setSelectedIssue(null)
          setDetail(null)
          setDetailError(null)
          return
        }
        if (!currentSelectedIssue || !available.some((issue) => issue.issue_identifier === currentSelectedIssue)) {
          setSelectedIssue(available[0].issue_identifier)
        }
      })
    } catch (error) {
      setStateError(asMessage(error))
    } finally {
      setLoadingState(false)
      setRefreshing(false)
    }
  }

  async function performLoadDetail(identifier: string) {
    setLoadingDetail(true)
    try {
      const nextDetail = await client.fetchIssue(identifier)
      startTransition(() => {
        setDetail(nextDetail)
        setDetailError(null)
      })
    } catch (error) {
      setDetail(null)
      setDetailError(asMessage(error))
    } finally {
      setLoadingDetail(false)
    }
  }

  const loadStateEffect = useEffectEvent((mode: 'initial' | 'refresh' = 'refresh') => {
    void performLoadState(mode)
  })

  const loadDetailEffect = useEffectEvent((identifier: string) => {
    void performLoadDetail(identifier)
  })

  useEffect(() => {
    loadStateEffect('initial')
    const timer = window.setInterval(() => {
      loadStateEffect()
    }, POLL_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [])

  useEffect(() => {
    if (!selectedIssue) {
      return
    }
    loadDetailEffect(selectedIssue)
  }, [selectedIssue])

  async function handleRefresh() {
    setRefreshing(true)
    try {
      await client.refresh()
    } catch (error) {
      setStateError(asMessage(error))
    }
    await performLoadState('refresh', selectedIssue)
    if (selectedIssue) {
      await performLoadDetail(selectedIssue)
    }
  }

  const summary = state
    ? [
        {
          label: 'Running',
          value: state.counts.running.toString(),
          tone: 'primary',
        },
        {
          label: 'Retrying',
          value: state.counts.retrying.toString(),
          tone: 'secondary',
        },
        {
          label: 'Total Tokens',
          value: state.codex_totals.total_tokens.toLocaleString(),
          tone: 'accent',
        },
        {
          label: 'Runtime Seconds',
          value: Math.round(state.codex_totals.seconds_running).toString(),
          tone: 'muted',
        },
      ]
    : []

  return (
    <main className="app-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Symphony Runtime</p>
          <h1>Operator dashboard for active work, retries, and issue detail.</h1>
          <p className="hero-copy">
            Libretto reads the Arpego runtime API and keeps the current queue visible without dropping operators into raw JSON.
          </p>
        </div>
        <div className="hero-actions">
          <button
            className="refresh-button"
            onClick={() => {
              void handleRefresh()
            }}
            disabled={refreshing}
          >
            {refreshing ? 'Refreshing…' : 'Refresh now'}
          </button>
          <span className="poll-note">Auto-refresh every 15s</span>
        </div>
      </section>

      {loadingState && !state ? (
        <section className="panel">
          <p className="status-message">Loading runtime state…</p>
        </section>
      ) : null}

      {stateError ? (
        <section className="panel error-panel">
          <div>
            <h2>State request failed</h2>
            <p>{stateError}</p>
          </div>
          <button
            className="ghost-button"
            onClick={() => {
              void performLoadState('initial', selectedIssue)
            }}
          >
            Retry state load
          </button>
        </section>
      ) : null}

      {state ? (
        <>
          <section className="summary-grid">
            {summary.map((item) => (
              <article className={`summary-card tone-${item.tone}`} key={item.label}>
                <p>{item.label}</p>
                <strong>{item.value}</strong>
              </article>
            ))}
          </section>

          <section className="content-grid">
            <div className="panel queue-panel">
              <div className="panel-heading">
                <div>
                  <p className="panel-kicker">Live runtime</p>
                  <h2>Running sessions</h2>
                </div>
              </div>
              <IssueList
                items={state.running}
                emptyMessage="No active sessions right now."
                selectedIssue={selectedIssue}
                onSelect={setSelectedIssue}
              />

              <div className="queue-divider" />

              <div className="panel-heading">
                <div>
                  <p className="panel-kicker">Retry queue</p>
                  <h2>Pending retries</h2>
                </div>
              </div>
              <IssueList
                items={state.retrying}
                emptyMessage="No queued retries."
                selectedIssue={selectedIssue}
                onSelect={setSelectedIssue}
              />
            </div>

            <div className="panel detail-panel">
              <div className="panel-heading">
                <div>
                  <p className="panel-kicker">Selected issue</p>
                  <h2>Detail</h2>
                </div>
                {detail?.status ? <span className={`status-chip status-${detail.status}`}>{detail.status}</span> : null}
              </div>

              {!selectedIssue ? (
                <p className="status-message">Select a running or retrying issue to inspect workspace and attempt details.</p>
              ) : null}

              {loadingDetail ? <p className="status-message">Loading issue detail…</p> : null}

              {detailError ? (
                <div className="detail-error">
                  <p>{detailError}</p>
                  <button
                    className="ghost-button"
                    onClick={() => {
                      if (selectedIssue) {
                        void performLoadDetail(selectedIssue)
                      }
                    }}
                  >
                    Retry detail load
                  </button>
                </div>
              ) : null}

              {detail ? (
                <article className="detail-stack">
                  <header>
                    <h3>{detail.issue_identifier}</h3>
                    <p>{detail.issue_id}</p>
                  </header>

                  <dl className="detail-grid">
                    <div>
                      <dt>Workspace</dt>
                      <dd>{detail.workspace.path}</dd>
                    </div>
                    <div>
                      <dt>Restart count</dt>
                      <dd>{detail.attempts.restart_count}</dd>
                    </div>
                    <div>
                      <dt>Current retry</dt>
                      <dd>{detail.attempts.current_retry_attempt}</dd>
                    </div>
                    <div>
                      <dt>Session</dt>
                      <dd>{detail.running?.session_id ?? 'Not running'}</dd>
                    </div>
                    <div>
                      <dt>State</dt>
                      <dd>{detail.running?.state ?? detail.status}</dd>
                    </div>
                    <div>
                      <dt>Last error</dt>
                      <dd>{detail.last_error ?? 'None'}</dd>
                    </div>
                  </dl>

                  {detail.running ? (
                    <div className="token-block">
                      <h4>Token usage</h4>
                      <p>
                        {detail.running.tokens.total_tokens.toLocaleString()} total tokens,{' '}
                        {detail.running.tokens.input_tokens.toLocaleString()} input, {detail.running.tokens.output_tokens.toLocaleString()}{' '}
                        output.
                      </p>
                    </div>
                  ) : null}
                </article>
              ) : null}
            </div>
          </section>
        </>
      ) : null}
    </main>
  )
}

type IssueListProps = {
  items: RuntimeIssue[]
  emptyMessage: string
  selectedIssue: string | null
  onSelect: (identifier: string) => void
}

function IssueList({ items, emptyMessage, selectedIssue, onSelect }: IssueListProps) {
  if (items.length === 0) {
    return <p className="status-message">{emptyMessage}</p>
  }

  return (
    <div className="issue-list">
      {items.map((item) => (
        <button
          className={`issue-row ${selectedIssue === item.issue_identifier ? 'issue-row-selected' : ''}`}
          key={`${item.kind}-${item.issue_id}`}
          onClick={() => {
            onSelect(item.issue_identifier)
          }}
        >
          <div>
            <strong>{item.issue_identifier}</strong>
            <p>{item.kind === 'running' ? item.state : `Retry ${item.attempt}`}</p>
          </div>
          <div className="issue-row-meta">
            <span>{item.kind === 'running' ? item.session_id || 'pending' : item.error}</span>
          </div>
        </button>
      ))}
    </div>
  )
}

function allIssues(state: RuntimeState): RuntimeIssue[] {
  return [...state.running, ...state.retrying]
}

function asMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }
  return 'Unknown error'
}

export default App
