## 1. Bootstrap

- [ ] 1.1 Update `arpego/go.mod` to Go 1.21; add dependencies: `gopkg.in/yaml.v3`, `github.com/fsnotify/fsnotify`
- [ ] 1.2 Add `arpego/Makefile` with `build`, `test`, `lint` targets

## 2. Workflow Loader (`internal/workflow`)

- [ ] 2.1 Implement `loader.go`: `Load(path string) (WorkflowDefinition, error)` — YAML front matter split, prompt body trim, typed error returns
- [ ] 2.2 Implement `watcher.go`: `Watch(path string, onChange func(WorkflowDefinition)) (io.Closer, error)` using `fsnotify`; invalid reload keeps last good config
- [ ] 2.3 Implement `template.go`: `Render(tmpl string, issue Issue, attempt *int) (string, error)` using `text/template` with `missingkey=error`
- [ ] 2.4 Write `workflow_test.go`: cover load/parse/error cases, template strict rendering

## 3. Config Layer (`internal/config`)

- [ ] 3.1 Implement `config.go`: `Config` struct with typed getters for all SPEC.md Section 6.4 fields, defaults, `$VAR` resolution, `~` expansion
- [ ] 3.2 Implement `validate.go`: `ValidateDispatch(c Config) error` — check tracker.kind, api_key, project_slug, codex.command
- [ ] 3.3 Write `config_test.go`: defaults, $VAR resolution, ~ expansion, preflight validation cases

## 4. Issue Tracker Client (`internal/tracker`)

- [ ] 4.1 Implement `model.go`: `Issue`, `BlockerRef` structs matching SPEC.md Section 4.1.1
- [ ] 4.2 Implement `linear.go`: `Client` with `FetchCandidates()`, `FetchByStates()`, `FetchStatesByIDs()` — GraphQL over `net/http`, 30s timeout, pagination
- [ ] 4.3 Implement normalization: labels lowercase, blockers from inverse `blocks` relations, priority int or nil, ISO-8601 timestamps
- [ ] 4.4 Implement error categorization: `linear_api_request`, `linear_api_status`, `linear_graphql_errors`, `linear_unknown_payload`
- [ ] 4.5 Write `linear_test.go`: normalization unit tests with mock HTTP server; pagination test; empty-ID-list short-circuit

## 5. Workspace Manager (`internal/workspace`)

- [ ] 5.1 Implement `safety.go`: `SanitizeKey(id string) string`, `ValidatePath(root, path string) error` (root containment check)
- [ ] 5.2 Implement `manager.go`: `EnsureWorkspace(root, identifier string) (Workspace, error)` — create or reuse, `created_now` flag
- [ ] 5.3 Implement `hooks.go`: `RunHook(script, cwd string, timeoutMs int) error` via `bash -lc`, per-hook failure semantics
- [ ] 5.4 Write `workspace_test.go`: sanitize key, root containment, create/reuse, hook timeout

## 6. Agent Runner (`internal/agent`)

- [ ] 6.1 Implement `protocol.go`: JSON-RPC message types for `initialize`, `initialized`, `thread/start`, `turn/start`, `turn/completed`, `turn/failed`, approvals, tool calls
- [ ] 6.2 Implement `client.go`: `AppServerClient` — `bash -lc` subprocess launch, stdout line scanner (10MB), stderr drain goroutine, stdin writer
- [ ] 6.3 Implement `session.go`: startup handshake sequence with `read_timeout_ms`; extract `thread_id`, `turn_id`; emit `session_started`
- [ ] 6.4 Implement turn streaming: `RunTurn(ctx, prompt, issue) (TurnResult, error)` with `turn_timeout_ms` context deadline
- [ ] 6.5 Implement approval handler: auto-approve command and file-change; return failure for unsupported tools; fail immediately on user-input-required
- [ ] 6.6 Implement `events.go`: token accounting — prefer thread totals, track deltas, emit `CodexUpdate` to orchestrator
- [ ] 6.7 Implement `runner.go`: `RunAttempt(issue, attempt, hooks, onEvent)` — full worker lifecycle: workspace → before_run → session → turn loop → after_run
- [ ] 6.8 Write `client_test.go`: mock subprocess test (echo server), handshake, approval, unsupported tool, token accounting

## 7. Orchestrator (`internal/orchestrator`)

- [ ] 7.1 Implement `state.go`: `State` struct with all fields from SPEC.md Section 4.1.8; mutex-protected
- [ ] 7.2 Implement `dispatch.go`: `dispatch(issue, attempt)` — spawn goroutine, update claimed/running, cancel existing retry timer
- [ ] 7.3 Implement `retry.go`: `scheduleRetry(issueID, attempt, info)` — compute backoff delay, create timer goroutine, store in `retryAttempts`
- [ ] 7.4 Implement `reconcile.go`: stall detection + tracker state refresh each tick; terminal → stop+cleanup; non-active → stop only
- [ ] 7.5 Implement `orchestrator.go`: `Orchestrator.Run(ctx)` — startup cleanup, immediate tick, ticker loop, result channel processing
- [ ] 7.6 Write `orchestrator_test.go`: dispatch eligibility, sort order, todo blocker rule, retry backoff, reconciliation transitions, stall detection — all with mock tracker

## 8. Structured Logging (`internal/logging`)

- [ ] 8.1 Implement `logging.go`: `New() *slog.Logger` — JSON handler to stderr; `WithIssue(l, id, identifier)`, `WithSession(l, sessionID)` helpers
- [ ] 8.2 Wire logger into all packages; verify `issue_id`, `issue_identifier`, `session_id` appear on relevant log entries

## 9. HTTP API (`internal/server`)

- [ ] 9.1 Implement `server.go`: `Server` with `Start(port)`, graceful `Shutdown(ctx)`, loopback bind
- [ ] 9.2 Implement `handlers.go`: `GET /api/v1/state`, `GET /api/v1/{id}`, `POST /api/v1/refresh` — JSON responses per SPEC.md Section 13.7.2
- [ ] 9.3 Implement minimal `GET /` HTML status page (server-rendered, no SPA)
- [ ] 9.4 Return `405` for unsupported methods; use JSON error envelope for all errors
- [ ] 9.5 Write `server_test.go`: state endpoint shape, 404 for unknown issue, 405 for wrong method, refresh 202

## 10. CLI Entrypoint (`cmd/arpego`)

- [ ] 10.1 Replace stub `internal/app/app.go` with full wiring: init logger, load workflow, validate, run startup cleanup, start orchestrator, optionally start HTTP server
- [ ] 10.2 Add CLI arg parsing: positional `[workflow-path]`, `--port` flag
- [ ] 10.3 Wire SIGINT/SIGTERM graceful shutdown (10s HTTP drain, cancel orchestrator context)
- [ ] 10.4 Exit nonzero with clear message on startup validation failure

## 11. Validation

- [ ] 11.1 Run `go test ./...` in `arpego/` — all unit tests pass
- [ ] 11.2 Run `go vet ./...` and `go build ./...` — clean
- [ ] 11.3 Run `golangci-lint run` (if available) — no lint errors
- [ ] 11.4 Write a sample `WORKFLOW.md` for local testing and verify `arpego ./WORKFLOW.md` starts without errors
- [ ] 11.5 Verify `/api/v1/state` returns valid JSON when server is enabled
- [ ] 11.6 Update `arpego/README.md` with build, run, and config instructions
