## Why

The delivery panel can now focus and summarize an SCM source, but the source list still preserves the original ordering even when a source is focused. Operators should not have to scan the list to find the highlighted row after explicitly selecting a source from an alert.

## What Changes

- Add a helper that reorders SCM sources so the focused source is rendered first.
- Keep the existing order stable for the non-focused sources.
- Use the reordered source list in the delivery panel while preserving highlight behavior.

## Capabilities

### New Capabilities
- `delivery-focused-source-priority`: render the focused SCM source at the top of the source list

### Modified Capabilities
- `delivery-source-alert-focus`: focused-source investigation becomes immediate in the source list
- `dashboard-serving`: the delivery panel keeps the selected source visually prominent

## Impact

- Affected code: `libretto/src/components`, `libretto/src/lib`
- Affected behavior: focused SCM sources move to the top of the delivery source list
- Affected tests: frontend unit tests and build validation
