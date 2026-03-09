## Why

Historical delivery points are useful, but operators still need a compact answer to "is this trend
getting worse?" Raw samples alone force humans to infer slope, volatility, and alert severity by
eye. The next step is to compute a few integral rollups and thresholds directly in the backend.

## What Changes

- Add delivery trend rollups for slope, volatility, and warning pressure over bounded windows.
- Add alert threshold evaluation for blocked work, failing checks, and delivery-health regression.
- Extend the delivery trend API with dashboard-ready rollups and alert summaries.
- Render a compact alert/rollup section in Libretto beside the existing trend cards.
- Update `SPEC.md`, `arpego/README.md`, and OpenSpec artifacts for alert semantics.

## Capabilities

### New Capabilities
- `delivery-rollup-alerts`: Backend-derived rollups and alert thresholds on top of delivery trend
  analytics.

### Modified Capabilities
- `delivery-trend-analytics`: Trend responses now include rollups and alert summaries.

## Impact

- `arpego/internal/insights`
- `arpego/internal/server`
- `libretto/src`
- `SPEC.md`
- `arpego/README.md`
