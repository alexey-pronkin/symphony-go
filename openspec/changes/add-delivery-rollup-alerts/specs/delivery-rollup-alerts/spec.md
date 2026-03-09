## ADDED Requirements

### Requirement: Dashboard MUST show a compact delivery alert rollup
The Libretto delivery insights panel MUST show a compact rollup of the most important delivery issues.

#### Scenario: Operator opens the delivery panel
- **WHEN** delivery insights are available
- **THEN** the dashboard shows a short alert list derived from the current report

### Requirement: Rollup alerts MUST prioritize critical delivery risks
The delivery alert rollup MUST prioritize critical issues ahead of warning-level issues.

#### Scenario: Report contains critical and warning conditions
- **WHEN** the current delivery report includes both critical and warning signals
- **THEN** critical alerts appear before warning alerts

### Requirement: Rollup alerts MUST derive from the current delivery report
The delivery alert rollup MUST use the current report payload rather than stale cached state.

#### Scenario: Delivery report changes after refresh
- **WHEN** the dashboard refreshes delivery insights
- **THEN** the rollup alerts reflect the latest report contents
