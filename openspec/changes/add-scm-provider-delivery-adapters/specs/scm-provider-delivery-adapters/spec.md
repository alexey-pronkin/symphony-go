## ADDED Requirements

### Requirement: Delivery metrics support provider-aware SCM sources
The system SHALL accept SCM source definitions that augment local git metrics with provider-aware
review and CI signals for GitHub, GitLab CE, and GitVerse.

#### Scenario: GitHub source contributes review metrics
- **WHEN** a delivery metrics source is configured for `kind: github` with valid provider metadata
- **THEN** the delivery report includes source metrics for open, approved, stale, and failing
  change requests

#### Scenario: GitLab CE source contributes merge-request metrics
- **WHEN** a delivery metrics source is configured for `kind: gitlab` with valid provider metadata
- **THEN** the delivery report includes normalized merge-request metrics in the same SCM source
  payload shape used for other providers

#### Scenario: GitVerse source degrades gracefully
- **WHEN** a delivery metrics source is configured for `kind: gitverse`
- **AND** provider metrics are partially unavailable
- **THEN** the delivery report still returns HTTP 200
- **AND** the affected source includes warning messages describing the degraded provider state

### Requirement: Delivery dashboard renders normalized provider metrics
The system SHALL expose provider-normalized SCM metrics through the delivery endpoint and dashboard
without requiring provider-specific frontend logic.

#### Scenario: Delivery report includes provider review totals
- **WHEN** the delivery insights endpoint is requested
- **THEN** the SCM totals include normalized counts for open, approved, stale, and failing change
  requests

#### Scenario: Dashboard remains usable when provider sources degrade
- **WHEN** one or more provider-backed SCM sources return warnings
- **THEN** the delivery dashboard still renders integral metrics and source cards
- **AND** visually indicates degraded SCM provider state
