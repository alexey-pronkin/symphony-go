## Why

The delivery dashboard already exposes detailed metrics, warnings, and trends, but operators still have to visually scan multiple cards and warning blocks to understand what needs attention first. A compact alert rollup is the next useful product slice because it turns the existing signals into an operator-prioritized summary.

## What Changes

- Add a delivery alert rollup section to the Libretto delivery insights panel.
- Derive alert items from degraded metric cards, report warnings, and SCM risk signals already present in the API payload.
- Present the highest-priority alerts first with lightweight severity cues.

## Capabilities

### New Capabilities
- `delivery-rollup-alerts`: summarize delivery risks into a compact operator alert list

### Modified Capabilities
- `dashboard-serving`: the dashboard highlights the most important delivery issues before the full metric breakdown
- `delivery-trend-analytics`: trend and warning data now feed a summary alert surface for operators

## Impact

- Affected code: `libretto/src/components`, `libretto/src/lib`
- Affected behavior: operators can see the most urgent delivery issues without scanning the full panel
- Affected tests: frontend unit tests and build validation
