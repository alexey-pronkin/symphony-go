## Context

The rollup alert list is already frontend-derived and ordered by severity. Filtering is therefore a pure presentation-layer feature. The cleanest implementation is a local filter state in the delivery panel that narrows the rendered alerts while leaving alert derivation and source focus unchanged.

## Goals / Non-Goals

**Goals:**
- Filter the alert rollup by severity.
- Preserve source focus actions on visible source-backed alerts.
- Keep the default view as the full prioritized alert list.

**Non-Goals:**
- Persist alert filter state across refreshes.
- Add text search over alert details.
- Change the backend delivery report contract.

## Decisions

Use a local enum-style filter with `all`, `critical`, and `warning`.
Rationale: the current rollup only exposes two severity levels, so a small fixed control is enough.

Apply filtering after alert derivation and ordering.
Rationale: derivation rules and priority ordering should stay canonical in one place.

Keep the existing alert cap after filtering.
Rationale: the rollup should remain compact even when many alerts of the same severity exist.

Render the filter controls next to the rollup header.
Rationale: operators should understand that the controls affect the summary alert list only.

## Risks / Trade-offs

[Filtering can hide currently focused source-backed alerts] -> Focus state remains local and source cards stay visible below the rollup.

[A fixed severity filter may be too limited later] -> This slice keeps the control intentionally narrow and can be extended without changing the backend.
