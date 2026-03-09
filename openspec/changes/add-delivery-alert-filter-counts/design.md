## Overview

This slice stays inside the existing Libretto delivery rollup. It reuses the current derived alert list, adds a small count helper in `delivery-insights.ts`, and extends the panel header and filtered-list rendering.

## Decisions

- Counts are derived from the full rollup alert list before filtering.
- The `All` filter count reflects the total visible rollup size after deduplication and severity ordering.
- Empty-state messaging is tied to the active filter label so operators know whether there are no critical alerts or no warning alerts.
- No backend or API changes are required.

## Validation

- Extend the delivery insight unit tests for alert counts.
- Run the existing frontend tests and build.
