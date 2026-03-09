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
- `GET /api/v1/insights/delivery`
- `GET /api/v1/insights/delivery/trends`
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
- optional `tracker.storage: file|postgres` selector
- optional `tracker.file` task-store path
- optional `storage.postgres_dsn` or `SYMPHONY_POSTGRES_DSN` for Postgres-backed local storage
- optional `storage.clickhouse_dsn` or `SYMPHONY_CLICKHOUSE_DSN` for persisted runtime-event history
- task CRUD APIs under `/api/v1/tasks`
- a Libretto UI that can create and move tasks while still showing runtime state

If `tracker.file` is omitted in local mode, Arpego resolves `TASKS.yaml` relative
to the selected `WORKFLOW.md`.

If `tracker.storage` is omitted in local mode, Arpego uses the file-backed
adapter. When `tracker.storage: postgres` is selected, Arpego uses the
transactional Postgres adapter implemented with GORM.

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

Postgres-backed local example:

```yaml
---
tracker:
  kind: local
  storage: postgres
  project_slug: sym
storage:
  postgres_dsn: $SYMPHONY_POSTGRES_DSN
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

When `storage.clickhouse_dsn` is configured, Arpego also persists runtime
events to ClickHouse and uses them to enrich `GET /api/v1/{issue_identifier}`
with recent event history and session-log references beyond the in-memory
runtime ring. The same ClickHouse DSN is also used for delivery trend
snapshots consumed by `GET /api/v1/insights/delivery/trends`.

## Delivery Metrics

Libretto now also consumes `GET /api/v1/insights/delivery` for a compact
delivery dashboard with:

- integral metrics: delivery health, flow efficiency, merge readiness, predictability
- agile-oriented task signals: throughput, completion ratio, review load
- kanban-oriented task signals: WIP, blocked ratio, aging work, flow load
- SCM gitflow and review/CI signals grouped by configured source
- historical trend cards from `GET /api/v1/insights/delivery/trends?window=7d&limit=12`

SCM sources are configured under `insights.scm_sources` and can be labeled as
`github`, `gitlab`, or `gitverse`. Sources may mix local `repo_path` inspection
with provider metadata such as `api_url`, `repository`, `project_id`, and
`api_token`. GitHub and GitLab sources now augment local branch metrics with
open-change, approval, stale-review, and failing-check signals. GitVerse
currently degrades gracefully with explicit source warnings when provider
metrics are unavailable.

Example:

```yaml
---
tracker:
  kind: local
  storage: postgres
  project_slug: sym
storage:
  postgres_dsn: $SYMPHONY_POSTGRES_DSN
insights:
  stale_branch_hours: 72
  throughput_window_days: 7
  scm_sources:
    - kind: github
      name: symphony-core
      repo_path: ~/src/symphony-go
      main_branch: main
      repository: alexey-pronkin/symphony-go
      api_token: $GITHUB_TOKEN
    - kind: gitlab
      name: internal-platform
      repo_path: ~/src/internal-platform
      main_branch: master
      api_url: https://gitlab.example.com/api/v4
      project_id: group%2Finternal-platform
      api_token: $GITLAB_TOKEN
server:
  port: 18080
---
Work on the selected Symphony task.
```

Trend queries are intentionally bounded. Arpego currently accepts `window`
values `24h`, `7d`, `30d`, and `90d`, plus a `limit` cap up to `48` points so
the dashboard can stay compact even when the analytics store contains more raw
snapshots.

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
