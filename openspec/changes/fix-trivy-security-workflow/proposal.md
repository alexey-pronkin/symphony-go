## Why

The GitHub Security workflow is currently unstable. Trivy exits before producing SARIF in some
cases, which causes secondary upload failures and hides the actual scanning result. The workflow
needs a deterministic Trivy invocation path for repo, config, and image scans.

## What Changes

- Replace the Trivy action wrapper with explicit Trivy CLI invocations in GitHub Actions.
- Upload SARIF only when a scan output file exists.
- Fail the job on Trivy findings or scanner errors after artifact handling.
- Pin the Trivy setup action and CLI version to the currently documented release.

## Capabilities

### New Capabilities
- `ci-trivy-workflow`: Stable repository, config, and image scanning in GitHub Actions.

## Impact

- `.github/workflows/security.yml`
- `trivy.yaml`
