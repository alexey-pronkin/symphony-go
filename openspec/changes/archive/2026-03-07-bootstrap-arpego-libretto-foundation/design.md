## Context

`libretto/` is still the default Vite + React starter while `arpego/` now
exposes the MVP runtime API. The next frontend slice should establish a usable
operator dashboard without introducing router/state-library overhead before the
product surface is better understood.

## Goals / Non-Goals

**Goals:**
- Replace the starter UI with a Symphony-specific dashboard shell.
- Read runtime state from the existing Arpego API and present it clearly.
- Support issue selection, detail loading, manual refresh, and periodic polling.
- Keep the frontend foundation typed, small, and easy to extend.

**Non-Goals:**
- Authentication, RBAC, or multi-user workflow.
- A full design system or routing architecture.
- Editing issues or mutating tracker state beyond the existing refresh endpoint.

## Decisions

### Use a small typed fetch client with same-origin default

Libretto will read `import.meta.env.VITE_SYMPHONY_API_BASE_URL` and default to
an empty base URL, so production can use same-origin serving while local
development can point at a different Arpego port. This avoids adding a query
library before the API surface is larger.

### Build the dashboard as one page with local hooks

The feature only needs a summary dashboard and a selected-issue detail panel.
Local state plus a couple of focused hooks keeps the code size small and avoids
premature routing/global-store choices.

### Poll state, fetch detail on demand

`/api/v1/state` will load on mount and refresh on a fixed client-side interval.
Issue detail will load only when an operator selects a running or retrying row.
This keeps the default request volume low while still supporting drill-down.

### Establish a real visual language now

The starter styles will be replaced with a deliberate Symphony dashboard look:
custom CSS variables, a brighter control surface, and responsive panels for
desktop and mobile. The foundation should feel product-specific, not like the
default Vite template.

## Risks / Trade-offs

- [Backend shape drift] -> Keep the client types aligned with the current
  Arpego JSON payloads and validate with build-time TypeScript checks.
- [Polling noise] -> Use a modest default interval and explicit manual refresh
  rather than aggressive background requests.
- [No router yet] -> Keep components modular so a later route split can reuse
  the API hooks and view sections.
- [Separate frontend/backend origins in local dev] -> Use the API base URL env
  variable instead of hard-coding loopback assumptions.

## Migration Plan

1. Add frontend API types and fetch helpers for the current Arpego endpoints.
2. Replace the starter page with summary, list, and detail components.
3. Add loading/error/empty states and refresh interactions.
4. Validate with frontend build/lint and document local startup.

## Open Questions

- Whether Libretto should later be served by Arpego directly or deployed as a
  separate static asset build.
- Whether issue history/log surfaces should be added in the next frontend slice
  or left to a later observability feature.
