## Context

The current delivery endpoint computes a useful point-in-time report, but it does not retain those
reports. That makes it impossible to answer basic operational questions such as whether blocked work
is rising, whether merge readiness is improving week over week, or whether retry pressure is
spiking after a workflow change.

ClickHouse is already in the stack and is a natural fit for high-volume historical analytics.

## Goals / Non-Goals

**Goals:**
- Persist normalized delivery snapshots at a bounded cadence.
- Query trend windows from ClickHouse without changing orchestrator correctness.
- Expose compact trend data that Libretto can render without a charting rewrite.

**Non-Goals:**
- Full BI/reporting workloads.
- User-defined dashboard builders.
- Replacing the existing point-in-time delivery endpoint.

## Decisions

- Store delivery snapshots as append-only ClickHouse rows keyed by source/window timestamp.
- Keep the current `/api/v1/insights/delivery` response and add a sibling trend endpoint rather than
  overloading one payload with both live and historical concerns.
- Return simple trend series sized for dashboard cards first; deeper analytics can build on the same
  store later.

## Risks / Trade-offs

- [Snapshot volume growth] → keep cadence bounded and define retention/partition strategy up front.
- [Metric drift between live and historical paths] → derive snapshots from the same normalized
  delivery report model used by the live endpoint.
- [Dashboard complexity] → start with compact trend cards instead of a full chart-heavy surface.
