## Overview

This change pivots from a file-backed tracker toward a scalable task platform and observability base. The system keeps a hexagonal architecture so the domain remains database-agnostic, uses PostgreSQL as the default transactional adapter, uses ClickHouse as the analytics/event adapter, adds a Docker monitoring profile with Prometheus and Grafana, and still closes the three runtime gaps: in-process continuation turns, full issue prompt context, and richer debug detail.

## Architecture Direction

### Hexagonal Service Boundaries

- Keep the core domain independent from any single database or tracker.
- Define ports for:
  - task repository
  - task query/read model
  - runtime event sink
  - metrics exporter
  - agent session runner
- Implement adapters for:
  - PostgreSQL task repository
  - ClickHouse runtime event sink and analytics query path
  - optional legacy/external tracker adapters

### Service Topology

The target topology is service-oriented, but the first implementation should keep the codebase deployable without forcing premature operational fragmentation.

- `task-platform-api`
  - owns task CRUD, state transitions, and query APIs
  - writes transactional state to PostgreSQL
- `orchestrator-worker`
  - polls eligible tasks through ports
  - manages workspaces and agent sessions
  - emits runtime events to the analytics port
- `observability-api`
  - serves runtime/debug queries optimized for Libretto
  - reads operational data from PostgreSQL and ClickHouse as appropriate
- `libretto`
  - React UI for task management and runtime visibility

Implementation note:

- Keep these boundaries explicit in code now, even if the first delivery still runs as one deployable process with multiple modules. That preserves the path to later extraction into separate services.

## Data Stores

### PostgreSQL

Use PostgreSQL as the source of truth for:

- tasks
- task state transitions
- workflow metadata
- service configuration metadata that must be durable

Rationale:

- PostgreSQL uses the PostgreSQL License, a BSD-like permissive license according to the official project site.
- It is the transactional store, not the analytics sink.

### ClickHouse

Use ClickHouse for:

- runtime event retention
- high-volume session and orchestration telemetry
- analytics-oriented UI queries
- future logs/traces/metrics correlation if needed

Rationale:

- ClickHouse publishes ClickHouse under Apache-2.0 in its official GitHub repository.
- It is a strong fit for event-heavy observability and operational analytics.

### DB-Agnostic Rule

- The domain layer must not depend on Postgres- or ClickHouse-specific packages.
- SQL schema and query specialization stays inside adapters.
- The application layer talks to repository/query ports, not directly to drivers.

## Monitoring Stack

### Prometheus And Grafana

- Add Prometheus scraping for service metrics endpoints.
- Add Grafana dashboards for API latency, orchestrator throughput, retries, task counts, and agent runtime metrics.
- Ship both under a Docker Compose `monitoring` profile so the default local stack stays lighter.

License note:

- Prometheus is Apache-2.0 in the official `prometheus/prometheus` repository.
- Grafana OSS is AGPLv3 according to Grafana Labs licensing docs. That is still open source, but it is not permissive. Treat this as an explicit tradeoff.

## Task Platform APIs

- `GET /api/v1/tasks`
- `POST /api/v1/tasks`
- `PATCH /api/v1/tasks/{issue_identifier}`
- `GET /api/v1/state`
- `GET /api/v1/{issue_identifier}`
- `POST /api/v1/refresh`

The task APIs should be defined against service DTOs, not tracker-specific payloads.

## Continuation Turns

### Runner Flow

- Split session behavior into:
  - startup handshake
  - start turn
  - stream turn result
- Reuse one app-server process and one `threadId` for multiple turns.
- After a successful turn:
  - re-fetch the issue through a callback provided by the orchestrator
  - stop in-process continuation if the issue is no longer active
  - otherwise start another turn on the same thread with continuation guidance
- Continue until:
  - issue is no longer active
  - `agent.max_turns` is reached
  - a turn fails, times out, stalls, or requests input

### Continuation Prompt

- First turn uses the rendered workflow prompt.
- Later turns send a short fixed continuation message that assumes prior context already exists in the thread history.
- The worker result remains a normal completion so the orchestrator can still schedule the existing short continuation retry after the in-process turn loop ends.

## Full Prompt Context

- Convert the full normalized `tracker.Issue` into the template input.
- Include optional fields as `null` when absent and preserve lists/maps for labels and blockers.
- Preserve `attempt` semantics for retries and continuation retries.

## Runtime Debug Details

### In-Memory Data

- Keep a bounded recent-event ring on each running entry.
- Include method, timestamp, and a concise message summary.
- Track the session log path for each running entry.

### Session Logging

- Create a small session log file under the workspace, e.g. `.symphony/session.jsonl`.
- Append outbound protocol requests and inbound protocol messages as JSON lines for debugging.
- Expose the latest log path in issue detail without making it part of orchestrator correctness.

### Issue Detail API

- Extend `GET /api/v1/{issue_identifier}` with:
  - `recent_events`
  - `logs.codex_session_logs`
  - additional tracked session metadata where available

## Risks

- Immediate microservice decomposition would slow delivery if done before ports/contracts exist, so code boundaries matter more than process count in the first pass.
- Grafana OSS is AGPLv3, which may be acceptable or unacceptable depending on distribution plans.
- Continuation turns add more session-state complexity, so tests need to cover thread reuse and max-turn behavior.
- Libretto should remain usable when task APIs or analytics backends are temporarily unavailable.
