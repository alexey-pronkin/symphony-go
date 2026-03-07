## Why

`arpego/` now exposes the MVP runtime API, but `libretto/` is still the default
Vite starter. Symphony needs a real frontend foundation so operators can inspect
running work, retries, token usage, and issue details without reading raw JSON.

## What Changes

- Replace the starter React screen with a Symphony dashboard shell.
- Add a typed client for `GET /api/v1/state`, `GET /api/v1/{id}`, and
  `POST /api/v1/refresh`.
- Render runtime summary cards, running/retrying lists, and an issue detail
  panel with loading and error states.
- Add frontend configuration for the API base URL and document local usage.
- Add frontend validation coverage for build and lint.

## Capabilities

### New Capabilities
- `runtime-dashboard`: Libretto renders the current Symphony runtime state,
  issue detail lookups, and a manual refresh control on top of the Arpego HTTP
  API.

### Modified Capabilities
- None.

## Impact

- `libretto/` application structure, styles, and API integration
- frontend docs and local runtime instructions
- local operator workflow for monitoring and refreshing Symphony runs
