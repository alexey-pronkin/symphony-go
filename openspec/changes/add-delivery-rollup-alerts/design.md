## Context

The current delivery insights panel already has all the raw signals needed for an alert rollup: summary card statuses, report-level warnings, SCM totals, and source warnings. This feature can stay entirely in the frontend by deriving alerts from the existing API payload instead of expanding the backend contract.

## Goals / Non-Goals

**Goals:**
- Surface a compact list of the most relevant delivery alerts near the top of the panel.
- Classify alerts into lightweight severity levels.
- Keep alert generation deterministic and fully derived from the current delivery report.

**Non-Goals:**
- Persist alert acknowledgements.
- Add backend-owned alert history.
- Introduce notification delivery or paging integrations.

## Decisions

Build rollup alerts in a frontend helper based on the existing `DeliveryInsights` payload.
Rationale: the data is already available to Libretto, and this keeps the slice small and product-focused.

Treat report warnings as warning-level alerts and failing SCM checks or risk-status summary cards as critical alerts.
Rationale: report warnings are already degraded conditions, while failing checks and risk-scored summary cards need stronger emphasis.

Deduplicate alerts by message and keep the list compact.
Rationale: operators need a readable rollup, not a copy of every warning surface in the panel.

Render the alert rollup above the detailed cards and trend views.
Rationale: the rollup should act as the operator's first read of the panel.

## Risks / Trade-offs

[Frontend-derived alerts could drift from future backend logic] -> Keep the derivation rules explicit in one helper with unit tests.

[Too many alerts can recreate the same scanning problem] -> Limit the rollup to the highest-priority unique items.
