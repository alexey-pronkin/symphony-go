import { startTransition, useEffect, useEffectEvent, useState } from 'react'
import './App.css'
import { DeliveryInsightsPanel } from './components/DeliveryInsightsPanel'
import {
  createSymphonyClient,
  type CreateTaskInput,
  type DeliveryInsights,
  type IssueDetail,
  type RuntimeIssue,
  type RuntimeState,
  type TaskListResponse,
  type TaskRecord,
} from './lib/api'

const client = createSymphonyClient()
const POLL_INTERVAL_MS = 15000
const TASK_COLUMNS = ['Todo', 'In Progress', 'Review', 'Done']

function App() {
  const [state, setState] = useState<RuntimeState | null>(null)
  const [tasks, setTasks] = useState<TaskListResponse | null>(null)
  const [delivery, setDelivery] = useState<DeliveryInsights | null>(null)
  const [stateError, setStateError] = useState<string | null>(null)
  const [tasksError, setTasksError] = useState<string | null>(null)
  const [deliveryError, setDeliveryError] = useState<string | null>(null)
  const [loadingState, setLoadingState] = useState(true)
  const [loadingTasks, setLoadingTasks] = useState(true)
  const [loadingDelivery, setLoadingDelivery] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [creatingTask, setCreatingTask] = useState(false)
  const [selectedIssue, setSelectedIssue] = useState<string | null>(null)
  const [detail, setDetail] = useState<IssueDetail | null>(null)
  const [detailError, setDetailError] = useState<string | null>(null)
  const [loadingDetail, setLoadingDetail] = useState(false)
  const [draft, setDraft] = useState<CreateTaskInput>({
    title: '',
    description: '',
    state: 'Todo',
  })

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
        const available = allRuntimeIssues(nextState)
        if (available.length === 0 && !currentSelectedIssue) {
          setDetail(null)
          setDetailError(null)
          return
        }
        if (!currentSelectedIssue && available.length > 0) {
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

  async function performLoadTasks(mode: 'initial' | 'refresh' = 'refresh') {
    if (mode === 'initial') {
      setLoadingTasks(true)
    }

    try {
      const nextTasks = await client.fetchTasks()
      startTransition(() => {
        setTasks(nextTasks)
        setTasksError(null)
        if (!selectedIssue && nextTasks.tasks.length > 0) {
          setSelectedIssue(nextTasks.tasks[0].identifier)
        }
      })
    } catch (error) {
      setTasksError(asMessage(error))
    } finally {
      setLoadingTasks(false)
    }
  }

  async function performLoadDelivery(mode: 'initial' | 'refresh' = 'refresh') {
    if (mode === 'initial') {
      setLoadingDelivery(true)
    }

    try {
      const nextDelivery = await client.fetchDeliveryInsights()
      startTransition(() => {
        setDelivery(nextDelivery)
        setDeliveryError(null)
      })
    } catch (error) {
      setDeliveryError(asMessage(error))
    } finally {
      setLoadingDelivery(false)
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

  const loadRuntimeEffect = useEffectEvent((mode: 'initial' | 'refresh' = 'refresh') => {
    void performLoadState(mode)
    void performLoadTasks(mode)
    void performLoadDelivery(mode)
  })

  const loadDetailEffect = useEffectEvent((identifier: string) => {
    void performLoadDetail(identifier)
  })

  useEffect(() => {
    loadRuntimeEffect('initial')
    const timer = window.setInterval(() => {
      loadRuntimeEffect()
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
    await performLoadTasks('refresh')
    await performLoadDelivery('refresh')
    if (selectedIssue) {
      await performLoadDetail(selectedIssue)
    }
  }

  async function handleCreateTask() {
    if (!draft.title?.trim()) {
      setTasksError('Task title is required')
      return
    }
    setCreatingTask(true)
    try {
      const created = await client.createTask({
        ...draft,
        title: draft.title.trim(),
        description: draft.description?.trim() ? draft.description.trim() : undefined,
      })
      setDraft({ title: '', description: '', state: draft.state ?? 'Todo' })
      setSelectedIssue(created.identifier)
      setTasksError(null)
      await performLoadTasks('refresh')
      await performLoadState('refresh', created.identifier)
      await performLoadDetail(created.identifier)
    } catch (error) {
      setTasksError(asMessage(error))
    } finally {
      setCreatingTask(false)
    }
  }

  async function handleMoveTask(task: TaskRecord, stateName: string) {
    try {
      await client.updateTask(task.identifier, { state: stateName })
      await performLoadTasks('refresh')
      await performLoadState('refresh', selectedIssue)
      if (selectedIssue === task.identifier) {
        await performLoadDetail(task.identifier)
      }
    } catch (error) {
      setTasksError(asMessage(error))
    }
  }

  const summary = state
    ? [
        { label: 'Running', value: state.counts.running.toString(), tone: 'primary' },
        { label: 'Retrying', value: state.counts.retrying.toString(), tone: 'secondary' },
        { label: 'Tasks', value: String(tasks?.counts.total ?? 0), tone: 'accent' },
        { label: 'Runtime Seconds', value: Math.round(state.codex_totals.seconds_running).toString(), tone: 'muted' },
      ]
    : []

  return (
    <main className="app-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Symphony Platform</p>
          <h1>Task platform and runtime control surface for Symphony operators.</h1>
          <p className="hero-copy">
            Libretto now combines task intake, task state updates, live runtime state, and issue debugging in one workspace.
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
          <span className="poll-note">Runtime poll every 15s</span>
        </div>
      </section>

      {loadingState && !state ? (
        <section className="panel single-panel">
          <p className="status-message">Loading Symphony runtime…</p>
        </section>
      ) : null}

      {state ? (
        <>
          <DeliveryInsightsPanel report={delivery} loading={loadingDelivery} error={deliveryError} />

          <section className="summary-grid">
            {summary.map((item) => (
              <article className={`summary-card tone-${item.tone}`} key={item.label}>
                <p>{item.label}</p>
                <strong>{item.value}</strong>
              </article>
            ))}
          </section>

          <section className="workspace-grid">
            <div className="panel platform-panel">
              <div className="panel-heading">
                <div>
                  <p className="panel-kicker">Task platform</p>
                  <h2>Work queue</h2>
                </div>
                {loadingTasks ? <span className="poll-note">Loading tasks…</span> : null}
              </div>

              <form
                className="task-form"
                onSubmit={(event) => {
                  event.preventDefault()
                  void handleCreateTask()
                }}
              >
                <label>
                  <span>Title</span>
                  <input
                    value={draft.title ?? ''}
                    onChange={(event) => {
                      setDraft((current) => ({ ...current, title: event.target.value }))
                    }}
                    placeholder="Implement orchestration slice"
                  />
                </label>
                <label>
                  <span>Description</span>
                  <textarea
                    rows={3}
                    value={draft.description ?? ''}
                    onChange={(event) => {
                      setDraft((current) => ({ ...current, description: event.target.value }))
                    }}
                    placeholder="Context, acceptance notes, or blockers"
                  />
                </label>
                <div className="task-form-row">
                  <label>
                    <span>State</span>
                    <select
                      value={draft.state ?? 'Todo'}
                      onChange={(event) => {
                        setDraft((current) => ({ ...current, state: event.target.value }))
                      }}
                    >
                      {TASK_COLUMNS.map((stateName) => (
                        <option key={stateName} value={stateName}>
                          {stateName}
                        </option>
                      ))}
                    </select>
                  </label>
                  <button className="refresh-button task-submit" type="submit" disabled={creatingTask}>
                    {creatingTask ? 'Creating…' : 'Add task'}
                  </button>
                </div>
              </form>

              {tasksError ? (
                <div className="task-platform-state error-panel">
                  <div>
                    <h2>Task platform unavailable</h2>
                    <p>{tasksError}</p>
                  </div>
                </div>
              ) : null}

              {tasks ? (
                <div className="task-board">
                  {TASK_COLUMNS.map((stateName) => (
                    <TaskColumn
                      key={stateName}
                      stateName={stateName}
                      tasks={tasks.tasks.filter((task) => task.state === stateName)}
                      selectedIssue={selectedIssue}
                      onSelect={setSelectedIssue}
                      onMove={handleMoveTask}
                    />
                  ))}
                </div>
              ) : null}
            </div>

            <div className="runtime-stack">
              <div className="panel queue-panel">
                <div className="panel-heading">
                  <div>
                    <p className="panel-kicker">Live runtime</p>
                    <h2>Active sessions</h2>
                  </div>
                </div>
                {stateError ? <p className="status-message">{stateError}</p> : null}
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

                {!selectedIssue ? <p className="status-message">Select a task or runtime issue to inspect detail.</p> : null}
                {loadingDetail ? <p className="status-message">Loading issue detail…</p> : null}
                {detailError ? <p className="status-message">{detailError}</p> : null}

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
                        <dt>Session</dt>
                        <dd>{detail.running?.session_id ?? 'Not running'}</dd>
                      </div>
                      <div>
                        <dt>State</dt>
                        <dd>{detail.running?.state ?? detail.status}</dd>
                      </div>
                      <div>
                        <dt>Last event</dt>
                        <dd>{detail.running?.last_event ?? 'None'}</dd>
                      </div>
                      <div>
                        <dt>Last error</dt>
                        <dd>{detail.last_error ?? 'None'}</dd>
                      </div>
                    </dl>

                    {detail.logs.codex_session_logs.length > 0 ? (
                      <div className="token-block">
                        <h4>Session logs</h4>
                        {detail.logs.codex_session_logs.map((log) => (
                          <p key={`${log.label}-${log.path}`}>
                            {log.label}: {log.path}
                          </p>
                        ))}
                      </div>
                    ) : null}

                    {detail.recent_events.length > 0 ? (
                      <div className="event-block">
                        <h4>Recent events</h4>
                        <div className="event-list">
                          {detail.recent_events.map((event) => (
                            <article key={`${event.at}-${event.event}`} className="event-row">
                              <strong>{event.event}</strong>
                              <span>{formatDate(event.at)}</span>
                              <p>{event.message}</p>
                            </article>
                          ))}
                        </div>
                      </div>
                    ) : null}
                  </article>
                ) : null}
              </div>
            </div>
          </section>
        </>
      ) : null}
    </main>
  )
}

type TaskColumnProps = {
  stateName: string
  tasks: TaskRecord[]
  selectedIssue: string | null
  onSelect: (identifier: string) => void
  onMove: (task: TaskRecord, stateName: string) => Promise<void>
}

function TaskColumn({ stateName, tasks, selectedIssue, onSelect, onMove }: TaskColumnProps) {
  return (
    <section className="task-column">
      <header>
        <p className="panel-kicker">{stateName}</p>
        <h3>{tasks.length}</h3>
      </header>
      <div className="task-column-body">
        {tasks.length === 0 ? <p className="status-message">No tasks</p> : null}
        {tasks.map((task) => (
          <article
            className={`task-card ${selectedIssue === task.identifier ? 'task-card-selected' : ''}`}
            key={task.id}
            onClick={() => {
              onSelect(task.identifier)
            }}
          >
            <div className="task-card-copy">
              <strong>{task.identifier}</strong>
              <h4>{task.title}</h4>
              {task.description ? <p>{task.description}</p> : null}
            </div>
            <div className="task-card-actions">
              {TASK_COLUMNS.filter((candidate) => candidate !== task.state)
                .slice(0, 2)
                .map((candidate) => (
                  <button
                    className="ghost-button task-move"
                    key={`${task.id}-${candidate}`}
                    onClick={(event) => {
                      event.stopPropagation()
                      void onMove(task, candidate)
                    }}
                    type="button"
                  >
                    {candidate}
                  </button>
                ))}
            </div>
          </article>
        ))}
      </div>
    </section>
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

function allRuntimeIssues(state: RuntimeState): RuntimeIssue[] {
  return [...state.running, ...state.retrying]
}

function formatDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

function asMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }
  return 'Unknown error'
}

export default App
