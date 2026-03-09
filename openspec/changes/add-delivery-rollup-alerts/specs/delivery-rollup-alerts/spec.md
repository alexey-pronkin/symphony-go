## ADDED Requirements

### Requirement: Delivery trend analytics expose integral rollups
The system SHALL compute compact rollups from bounded delivery trend windows so operators can assess
whether delivery is improving or regressing without reading raw samples manually.

#### Scenario: Trend response includes backend-derived rollups
- **WHEN** a client requests delivery trend analytics
- **THEN** the response includes normalized rollups for score direction and warning pressure
- **AND** the rollups are derived from the same bounded sample set returned to the client

### Requirement: Delivery trend analytics expose alert summaries
The system SHALL evaluate a small set of alert thresholds on delivery trend data and return them in
the API response for dashboard rendering.

#### Scenario: Trend regression triggers alert output
- **WHEN** blocked work, failing checks, or delivery health regress beyond configured thresholds
- **THEN** the trend response includes alert entries with severity and operator-facing detail
