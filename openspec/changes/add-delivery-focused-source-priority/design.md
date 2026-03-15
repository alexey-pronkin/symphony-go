## Overview

This slice stays entirely in Libretto. The current focus state already resolves to a source key and a source object. A small list-order helper is enough to lift the focused source to the top of the rendered list while leaving the remaining entries untouched.

## Decisions

- Only one focused source can exist at a time, so the helper moves at most one entry.
- If there is no focused source, the list order is unchanged.
- If the focused source is missing, the list order is unchanged.
- The helper keeps the relative order of all non-focused sources stable.

## Validation

- Add unit coverage for focused-source list prioritization.
- Run the existing frontend tests and production build.
