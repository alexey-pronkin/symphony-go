## Why

The `arpego/` Go backend is a stub with only a health endpoint. Symphony needs a fully operational Go implementation conforming to SPEC.md Section 18.1 (Core Conformance) so the service can actually orchestrate Codex agents against Linear issues. The Elixir implementation serves as reference only; Go is the target stack.

## What Changes

- Implement `workflow` package: WORKFLOW.md loader with YAML front matter parsing, prompt body extraction, and file-watch dynamic reload
- Implement `config` package: typed config layer with defaults, `$VAR` env resolution, `~` path expansion, and dispatch preflight validation
- Implement `tracker` package: Linear GraphQL client with candidate fetch, state refresh, terminal fetch, pagination, and normalized Issue model
- Implement `orchestrator` package: poll loop, single-authority state machine, dispatch/retry/reconciliation, stall detection, exponential backoff
- Implement `workspace` package: per-issue workspace lifecycle with sanitized paths, root containment safety, and all four hooks
- Implement `agent` package: Codex app-server subprocess client over JSON-RPC stdio (initialize, thread/start, turn/start, streaming turn processing, approval handling, token accounting)
- Implement `server` package: optional HTTP server with `/api/v1/state`, `/api/v1/:id`, `POST /api/v1/refresh`
- Implement `logging` package: structured logging with `issue_id`, `issue_identifier`, `session_id` context fields
- Wire CLI: positional WORKFLOW.md path argument, `--port` flag, startup validation, graceful shutdown
- Add unit tests for all packages targeting Core Conformance test matrix (SPEC.md Section 17)

## Capabilities

### New Capabilities

- `workflow-loader`: Parse WORKFLOW.md front matter + prompt body; file-watch dynamic reload; error classes
- `config-layer`: Typed config getters, defaults, `$VAR`/`~` resolution, dispatch preflight validation
- `linear-tracker`: Linear GraphQL adapter — candidate fetch, state refresh, terminal fetch, pagination, normalization
- `orchestrator`: Poll loop, in-memory state machine, dispatch, retry queue with exponential backoff, stall detection, reconciliation
- `workspace-manager`: Per-issue workspace creation/reuse, sanitized keys, root containment, lifecycle hooks
- `agent-runner`: Codex app-server stdio client, JSON-RPC handshake, streaming turn processing, approval/tool-call handling, token accounting
- `http-api`: Optional HTTP server — `/api/v1/state`, `/api/v1/:id`, `POST /api/v1/refresh`, dashboard at `/`
- `structured-logging`: slog-based structured logs with required context fields

### Modified Capabilities

- `arpego-entrypoint`: Replace stub HTTP-only server with full service wiring (CLI args, all packages, graceful shutdown)

## Impact

- `arpego/` — primary implementation target; all new packages added here
- `go.mod` — will add dependencies: `gopkg.in/yaml.v3`, `github.com/go-yaml/yaml`, `github.com/fsnotify/fsnotify` (file watch), `github.com/aymerick/raymond` or `github.com/Masterminds/sprig` (Liquid-compatible templates), `log/slog` (stdlib Go 1.21+)
- `SPEC.md` — no changes required; implementation conforms without extending
- `WORKFLOW.md` — user-provided repo runtime config; no changes to spec contract
- No Elixir or frontend impact
