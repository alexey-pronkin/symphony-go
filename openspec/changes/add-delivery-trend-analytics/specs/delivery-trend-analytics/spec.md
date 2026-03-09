## ADDED Requirements

### Requirement: Delivery insights retain historical trend snapshots
The system SHALL persist normalized delivery metric snapshots so operators can inspect historical
trends in addition to the current point-in-time delivery report.

#### Scenario: Historical snapshots are queryable
- **WHEN** delivery trend analytics are enabled
- **THEN** the system stores timestamped delivery snapshots in the analytics store
- **AND** exposes trend queries over a bounded time window

### Requirement: Dashboard renders compact historical trend output
The system SHALL expose dashboard-oriented delivery trend data without requiring provider-specific or
database-specific frontend logic.

#### Scenario: Trend endpoint returns dashboard-ready series
- **WHEN** a client requests delivery trend analytics
- **THEN** the response includes compact time-series values for key delivery metrics
- **AND** the dashboard can render the trend data alongside the current live report
