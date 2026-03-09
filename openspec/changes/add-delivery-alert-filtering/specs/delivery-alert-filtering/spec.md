## ADDED Requirements

### Requirement: Delivery rollup MUST support severity filtering
The Libretto delivery alert rollup MUST let operators filter alerts by severity.

#### Scenario: Operator shows only critical alerts
- **WHEN** the operator selects the critical-only filter
- **THEN** the rollup shows only critical alerts

### Requirement: Delivery rollup MUST preserve the default combined view
The Libretto delivery alert rollup MUST preserve a default view containing all alert severities.

#### Scenario: Operator returns to the full alert list
- **WHEN** the operator selects the all-alerts filter
- **THEN** the rollup shows the full prioritized alert list

### Requirement: Source-focused alerts MUST remain actionable through filters
The Libretto delivery rollup MUST keep source focus actions available on visible filtered alerts.

#### Scenario: Operator focuses a source from a filtered alert list
- **WHEN** a visible filtered alert is tied to a specific SCM source
- **AND** the operator selects that alert action
- **THEN** the delivery panel focuses the matching source
