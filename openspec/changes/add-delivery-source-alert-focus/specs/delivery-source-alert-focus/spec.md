## ADDED Requirements

### Requirement: Source-backed delivery alerts MUST support source focus
The Libretto delivery alert rollup MUST let operators focus the SCM source behind a source-related alert.

#### Scenario: Operator selects a source warning alert
- **WHEN** a delivery alert is tied to a specific SCM source
- **AND** the operator selects that alert
- **THEN** the matching source becomes focused in the delivery panel

### Requirement: Focused sources MUST be visually identifiable
The delivery panel MUST make the focused SCM source visually identifiable.

#### Scenario: Source focus is active
- **WHEN** an SCM source has been focused from the alert rollup
- **THEN** the corresponding source card is highlighted

### Requirement: Source focus MUST follow the current delivery report
The delivery panel MUST avoid keeping stale source focus when the current report no longer contains the selected source.

#### Scenario: Delivery report refresh removes the focused source
- **WHEN** the delivery report changes
- **AND** the previously focused source no longer exists
- **THEN** the panel clears the stale source focus
