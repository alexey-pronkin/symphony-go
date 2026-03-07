## ADDED Requirements

### Requirement: Delivery metrics API returns integral task and gitflow metrics
The system SHALL expose a delivery metrics API that combines task-platform metrics with SCM gitflow metrics and returns a compact dashboard-focused payload.

#### Scenario: Delivery metrics API returns integral scores
- **WHEN** a client sends `GET /api/v1/insights/delivery`
- **THEN** the server returns JSON with integral metrics for delivery health, flow efficiency, merge readiness, and predictability
- **AND** the response includes supporting task metrics and SCM source metrics

#### Scenario: Delivery metrics API tolerates missing SCM sources
- **WHEN** no SCM sources are configured or a configured source cannot be inspected
- **THEN** the server still returns task-platform metrics
- **AND** the response includes warnings describing the missing or degraded SCM coverage

### Requirement: SCM sources support provider labels for GitHub, GitLab, and GitVerse
The system SHALL accept SCM source definitions that identify the provider kind and local repository path used for metric collection.

#### Scenario: SCM source uses provider-specific label
- **GIVEN** a configured SCM source with `kind` set to `github`, `gitlab`, or `gitverse`
- **WHEN** delivery metrics are computed
- **THEN** the SCM metrics output preserves that provider label for the source
- **AND** the source contributes to the aggregate gitflow metrics when its repository can be inspected

### Requirement: Dashboard presents integral metrics with graceful degradation
The dashboard SHALL render the delivery metrics in a compact section and remain usable when only partial metrics are available.

#### Scenario: Dashboard renders delivery score cards
- **WHEN** the delivery metrics API returns successfully
- **THEN** the dashboard shows a compact set of score cards for the integral metrics
- **AND** the dashboard shows supporting task-flow and SCM breakdowns

#### Scenario: Dashboard shows degraded metrics state
- **WHEN** delivery metrics include warnings or partial SCM coverage
- **THEN** the dashboard shows the warnings without hiding the available task metrics
- **AND** the rest of the runtime and task platform UI remains usable
