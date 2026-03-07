## Why

Symphony now has a task platform and runtime dashboard, but operators still lack a compact way to judge delivery health across gitflow and tracker activity. Teams using GitHub, GitVerse, and self-hosted GitLab need a small set of strong integral metrics that summarize flow, merge readiness, and work health without requiring a full BI stack.

## What Changes

- Add a delivery-metrics service that combines task-platform signals with gitflow signals from configured SCM sources.
- Support provider-labeled SCM sources for `github`, `gitlab`, and `gitverse`, with the first implementation reading local git repositories so it works across hosted and self-hosted setups.
- Expose a dashboard-focused JSON API for integral delivery metrics, agile metrics, kanban metrics, and per-source gitflow metrics.
- Extend Libretto with a simple delivery-metrics dashboard section that highlights a few high-signal integral metrics and key supporting breakdowns.
- Extend config and docs with SCM source definitions, metric thresholds, and dashboard behavior.

## Capabilities

### New Capabilities
- `delivery-metrics-dashboard`: Delivery health metrics and dashboard API spanning task flow and SCM gitflow.

### Modified Capabilities

## Impact

- `SPEC.md`
- `arpego/internal/config/*`
- `arpego/internal/server/*`
- `arpego/internal/insights/*`
- `libretto/src/*`
- OpenSpec artifacts and docs for delivery metrics, SCM source config, and dashboard behavior
