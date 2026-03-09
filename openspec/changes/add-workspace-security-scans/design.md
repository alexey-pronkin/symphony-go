## Design

This slice keeps scanning out of the orchestrator control loop. Instead, the
HTTP issue-detail path enriches the existing issue payload with an optional
workspace scan result.

### Backend

- Add `security.workspace_scan` config for enablement, command, timeout, and cache TTL.
- Introduce a Trivy scanner service that executes `trivy fs` and parses JSON
  output into a compact summary plus a bounded set of findings.
- Wire the scanner into the HTTP server so `GET /api/v1/{issue_identifier}`
  attaches `workspace_scan` when enabled.

### Frontend

- Extend selected issue detail types with `workspace_scan`.
- Render a compact scan summary and top findings in the existing detail panel.

### Scope

- On-demand/cached issue-detail scans only.
- No automatic background scans or persistent finding history in this slice.
