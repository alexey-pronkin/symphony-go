## Why

The delivery panel can focus a source from source-backed alerts, but the active focus notice is still generic. Operators need the notice itself to identify which repository is selected and summarize the key merge-readiness risks before scanning the full source list.

## What Changes

- Add a helper to resolve the currently focused delivery source object.
- Render a focused-source summary in the delivery panel notice with repository identity and key metrics.
- Keep the existing clear-focus action alongside the richer summary.

## Capabilities

### New Capabilities
- `delivery-focus-summary`: summarize the focused SCM source in the delivery panel notice

### Modified Capabilities
- `delivery-source-alert-focus`: focused-source state now exposes concrete source identity and risk counts
- `dashboard-serving`: operators can understand focused-source context without scanning the full source list

## Impact

- Affected code: `libretto/src/components`, `libretto/src/lib`
- Affected behavior: focused delivery sources surface a compact summary
- Affected tests: frontend unit tests and build validation
