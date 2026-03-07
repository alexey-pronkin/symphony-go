## 1. Architecture And Persistence Ports

- [x] 1.1 Define hexagonal ports for task persistence, task queries, runtime event sinks, and metrics export
- [x] 1.2 Add failing backend tests for Postgres-backed task CRUD and DB-agnostic service contracts
- [x] 1.3 Introduce the Postgres adapter for transactional task storage

## 2. Analytics And Observability

- [x] 2.1 Add failing backend tests for ClickHouse-backed runtime event writes and issue debug/event queries
- [x] 2.2 Introduce the ClickHouse adapter for runtime events and observability reads
- [x] 2.3 Add Prometheus metrics endpoints and Docker Compose services for Prometheus/Grafana under `--profile monitoring`

## 3. Continuation And Debug Runtime

- [x] 3.1 Add failing tests for session thread reuse, in-process continuation turns, and full issue prompt context
- [x] 3.2 Implement multi-turn runner behavior on one live thread with issue refresh checks and continuation guidance
- [x] 3.3 Add recent event history and session/log references to runtime issue detail

## 4. Libretto Task Platform

- [x] 4.1 Add failing frontend tests for task list loading, task creation, task state updates, and degraded-observability handling
- [x] 4.2 Extend the typed client with task platform and observability endpoints
- [x] 4.3 Implement a simple task platform UI in Libretto that combines task management with the current runtime dashboard

## 5. Spec And Docs

- [x] 5.1 Update `SPEC.md` for database-agnostic task platform services, Postgres/ClickHouse adapters, and continuation/debug behavior
- [x] 5.2 Update README/docs for Docker stack usage, monitoring profile, and licensing tradeoffs
