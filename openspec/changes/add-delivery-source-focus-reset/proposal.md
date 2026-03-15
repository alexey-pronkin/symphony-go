## Why

The delivery panel can now focus an SCM source from a source-backed alert, but that focus behaves like a dead-end state. Operators need to clear or toggle focus directly so they can continue reviewing the source list without refreshing the page.

## What Changes

- Let source-backed alert actions toggle focus on and off.
- Add a clear-focus action to the focused-source notice in the delivery panel.
- Move source-focus state handling into small helper utilities with unit coverage.

## Capabilities

### New Capabilities
- `delivery-source-focus-reset`: toggle and clear focused SCM sources from the delivery panel

### Modified Capabilities
- `delivery-source-alert-focus`: source focus can now be cleared without refreshing the dashboard
- `dashboard-serving`: operators can move in and out of focused source review more cleanly

## Impact

- Affected code: `libretto/src/components`, `libretto/src/lib`
- Affected behavior: source focus in the delivery panel is reversible
- Affected tests: frontend unit tests and build validation
