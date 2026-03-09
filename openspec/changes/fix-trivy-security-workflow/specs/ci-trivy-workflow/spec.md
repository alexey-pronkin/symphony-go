## ADDED Requirements

### Requirement: Security workflow runs Trivy scans deterministically
The system SHALL execute repository, config, and image Trivy scans in CI with explicit failure
handling so scanner errors and findings are observable independently of SARIF upload behavior.

#### Scenario: Trivy scan produces SARIF
- **WHEN** a CI Trivy scan completes and writes a SARIF file
- **THEN** the workflow uploads the SARIF artifact
- **AND** fails the job only after upload if Trivy reported findings or errors

#### Scenario: Trivy scan fails before producing SARIF
- **WHEN** Trivy exits without creating a SARIF file
- **THEN** the workflow skips the SARIF upload step
- **AND** fails with the original Trivy exit status instead of a secondary missing-file error
