## Why

The delivery rollup now surfaces the most important issues first, but source-related alerts are still dead ends. Operators need to move directly from an SCM alert into the source card that caused it so they can inspect the underlying repository metrics without searching manually.

## What Changes

- Attach source focus metadata to source-related delivery alerts.
- Let operators select a source directly from the alert rollup.
- Highlight the focused source in the SCM source list and show the focus state clearly.

## Capabilities

### New Capabilities
- `delivery-source-alert-focus`: jump from a delivery alert to the SCM source behind it

### Modified Capabilities
- `delivery-rollup-alerts`: source-backed alerts now support operator focus actions
- `dashboard-serving`: the delivery panel supports source-oriented investigation directly from the summary surface

## Impact

- Affected code: `libretto/src/components`, `libretto/src/lib`
- Affected behavior: operators can navigate directly from summary alerts to the relevant SCM source
- Affected tests: frontend unit tests and build validation
