## ADDED Requirements

### Requirement: Delivery alert filters show counts

The delivery rollup MUST expose the number of alerts available for each severity filter option.

#### Scenario: Operator reviews alert volume before filtering

- **WHEN** the delivery panel renders a rollup with critical and warning alerts
- **THEN** the filter controls show counts for `All`, `Critical`, and `Warnings`
- **AND** the counts are derived from the same prioritized rollup alerts rendered in the panel

### Requirement: Delivery alert filtering shows an explicit empty state

The delivery rollup MUST show an explicit empty state when the active severity filter has no matching alerts.

#### Scenario: Selected severity has no matching alerts

- **GIVEN** the operator selects a severity filter
- **WHEN** the current rollup has zero alerts for that severity
- **THEN** the panel shows a short empty-state message instead of a blank space
- **AND** the rest of the delivery panel remains available
