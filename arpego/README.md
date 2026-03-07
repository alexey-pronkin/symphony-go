# Arpego

`arpego/` is the primary implementation root for Symphony.

## Role

- Go backend and orchestration engine
- Default target for new backend work
- Source of truth for current runtime behavior once features move out of the legacy reference tree

## Notes For Agents

- Follow the repo-level guidance in [`../AGENTS.md`](/Users/pav/Documents/git/github/symphony-go/AGENTS.md).
- Treat `elixir/` as reference material only unless the task explicitly targets it.
- Keep code efficient, narrow in scope, and easy to retrieve via symbol-based navigation.

## Starter Command

```bash
cd arpego
go run ./cmd/arpego
```

## Build

```bash
cd arpego
go build ./...
```

## Run

Arpego expects a `WORKFLOW.md` file with YAML front matter plus the prompt body.

```bash
cd arpego
go run ./cmd/arpego ./WORKFLOW.md
```

CLI flags:

- `--port <n>` overrides `server.port` from `WORKFLOW.md`
- positional `[workflow-path]` overrides the default `./WORKFLOW.md`

The repo includes a sample [`WORKFLOW.md`](/Users/pav/Documents/git/github/symphony-go/arpego/WORKFLOW.md) for local startup checks. It binds the HTTP server to `127.0.0.1:18080` and uses a dummy Linear endpoint so the service can boot without real credentials.

## HTTP API

When enabled through `server.port` or `--port`, the server binds to loopback and exposes:

- `GET /`
- `GET /metrics`
- `GET /api/v1/state`
- `GET /api/v1/{issue_identifier}`
- `GET /api/v1/tasks`
- `POST /api/v1/tasks`
- `PATCH /api/v1/tasks/{issue_identifier}`
- `POST /api/v1/refresh`

Example:

```bash
curl -sS http://127.0.0.1:18080/api/v1/state
```

## Local Task Platform

Arpego now supports a built-in local task platform with:

- `tracker.kind: local`
- optional `tracker.file` task-store path
- task CRUD APIs under `/api/v1/tasks`
- a Libretto UI that can create and move tasks while still showing runtime state

If `tracker.file` is omitted in local mode, Arpego resolves `TASKS.yaml` relative
to the selected `WORKFLOW.md`.

Minimal local example:

```yaml
---
tracker:
  kind: local
  project_slug: sym
server:
  port: 18080
---
Work on the selected Symphony task.
```

## Monitoring Stack

The repo now includes a root [`docker-compose.yml`](/Users/pav/Documents/git/github/symphony-go/docker-compose.yml)
with:

- PostgreSQL for transactional storage foundation
- ClickHouse for observability/analytics foundation
- Prometheus and Grafana behind `--profile monitoring`

Compose example:

```bash
docker compose up --build
docker compose --profile monitoring up --build
```

Prometheus scrapes `GET /metrics` from Arpego. The compose workflow under
[`docker/WORKFLOW.compose.md`](/Users/pav/Documents/git/github/symphony-go/docker/WORKFLOW.compose.md)
uses the local task platform so the stack can boot without external tracker
credentials.

## Dashboard Serving

If a built Libretto dashboard is present, Arpego serves it from `/` and uses
SPA fallback for non-API routes.

Repo-local dashboard discovery currently checks:

- `libretto/dist`
- `../libretto/dist`

Typical local flow:

```bash
cd libretto
npm run build

cd ../arpego
go run ./cmd/arpego ./WORKFLOW.md
```

If no built dashboard is found, Arpego falls back to the minimal HTML status
page at `/`.

## Validation

```bash
cd arpego
go test ./...
go vet ./...
go build ./...
golangci-lint run
```

## Required Workflow Config

Minimum startup fields:

- `tracker.kind: linear`
- `tracker.api_key`
- `tracker.project_slug`
- `codex.command`

For local task-platform mode:

- `tracker.kind: local`
- `codex.command`

Useful optional fields:

- `polling.interval_ms`
- `workspace.root`
- `agent.max_concurrent_agents`
- `agent.max_concurrent_agents_by_state`
- `agent.max_retry_backoff_ms`
- `codex.read_timeout_ms`
- `codex.turn_timeout_ms`
- `codex.stall_timeout_ms`
- `hooks.after_create`, `hooks.before_run`, `hooks.after_run`, `hooks.before_remove`
- `server.port`
