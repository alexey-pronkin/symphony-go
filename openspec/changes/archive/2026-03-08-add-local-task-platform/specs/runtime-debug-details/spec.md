## Requirement: Issue detail includes recent events and session log references

The issue detail API SHALL expose richer debugging information for active or retrying issues.

#### Scenario: Running issue detail includes recent events
- **GIVEN** an issue is currently running
- **WHEN** a client sends `GET /api/v1/MT-101`
- **THEN** the response includes a bounded `recent_events` list
- **AND** each event includes its timestamp, event name, and a concise message summary

#### Scenario: Running issue detail includes log references
- **GIVEN** an issue has an associated Codex session log
- **WHEN** a client sends `GET /api/v1/MT-101`
- **THEN** the response includes `logs.codex_session_logs`
- **AND** each log entry includes a label and filesystem path
