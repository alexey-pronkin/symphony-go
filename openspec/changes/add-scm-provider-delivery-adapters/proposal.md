## Why

The current delivery metrics pipeline only inspects local git repositories, so it misses the review,
approval, and pipeline signals that actually determine merge readiness in GitHub, GitLab CE, and
GitVerse workflows. Symphony needs provider-aware SCM adapters so the dashboard can report a more
useful integral view of agile, kanban, and gitflow health across hosted and self-hosted forges.

## What Changes

- Add provider-aware SCM source configuration for API-backed delivery metrics.
- Introduce read-only SCM inspectors for GitHub, GitLab CE, and GitVerse sources with graceful
  degradation when credentials or provider APIs are unavailable.
- Extend delivery metrics aggregation with change-request, approval, and CI health signals in
  addition to the existing local branch metrics.
- Update the Libretto delivery dashboard to render the richer provider metrics without breaking the
  current local-repository path.
- Update `SPEC.md` and Arpego docs for provider source configuration and degraded-provider behavior.

## Capabilities

### New Capabilities
- `scm-provider-delivery-adapters`: Provider-aware SCM metrics for GitHub, GitLab CE, and GitVerse
  sources, including graceful degradation and dashboard output.

### Modified Capabilities

## Impact

- `arpego/internal/config`
- `arpego/internal/insights`
- `arpego/internal/server`
- `libretto/src`
- `SPEC.md`
- `arpego/README.md`
