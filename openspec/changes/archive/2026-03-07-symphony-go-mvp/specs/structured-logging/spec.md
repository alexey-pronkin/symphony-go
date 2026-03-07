## ADDED Requirements

### Requirement: Structured JSON logs to stderr
The system SHALL emit structured logs using `log/slog` with JSON format to stderr. Log level SHALL be configurable (default `INFO`).

#### Scenario: Logs are valid JSON
- **WHEN** the service emits a log line
- **THEN** each line is parseable JSON

### Requirement: Required context fields on issue logs
Issue-related log entries SHALL include `issue_id` and `issue_identifier` fields. Session lifecycle logs SHALL also include `session_id`.

#### Scenario: Dispatch log includes issue fields
- **WHEN** an issue is dispatched
- **THEN** the log entry has `issue_id` and `issue_identifier`

#### Scenario: Session log includes session_id
- **WHEN** a Codex session starts
- **THEN** the log entry has `session_id` in `<thread_id>-<turn_id>` format

### Requirement: Validation failures are operator-visible
The system SHALL log dispatch preflight validation failures at ERROR level so operators can diagnose issues without a debugger.

#### Scenario: Missing API key logged at error
- **WHEN** preflight validation fails due to missing tracker API key
- **THEN** an ERROR log entry is emitted describing the failure
