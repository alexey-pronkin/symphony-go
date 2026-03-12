## 1. Runtime State Port

- [x] 1.1 Define persisted models and a runtime-state repository interface for retry and running-session records
- [x] 1.2 Add orchestrator tests for persisting retry transitions and restoring state on startup

## 2. Postgres Adapter

- [x] 2.1 Implement a GORM-backed Postgres runtime-state store
- [x] 2.2 Wire config/app startup for optional runtime-state persistence

## 3. Restore And Observability

- [x] 3.1 Restore retry/running state before the first orchestrator tick
- [x] 3.2 Surface degraded persistence state in snapshots and issue detail
- [x] 3.3 Add HTTP/server regression coverage for restored runtime state

## 4. Verification

- [x] 4.1 Run `cd arpego && go test ./...`
- [x] 4.2 Run `cd arpego && go build ./...`
- [x] 4.3 Update docs/specs for runtime-state durability
