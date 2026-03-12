## Design

This slice persists the minimum orchestrator state needed to survive a restart
without changing the core scheduling model.

### Scope

- Persist retry queue entries.
- Persist running session metadata needed for observability and continuation.
- Restore persisted state during startup before tick processing begins.
- Keep persistence behind a narrow runtime-state port so storage stays
  replaceable.

### Storage Model

- Introduce a runtime-state repository interface owned by the orchestrator.
- Represent retry entries with issue identifier, attempt metadata,
  continuation flag, reason, and next-at timestamp.
- Represent running-session state with issue identifier, session identifiers,
  workspace path, started-at timestamp, retry attempt, and last known status.
- Default to Postgres for the first durable adapter, using GORM like the
  existing transactional task storage.

### Restore Behavior

- Load persisted retry and running records on startup before the orchestrator
  loop begins.
- Rehydrate snapshot/status surfaces from restored state immediately.
- Reconcile restored running entries conservatively: treat them as stale until
  the first reconcile pass confirms whether the worker/session is still active.

### Failure Handling

- Persistence failures should degrade observability and retry durability, but
  must not crash the orchestrator after startup.
- Startup restore failure should be surfaced clearly and fail fast so operators
  do not run with silently partial state.
