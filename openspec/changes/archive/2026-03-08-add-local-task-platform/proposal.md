## Why

Symphony needs its own task platform and observability stack, but a file-backed tracker will not hold up if the system is expected to scale or evolve into multiple services. The next change should establish database-agnostic service boundaries, use an OSS-friendly transactional store, add an analytics/event store, and close the current runtime gaps around continuation turns, prompt context, and debugging detail.

## What Changes

- Introduce a database-agnostic task platform architecture using hexagonal ports/adapters.
- Use PostgreSQL as the primary transactional store for tasks, workflow state, and service metadata.
- Use ClickHouse as an optional analytics/telemetry store for runtime events and observability queries.
- Add Docker Compose services for PostgreSQL and ClickHouse plus a `monitoring` profile with Prometheus and Grafana.
- Extend Libretto into a Symphony task platform UI backed by service APIs rather than tracker-specific assumptions.
- Reuse a live Codex thread for in-process continuation turns instead of ending the worker after the first completed turn.
- Pass the full normalized issue model into prompt rendering for first turns and retries.
- Enrich issue detail responses with recent event history and log/session references.

## Capabilities

### New Capabilities
- `task-platform-services`: Database-agnostic task platform services with Postgres and ClickHouse adapters.
- `monitoring-stack`: Optional Prometheus and Grafana monitoring profile for local operations.
- `agent-continuation-turns`: Multi-turn worker execution on one live Codex thread with continuation guidance and full issue prompt context.
- `runtime-debug-details`: Richer issue detail payloads with recent events and session log references for debugging active runs.

### Modified Capabilities

## Impact

- `SPEC.md`
- `arpego/internal/*` architecture and service boundaries
- `libretto/src`
- Docker Compose and monitoring configuration
- Go and React tests for persistence adapters, service APIs, continuation behavior, and observability integration
