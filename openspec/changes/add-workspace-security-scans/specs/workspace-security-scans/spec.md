## ADDED Requirements

### Requirement: Workspace Security Scan Summary

When workspace security scanning is enabled, the runtime MUST be able to attach
an optional workspace scan summary to issue detail responses.

#### Scenario: Issue detail includes workspace scan findings

- **WHEN** `GET /api/v1/{issue_identifier}` is requested for an issue with a workspace
- **AND** workspace scanning is enabled
- **THEN** the response includes a `workspace_scan` object with scan status,
  summary counts, and a bounded list of findings

#### Scenario: Scanner is unavailable

- **WHEN** workspace scanning is enabled but the configured scanner cannot run
- **THEN** the response still succeeds
- **AND** `workspace_scan.status` reports the degraded state without failing the issue detail request
