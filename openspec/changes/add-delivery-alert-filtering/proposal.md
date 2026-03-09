## Why

The delivery rollup now prioritizes alerts and supports source focus, but it still presents one fixed list. Operators need a quick way to narrow the rollup to only critical issues or to review all alerts again without losing the current interaction flow.

## What Changes

- Add a severity filter control to the delivery alert rollup.
- Support switching between all alerts, critical alerts only, and warning alerts only.
- Keep source focus behavior working from the filtered alert list.

## Capabilities

### New Capabilities
- `delivery-alert-filtering`: filter delivery rollup alerts by severity

### Modified Capabilities
- `delivery-rollup-alerts`: operators can narrow the alert list to the severity they want to review
- `delivery-source-alert-focus`: source focus remains available from filtered source-backed alerts

## Impact

- Affected code: `libretto/src/components`, `libretto/src/lib`
- Affected behavior: operators can review critical and warning alerts separately
- Affected tests: frontend unit tests and build validation
