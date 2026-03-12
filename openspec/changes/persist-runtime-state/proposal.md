## Why

Arpego currently keeps retry queue entries, running-session metadata, and
continuation state in memory only. A process restart drops that state and
forces operators to reconstruct what was running, what was retrying, and which
sessions should resume.

## What Changes

- Add a persistent runtime-state store for retry queue entries and running
  session metadata.
- Restore persisted runtime state during startup before the first poll/dispatch
  cycle.
- Keep the existing snapshot and issue-detail surfaces consistent after
  restarts by serving restored retry/session data.

## Impact

- Affected code: `arpego/internal/orchestrator`, `arpego/internal/app`,
  `arpego/internal/config`, `arpego/internal/server`
- Affected behavior: runtime state survives process restarts instead of being
  memory-only
- Affected tests: orchestrator persistence/restore coverage plus HTTP snapshot
  regression checks
