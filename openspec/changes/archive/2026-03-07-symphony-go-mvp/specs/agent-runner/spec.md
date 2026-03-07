## ADDED Requirements

### Requirement: Launch Codex app-server via bash -lc
The system SHALL launch the configured `codex.command` via `bash -lc <command>` with cwd set to the workspace path.

#### Scenario: Subprocess starts in workspace directory
- **WHEN** a worker is dispatched for an issue
- **THEN** the Codex subprocess cwd is the per-issue workspace path

#### Scenario: Wrong cwd rejected before launch
- **WHEN** the resolved workspace path is not the intended cwd
- **THEN** the launch is aborted with `invalid_workspace_cwd` error

### Requirement: JSON-RPC startup handshake
The system SHALL send `initialize`, `initialized`, `thread/start`, `turn/start` messages in order, waiting for responses with `codex.read_timeout_ms` timeout.

#### Scenario: Startup handshake completes successfully
- **WHEN** the app-server responds to all startup messages
- **THEN** thread_id and turn_id are extracted and session_started event is emitted

#### Scenario: Read timeout during handshake fails session
- **WHEN** no response arrives within `codex.read_timeout_ms`
- **THEN** the session fails with `response_timeout` error

### Requirement: Line-buffered stdout parsing
The system SHALL buffer stdout line by line (max 10MB per line) and parse complete lines as JSON. Stderr SHALL be drained and logged separately, never parsed as protocol.

#### Scenario: Partial line buffered until newline
- **WHEN** a protocol message arrives in two chunks without a newline
- **THEN** parsing waits for the newline before processing

#### Scenario: Non-JSON stderr does not crash parser
- **WHEN** Codex writes a plaintext message to stderr
- **THEN** it is logged and parsing continues

### Requirement: Turn timeout and stall handling
The system SHALL enforce `codex.turn_timeout_ms` per turn. Stall detection (no event within `codex.stall_timeout_ms`) is enforced by the orchestrator.

#### Scenario: Turn timeout kills subprocess and fails attempt
- **WHEN** a turn exceeds `codex.turn_timeout_ms`
- **THEN** the subprocess is terminated and the attempt fails with `turn_timeout`

### Requirement: Auto-approval and unsupported tool handling
The system SHALL auto-approve command and file-change approvals. Unsupported dynamic tool calls SHALL return a failure result without stalling. User-input-required SHALL immediately fail the run.

#### Scenario: Approval request is auto-approved
- **WHEN** the app-server sends an approval request
- **THEN** the system responds with `{"approved": true}` and continues

#### Scenario: Unsupported tool call returns failure
- **WHEN** the app-server requests an unknown tool
- **THEN** the system returns `{"success": false, "error": "unsupported_tool_call"}` and the session continues

#### Scenario: User input required fails the run
- **WHEN** the app-server signals user input is required
- **THEN** the run attempt fails immediately with `turn_input_required`

### Requirement: Token accounting
The system SHALL extract input/output/total token counts from agent events, prefer absolute thread totals, track deltas to avoid double-counting, and accumulate in orchestrator state.

#### Scenario: Thread token totals accumulated without double-counting
- **WHEN** two consecutive events both report cumulative thread totals
- **THEN** the orchestrator increments only the delta since the last report
