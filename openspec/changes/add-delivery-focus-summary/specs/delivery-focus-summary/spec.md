## ADDED Requirements

### Requirement: Delivery panel summarizes the focused source

The delivery panel MUST show a compact summary of the currently focused SCM source.

#### Scenario: Operator focuses a source-backed alert

- **GIVEN** a source-backed alert activates focus for an SCM source
- **WHEN** the focused source still exists in the current report
- **THEN** the delivery panel shows the source name, provider kind, and main branch in the focus notice
- **AND** the notice shows key merge-readiness metrics for that source

### Requirement: Focus summary only renders for resolvable sources

The delivery panel MUST only render the focused-source summary when the focused source key matches a current source.

#### Scenario: Focused source no longer exists

- **GIVEN** the dashboard refreshes and the previously focused source is no longer present
- **WHEN** the delivery panel renders
- **THEN** the focused-source summary is not shown
