## Overview

This change keeps the delivery panel state local to Libretto and does not require backend changes. It extracts the source-focus transition rules into `delivery-insights.ts` so the component remains simple and the behavior stays unit-testable.

## Decisions

- Clicking the currently focused source-backed alert clears focus.
- Clicking a different source-backed alert moves focus to the new source.
- The focused-source notice includes a dedicated clear button so operators do not need to click an alert again to reset the panel.
- If the focused source disappears from the latest report, the focus resolves to `null`.

## Validation

- Add unit coverage for focus toggle and focus resolution helpers.
- Run the existing frontend tests and production build.
