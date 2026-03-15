## Overview

This slice stays entirely in Libretto. The source focus state already resolves to a source key; this change adds a small helper that turns that key back into the source object so the panel can render a compact summary card above the source list.

## Decisions

- The summary shows source name, provider kind, main branch, repo path, merge readiness, failing change requests, and stale change requests.
- The summary only renders when the focused source still exists in the latest report.
- The clear-focus action remains in the summary container so focus exit stays one click away.

## Validation

- Add unit coverage for the focused-source lookup helper.
- Run the existing frontend tests and production build.
