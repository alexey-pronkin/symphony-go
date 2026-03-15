## ADDED Requirements

### Requirement: Delivery source focus can be toggled

The delivery panel MUST let operators toggle source focus directly from source-backed rollup alerts.

#### Scenario: Operator clicks the same focused alert again

- **GIVEN** a source-backed delivery alert is already focused
- **WHEN** the operator activates the same alert again
- **THEN** the source focus is cleared

#### Scenario: Operator clicks a different source-backed alert

- **GIVEN** one source is already focused
- **WHEN** the operator activates a different source-backed alert
- **THEN** focus moves to the new source

### Requirement: Delivery panel exposes an explicit focus reset action

The delivery panel MUST show a clear action while a source focus is active.

#### Scenario: Operator clears focus from the panel notice

- **GIVEN** a source is currently focused
- **WHEN** the operator activates the clear action in the delivery panel
- **THEN** the source list returns to the unfocused state
