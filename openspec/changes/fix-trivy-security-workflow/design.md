## Context

The existing workflow delegates everything to `aquasecurity/trivy-action`. In practice that makes
scanner bootstrap failures and SARIF upload failures hard to distinguish. The CI path should expose
Trivy's real exit code while still uploading SARIF when available.

## Decisions

- Install the latest safe Trivy CLI directly from the GitHub release asset and checksum.
- Run `trivy fs`, `trivy config`, and `trivy image` directly in shell steps.
- Capture the Trivy exit status to `GITHUB_OUTPUT`, upload SARIF conditionally, and fail after
  upload.
- Keep the existing job split for repo, config, and image scanning.

## Risks / Trade-offs

- Workflow YAML is slightly longer, but much easier to debug from logs.
- If setup itself fails, the job still fails early, which is acceptable and clearer than missing
  SARIF follow-on errors.
