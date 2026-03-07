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
- `GET /api/v1/state`
- `GET /api/v1/{issue_identifier}`
- `POST /api/v1/refresh`

Example:

```bash
curl -sS http://127.0.0.1:18080/api/v1/state
```

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
