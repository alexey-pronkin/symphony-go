## ADDED Requirements

### Requirement: Runtime State MUST Survive Restarts

The orchestrator MUST persist retry queue entries and running-session metadata
so state can be restored after a process restart.

#### Scenario: Retry queue is restored on startup

- **GIVEN** persisted retry queue entries exist from a prior run
- **WHEN** the process starts
- **THEN** the orchestrator restores those entries before the first dispatch
  cycle
- **AND** the runtime snapshot reports the restored retry queue

#### Scenario: Running session metadata is restored on startup

- **GIVEN** persisted running-session metadata exists from a prior run
- **WHEN** the process starts
- **THEN** issue detail and runtime snapshot surfaces expose the restored
  metadata before reconciliation completes

### Requirement: Runtime State Persistence MUST Degrade Safely

If runtime-state persistence fails after startup, the orchestrator MUST keep
running while surfacing the degraded durability condition.

#### Scenario: Persisting a retry update fails

- **WHEN** the orchestrator cannot write an updated retry entry
- **THEN** scheduling continues in memory
- **AND** operators can observe that runtime-state durability is degraded
