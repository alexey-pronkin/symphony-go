## Context

`arpego/` is a bare Go module with a stub HTTP health server. The Elixir implementation in `elixir/` provides working reference behavior. SPEC.md defines the language-agnostic contract. This design targets Go 1.21+ (stdlib `log/slog`, `sync`, `context`, `os/exec`) and avoids heavy frameworks.

## Goals / Non-Goals

**Goals:**
- Implement all SPEC.md Section 18.1 Core Conformance requirements in Go
- Keep the package structure flat and focused: one package per concern
- Use the Go stdlib where possible; minimize external dependencies
- Produce a runnable binary: `arpego [path-to-WORKFLOW.md] [--port N]`
- Include unit tests for deterministic behavior; mark integration tests requiring credentials as skippable

**Non-Goals:**
- Frontend (`libretto/`) — not in this change
- Persistent retry queue across restarts (noted as TODO in SPEC.md 18.2)
- Non-Linear tracker adapters
- Full dashboard UI (HTML at `/` is fine as a stub; JSON API is the deliverable)

## Decisions

### Package layout

```
arpego/
  cmd/arpego/main.go          CLI entrypoint
  internal/
    workflow/    loader.go, watcher.go
    config/      config.go
    tracker/     linear.go, model.go
    orchestrator/ orchestrator.go, state.go, dispatch.go, retry.go, reconcile.go
    workspace/   manager.go, hooks.go, safety.go
    agent/       runner.go, client.go, protocol.go, events.go
    server/      server.go, handlers.go
    logging/     logging.go
```

**Rationale**: flat `internal/` packages let each concern be tested independently. No god-object. The orchestrator owns all mutable state via a mutex-protected struct; all other packages are stateless or receive explicit deps.

### Orchestrator concurrency model

Single `Orchestrator` struct with `sync.Mutex` protecting all state fields (`running`, `claimed`, `retryAttempts`, etc.). Workers run as goroutines; they send results back via a dedicated `chan workerResult`. The main poll loop processes the channel between ticks, avoiding race conditions without channels-of-channels complexity.

**Alternative considered**: actor model (channel-per-message). Rejected — overkill for Go; mutex + channel result is simpler and easier to test.

### Template engine

Use `text/template` (stdlib) with a thin adapter that maps Liquid `{{ issue.title }}` syntax to Go template `{{ .issue.title }}`. Unknown variables and filters fail explicitly (strict mode via `Option("missingkey=error")`).

**Alternative considered**: `aymerick/raymond` (Handlebars). Rejected — Liquid compatibility is not required by SPEC.md; "Liquid-compatible semantics are sufficient" means strict unknown-variable failure is what matters. stdlib keeps the dependency count low.

### File watcher

Use `github.com/fsnotify/fsnotify` for `WORKFLOW.md` watch/reload. On change, reload and reapply config without restart. Invalid reloads keep last known good config and emit a structured log error.

### Linear client

Pure `net/http` + `encoding/json` GraphQL client. No SDK dependency. Pagination via `pageInfo.endCursor`. Timeout: 30s per request.

### Codex app-server client

Launch via `exec.Command("bash", "-lc", codexCommand)`. Read stdout line-by-line with `bufio.Scanner` (max token 10MB). Parse JSON per line. Stderr drained separately (logged, not parsed). Write JSON-RPC messages to stdin. Turn timeout via `context.WithTimeout`. Stall detection in orchestrator (elapsed since last event).

### HTTP server

Optional; enabled by `--port` flag or `server.port` in WORKFLOW.md. Uses stdlib `net/http`. Bind to `127.0.0.1` by default. Graceful shutdown on SIGINT/SIGTERM with 10s deadline.

### Logging

`log/slog` with JSON handler to stderr. Structured fields: `issue_id`, `issue_identifier`, `session_id` injected via `slog.With`. No external logging library needed.

## Risks / Trade-offs

- **Mutex contention**: Single mutex on orchestrator state could bottleneck at high concurrency. Mitigation: acceptable at default `max_concurrent_agents=10`; profiling deferred to post-MVP.
- **Template syntax gap**: SPEC.md says "Liquid-compatible semantics are sufficient"; `text/template` uses `{{ }}` not `{% %}`. Mitigation: document that WORKFLOW.md templates must use Go template syntax, not Liquid; update sample WORKFLOW.md accordingly.
- **fsnotify reliability on macOS**: kqueue events may batch or miss rapid successive writes. Mitigation: defensive re-read inside dispatch preflight validation every tick regardless of watch events (SPEC.md §6.2).
- **Codex binary not found**: If `codex` is not in PATH inside `bash -lc`, launch fails. Mitigation: validate `codex.command` at startup preflight and emit clear error; not a runtime crash.

## Migration Plan

1. Implement packages incrementally in dependency order: `workflow` → `config` → `tracker` → `workspace` → `agent` → `orchestrator` → `server` → `cmd`
2. Each package gets its own test file before wiring
3. Replace stub `internal/app/app.go` with the full wiring once all packages exist
4. Update `go.mod` with required external deps: `fsnotify`, `yaml.v3`
5. Add `Makefile` targets: `build`, `test`, `lint`

## Open Questions

- Should the HTTP dashboard at `/` serve a React SPA (from `libretto/`) or a simple server-rendered HTML page? **Decision for MVP**: simple server-rendered HTML status page; SPA integration deferred.
