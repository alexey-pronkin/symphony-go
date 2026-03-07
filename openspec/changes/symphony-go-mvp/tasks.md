## 1. Foundation Branch (`main`) [done]

This slice already landed in commit `66f0759`.

- [x] 1.1 Bootstrap repo tooling: shared `Makefile`, git hooks, lint/format config, Nx workspace, and generated OpenSpec integration assets
- [x] 1.2 Implement `internal/workflow`: loader, watcher, template rendering, and tests
- [x] 1.3 Implement `internal/config`: typed getters, env/path resolution, dispatch validation, and tests
- [x] 1.4 Add baseline Docker, Traefik, CrowdSec, and Trivy scaffolding plus repo docs

## 2. Branch Slice: `feat/tracker-workspace-slice` [current]

Goal: finish the tracker and workspace contracts from `SPEC.md` sections 9 and 11 so the orchestrator slice has real dependencies to build against.

### Commit A: Tracker Models And Tests

- [x] 2.1 Add failing tests for Linear normalization, pagination, empty-ID short-circuit, and error categorization
- [x] 2.2 Implement `internal/tracker/model.go`: `Issue` and `BlockerRef` matching `SPEC.md` section 4.1.1
- [x] 2.3 Implement normalization helpers: labels lowercase, blockers from inverse `blocks` relations, priority integer-or-nil, parsed timestamps

### Commit B: Linear Client

- [x] 2.4 Implement `internal/tracker/linear.go`: GraphQL client with `FetchCandidates()`, `FetchByStates()`, and `FetchStatesByIDs()`
- [x] 2.5 Implement transport and payload error mapping: `linear_api_request`, `linear_api_status`, `linear_graphql_errors`, `linear_unknown_payload`
- [x] 2.6 Make tracker tests pass under `go test ./...`

### Commit C: Workspace Safety And Hooks

- [x] 2.7 Add failing tests for sanitize key, root containment, create/reuse behavior, and hook timeout/failure semantics
- [x] 2.8 Implement `internal/workspace/safety.go`: sanitized workspace keys and root-containment validation
- [x] 2.9 Implement `internal/workspace/manager.go`: deterministic workspace creation/reuse with `created_now`
- [x] 2.10 Implement `internal/workspace/hooks.go`: `bash -lc` hook execution with timeout and per-hook failure handling
- [x] 2.11 Make workspace tests pass under `go test ./...`

## 3. Branch Slice: Agent Runner

Goal: satisfy `SPEC.md` section 10 on top of the tracker/workspace foundations.

- [x] 3.1 Add failing tests for handshake ordering, line-buffered parsing, approval handling, unsupported tools, and token accounting
- [x] 3.2 Implement `internal/agent/protocol.go` request/response types
- [x] 3.3 Implement `internal/agent/client.go` stdio subprocess client
- [x] 3.4 Implement `internal/agent/session.go` and `events.go` for handshake, event streaming, and token/rate-limit extraction
- [x] 3.5 Implement `internal/agent/runner.go` for workspace → prompt → session → turn loop lifecycle

## 4. Branch Slice: Orchestrator And Logging

Goal: satisfy `SPEC.md` sections 8 and 13.1-13.5.

- [x] 4.1 Add failing tests for dispatch eligibility, sort order, per-state concurrency, retry backoff, reconciliation, and stall detection
- [x] 4.2 Implement `internal/logging/logging.go`
- [x] 4.3 Implement `internal/orchestrator/state.go`, `dispatch.go`, `retry.go`, `reconcile.go`, and `orchestrator.go`
- [x] 4.4 Wire structured logging and aggregate runtime/token accounting through orchestrator state

## 5. Branch Slice: HTTP API And Entrypoint

Goal: finish the optional HTTP surface plus CLI wiring from `SPEC.md` section 13.7 and the entrypoint requirements.

- [x] 5.1 Add failing tests for `/api/v1/state`, `/api/v1/{id}`, `/api/v1/refresh`, `405`, and unknown issue handling
- [x] 5.2 Implement `internal/server/server.go` and `handlers.go`
- [x] 5.3 Replace stub `internal/app/app.go` wiring with workflow/config/orchestrator/server startup and graceful shutdown
- [x] 5.4 Add CLI arg parsing for `[workflow-path]` and `--port`

## 6. Validation And Docs

- [x] 6.1 Run `go test ./...` in `arpego/`
- [x] 6.2 Run `go vet ./...` and `go build ./...` in `arpego/`
- [x] 6.3 Run `golangci-lint run` in `arpego/`
- [x] 6.4 Write a sample `WORKFLOW.md` for local testing and verify `arpego ./WORKFLOW.md` starts without errors
- [x] 6.5 Verify `/api/v1/state` returns valid JSON when server is enabled
- [x] 6.6 Update `arpego/README.md` with build, run, and config instructions
