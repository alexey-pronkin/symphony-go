## ADDED Requirements

### Requirement: Optional HTTP server enablement
The system SHALL start an HTTP server bound to `127.0.0.1` when `--port` CLI flag or `server.port` WORKFLOW.md config is present. CLI `--port` SHALL override `server.port`.

#### Scenario: Server starts on --port flag
- **WHEN** `--port 8080` is provided
- **THEN** HTTP server binds to `127.0.0.1:8080`

#### Scenario: Server not started without port config
- **WHEN** neither `--port` nor `server.port` is set
- **THEN** no HTTP listener is started

### Requirement: GET /api/v1/state returns runtime summary
The system SHALL return JSON with `generated_at`, `counts`, `running`, `retrying`, `codex_totals`, and `rate_limits` fields.

#### Scenario: State endpoint returns current running sessions
- **WHEN** one issue is running
- **THEN** `counts.running` is 1 and `running` array has one entry with required fields

#### Scenario: Empty state returns zeroed totals
- **WHEN** no issues are running or retrying
- **THEN** response has empty arrays and zero token counts

### Requirement: GET /api/v1/:id returns issue detail or 404
The system SHALL return issue-specific runtime details when the identifier is known, or `404` with `{"error":{"code":"issue_not_found","message":"..."}}` when unknown.

#### Scenario: Known issue returns detail
- **WHEN** `MT-649` is currently running
- **THEN** GET `/api/v1/MT-649` returns status, workspace, session, and token info

#### Scenario: Unknown issue returns 404
- **WHEN** `MT-999` is not in current state
- **THEN** GET `/api/v1/MT-999` returns 404 with error envelope

### Requirement: POST /api/v1/refresh queues immediate poll
The system SHALL trigger an immediate poll+reconcile cycle (best-effort) and return `202 Accepted`.

#### Scenario: Refresh accepted and triggers poll
- **WHEN** POST /api/v1/refresh is called
- **THEN** response is 202 with `{"queued":true}` and a poll cycle is triggered
