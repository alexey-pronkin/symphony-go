## Context

Symphony already exposes compact delivery metrics, but the SCM portion is currently derived only
from local branch topology via `go-git`. That is useful for branch drift, stale work, and unmerged
branch counts, but it does not capture the hosted review state that actually blocks delivery in the
target workflows: GitHub pull requests, GitLab merge requests, and GitVerse change reviews.

The existing service shape is already close to what we need:

- `config.Config` parses `insights.scm_sources`
- `insights.Service` aggregates tracker and SCM metrics into the integral dashboard cards
- `libretto` consumes a stable JSON shape from `/api/v1/insights/delivery`

The next change should preserve that pipeline and only widen the SCM portion.

## Goals / Non-Goals

**Goals:**
- Keep local `go-git` branch inspection as the baseline SCM adapter.
- Add provider-specific API augmentation for GitHub, GitLab CE, and GitVerse.
- Report provider-review signals in a way that degrades cleanly when a source is misconfigured,
  unauthenticated, or partially unsupported.
- Keep the implementation read-only and safe for long-running operator use.

**Non-Goals:**
- Writing tracker or forge state back to GitHub, GitLab, or GitVerse.
- Replacing the current integral metric cards with a second dashboard model.
- Building a generic SCM SDK for every forge on the market.

## Decisions

### 1. Keep a single SCM inspector interface and introduce provider dispatch behind it

`insights.Service` already depends on an inspector interface, so the cleanest extension is to keep
that contract and route each source through a provider-aware implementation. That preserves the
existing aggregation logic and keeps the orchestration/runtime layers unchanged.

Alternative considered:
- Separate remote and local inspector pipelines.
  Rejected because it would duplicate warning handling and summary aggregation.

### 2. Treat provider APIs as augmentation on top of branch metrics, not a replacement

Each source should still report branch drift/staleness when `repo_path` is available. Provider API
data adds review-level metrics such as open change requests, approved changes, and failing checks.
This keeps the system useful for partially connected sources and for self-hosted environments where
API scopes may be restricted.

Alternative considered:
- Provider metrics only, no local fallback.
  Rejected because it would make local/offline and partial-auth setups much less useful.

### 3. Make degraded-provider behavior first-class

The delivery endpoint should never fail just because one forge token is missing or one provider API
is unavailable. Instead:

- source-level warnings stay attached to the affected source
- report-level warnings summarize degraded sources
- integral metrics continue to render from the available SCM and tracker signals

This is consistent with the existing monitoring and task-platform behavior.

### 4. Extend the JSON payload with review-specific SCM fields

The dashboard needs a few review/CI signals, not raw provider payloads. The source and total metric
shapes should add:

- `open_change_requests`
- `approved_change_requests`
- `failing_change_requests`
- `stale_change_requests`

Those fields are stable enough to support GitHub, GitLab, and GitVerse without leaking provider API
schemas into the frontend.

## Risks / Trade-offs

- [GitVerse API variance] → Keep GitVerse behind the same source-level warning/degradation model so
  unsupported fields reduce fidelity instead of breaking the report.
- [API fan-out per source] → Keep source queries bounded and only request the fields needed for the
  compact dashboard metrics.
- [Credential handling] → Resolve tokens from config/env in the config layer and never emit them in
  logs, snapshots, or dashboard payloads.
- [Metric comparability across providers] → Normalize into a small shared metric vocabulary rather
  than provider-native status taxonomies.
