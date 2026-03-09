## Why

The delivery alert rollup now supports severity filtering, but the controls do not show how many alerts each filter contains and the panel goes visually blank when the selected severity has no matches. Operators need lightweight filter counts and an explicit empty state so the rollup remains readable during triage.

## What Changes

- Add per-severity counts to the delivery alert filter controls.
- Highlight the active severity filter in the delivery panel.
- Show an explicit empty-state message when the selected severity has no matching alerts.

## Capabilities

### New Capabilities
- `delivery-alert-filter-counts`: show delivery rollup filter counts and empty states

### Modified Capabilities
- `delivery-alert-filtering`: operators can see alert volume per severity before switching filters
- `dashboard-serving`: delivery alert triage stays legible when a filter has no matches

## Impact

- Affected code: `libretto/src/components`, `libretto/src/lib`
- Affected behavior: delivery severity filters expose counts and empty states
- Affected tests: frontend unit tests and build validation
