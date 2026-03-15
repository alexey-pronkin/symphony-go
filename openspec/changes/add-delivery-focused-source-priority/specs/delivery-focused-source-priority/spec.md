## ADDED Requirements

### Requirement: Focused delivery source is rendered first

The delivery panel MUST render the currently focused SCM source at the top of the source list.

#### Scenario: Operator focuses a source-backed alert

- **GIVEN** a delivery alert focuses an SCM source
- **WHEN** the delivery source list renders
- **THEN** the focused source appears first in the list
- **AND** the remaining sources keep their relative order

### Requirement: Source list order is unchanged without focus

The delivery panel MUST keep the original SCM source ordering when no source is focused.

#### Scenario: No focused source is active

- **GIVEN** there is no active focused source
- **WHEN** the delivery source list renders
- **THEN** the sources appear in the original report order
