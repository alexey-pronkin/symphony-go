## ADDED Requirements

### Requirement: Dashboard loads and refreshes runtime state
The Libretto application SHALL load Symphony runtime state from
`GET /api/v1/state` on initial render, expose a manual refresh action, and
continue refreshing state on a fixed client-side polling interval.

#### Scenario: Initial state load succeeds
- **WHEN** the dashboard mounts
- **THEN** it fetches `/api/v1/state` and renders the returned counts, runtime
  totals, and issue lists

#### Scenario: Manual refresh triggers immediate reload
- **WHEN** the operator activates the refresh control
- **THEN** the dashboard reloads `/api/v1/state` and `POST /api/v1/refresh`

#### Scenario: State load fails
- **WHEN** `/api/v1/state` returns an error or invalid payload
- **THEN** the dashboard shows an operator-visible error state and allows retry

### Requirement: Dashboard presents runtime summary and issue queues
The Libretto dashboard SHALL render a human-readable summary of the current
runtime, including counts, token totals, running sessions, and retry queue
entries from the state response.

#### Scenario: Running and retrying rows are present
- **WHEN** the state response contains running or retrying entries
- **THEN** the dashboard renders list rows with issue identifiers, status
  fields, and the most relevant timing or token metadata

#### Scenario: State response is empty
- **WHEN** the state response contains no running or retrying entries
- **THEN** the dashboard shows explicit empty-state messaging instead of blank
  panels

### Requirement: Dashboard loads selected issue detail
The Libretto dashboard SHALL load issue detail from
`GET /api/v1/{issue_identifier}` when an operator selects a running or retrying
issue from the dashboard.

#### Scenario: Running issue detail loads
- **WHEN** the operator selects a running issue row
- **THEN** the dashboard fetches `/api/v1/{issue_identifier}` and renders the
  returned workspace, session, and attempt information

#### Scenario: Unknown issue detail fails
- **WHEN** the selected issue detail request returns `404`
- **THEN** the dashboard shows a detail-panel error state without clearing the
  rest of the dashboard

### Requirement: Dashboard API target is configurable
The Libretto frontend SHALL allow the Symphony API base URL to be configured by
environment variable and default to same-origin requests when unset.

#### Scenario: Explicit API base URL is configured
- **WHEN** `VITE_SYMPHONY_API_BASE_URL` is set
- **THEN** dashboard requests are sent relative to that base URL

#### Scenario: API base URL is unset
- **WHEN** `VITE_SYMPHONY_API_BASE_URL` is absent
- **THEN** dashboard requests are sent to same-origin `/api/v1/*` paths
