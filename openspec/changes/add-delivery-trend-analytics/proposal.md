## Why

Symphony now exposes point-in-time delivery metrics, but operators still cannot see whether flow is
improving or regressing over time. Since ClickHouse is already present for runtime observability, the
next step is to persist delivery snapshots and expose trend-oriented analytics to Libretto.

## What Changes

- Persist periodic delivery metric snapshots into ClickHouse.
- Add backend trend queries for throughput, blocked work, retry pressure, and merge readiness over
  configurable windows.
- Extend the delivery dashboard with compact historical trend cards and sparkline-ready data.
- Update `SPEC.md` and docs for historical delivery analytics configuration and retention behavior.

## Capabilities

### New Capabilities
- `delivery-trend-analytics`: Historical delivery metrics snapshots and dashboard trend queries
  backed by ClickHouse.

### Modified Capabilities

## Impact

- `arpego/internal/insights`
- `arpego/internal/server`
- `libretto/src`
- `SPEC.md`
- `arpego/README.md`
