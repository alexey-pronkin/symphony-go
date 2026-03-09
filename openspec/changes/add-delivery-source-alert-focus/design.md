## Context

The existing rollup alerts are already derived in the frontend from the delivery report. The SCM source list is also rendered from the same report. That makes source focus a local UI-state feature: alerts can carry an optional source key, and the panel can highlight the matching source card when the operator selects it.

## Goals / Non-Goals

**Goals:**
- Mark source-backed alerts with enough metadata to locate the related SCM source.
- Let operators activate source-backed alerts from the rollup.
- Visually focus the matching source card.

**Non-Goals:**
- Add backend navigation identifiers.
- Persist source selection across refreshes.
- Filter the source list down to a single entry in this slice.

## Decisions

Represent source focus with a frontend-only source key derived from the existing source fields.
Rationale: the report already has a stable combination of source kind, name, and repo path.

Only make source-backed alerts interactive.
Rationale: some alerts summarize global delivery conditions and should stay informational.

Highlight the selected source card and add a small focus label in the panel.
Rationale: operators need explicit confirmation that the alert selected the intended source.

Reset focus when the report changes and the selected source no longer exists.
Rationale: stale selection should not survive incompatible report refreshes.

## Risks / Trade-offs

[Derived source keys could drift if source identity rules change] -> Keep key generation in a shared helper used by both alert derivation and panel rendering.

[Interactive alerts can blur the line between summary and detail] -> Limit interaction to source-backed alerts only and keep the focused state visually lightweight.
