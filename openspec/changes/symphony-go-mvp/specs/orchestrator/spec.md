## ADDED Requirements

### Requirement: Poll loop with configurable interval
The system SHALL run an immediate tick on startup, then repeat every `polling.interval_ms`. The interval SHALL be updated when config is reloaded without restart.

#### Scenario: Immediate tick on startup
- **WHEN** the service starts
- **THEN** a tick executes before the first interval elapses

#### Scenario: Interval update takes effect after reload
- **WHEN** `polling.interval_ms` changes via WORKFLOW.md reload
- **THEN** subsequent ticks use the new interval

### Requirement: Dispatch eligibility enforcement
The system SHALL dispatch an issue only when all eligibility conditions from SPEC.md Section 8.2 are satisfied: valid fields, active state, not claimed/running, global and per-state slot available, Todo blocker rule passes.

#### Scenario: Claimed issue is not redispatched
- **WHEN** an issue is already in the `claimed` set
- **THEN** the issue is skipped during dispatch

#### Scenario: Todo with non-terminal blocker is skipped
- **WHEN** a Todo issue has a blocker whose state is not terminal
- **THEN** the issue is not dispatched

#### Scenario: Global concurrency limit respected
- **WHEN** `running` count equals `max_concurrent_agents`
- **THEN** no further issues are dispatched this tick

### Requirement: Dispatch sort order
The system SHALL sort candidates by priority ascending (null last), then `created_at` oldest first, then `identifier` lexicographically.

#### Scenario: Lower priority number dispatched first
- **WHEN** two eligible issues have priority 1 and priority 3
- **THEN** priority 1 is dispatched first

### Requirement: Exponential retry backoff
The system SHALL retry failed workers with delay `min(10000 * 2^(attempt-1), max_retry_backoff_ms)`. Normal exits SHALL schedule a 1000ms continuation retry at attempt 1.

#### Scenario: First failure retries after 10 seconds
- **WHEN** a worker exits abnormally on attempt 1
- **THEN** retry is scheduled with ~10000ms delay

#### Scenario: Backoff capped at max_retry_backoff_ms
- **WHEN** computed delay exceeds `agent.max_retry_backoff_ms`
- **THEN** the delay is capped at `max_retry_backoff_ms`

#### Scenario: Normal exit schedules 1s continuation retry
- **WHEN** a worker exits normally
- **THEN** a retry is scheduled with 1000ms delay at attempt 1

### Requirement: Reconciliation on every tick
The system SHALL reconcile running issues every tick: stall detection first, then tracker state refresh. Terminal state stops worker and cleans workspace; non-active state stops worker without cleanup.

#### Scenario: Terminal state triggers workspace cleanup
- **WHEN** a running issue's Linear state becomes `Done`
- **THEN** the worker is terminated and the workspace directory is removed

#### Scenario: Non-active non-terminal stops worker only
- **WHEN** a running issue's state is neither active nor terminal
- **THEN** the worker is terminated and workspace is preserved

#### Scenario: Stall detection kills and retries
- **WHEN** elapsed time since last agent event exceeds `codex.stall_timeout_ms`
- **THEN** the worker is killed and a retry is scheduled

### Requirement: Startup terminal workspace cleanup
The system SHALL query for terminal-state issues at startup and remove their workspaces before the first tick.

#### Scenario: Terminal workspace removed at startup
- **WHEN** a workspace directory exists for a terminal Linear issue at startup
- **THEN** the workspace directory is removed before the first tick
