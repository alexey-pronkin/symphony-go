## Why

The CI workflow now scans project code and shipped images, but operators still
cannot see whether agent-produced workspace code has introduced obvious
high-severity risks during task execution.

## What Changes

- Add optional Trivy-backed workspace security scans to issue detail reads.
- Return a cached summary plus top findings from `GET /api/v1/{issue_identifier}`.
- Surface the workspace scan summary in the Libretto selected-issue panel.

## Impact

- Affected code: `arpego/internal/app`, `arpego/internal/server`, `arpego/internal/securityscan`, `libretto/src`
- Affected behavior: operators can inspect high/critical workspace findings per issue
- Affected tests: Go unit tests for scan parsing/server enrichment, frontend build/test validation
