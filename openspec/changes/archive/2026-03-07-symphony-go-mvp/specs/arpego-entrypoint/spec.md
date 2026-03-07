## MODIFIED Requirements

### Requirement: CLI wires all packages into running service
The entrypoint SHALL accept an optional positional WORKFLOW.md path argument and optional `--port` flag. It SHALL validate config at startup, run terminal cleanup, start the poll loop, and shut down gracefully on SIGINT/SIGTERM.

#### Scenario: Explicit workflow path used when provided
- **WHEN** `arpego /path/to/WORKFLOW.md` is run
- **THEN** the workflow is loaded from that path

#### Scenario: Default workflow path used when absent
- **WHEN** `arpego` is run without a path argument
- **THEN** the workflow is loaded from `./WORKFLOW.md`

#### Scenario: Startup fails cleanly on missing workflow
- **WHEN** the specified WORKFLOW.md does not exist
- **THEN** the process exits nonzero with a human-readable error message

#### Scenario: Graceful shutdown on SIGTERM
- **WHEN** SIGTERM is received while agents are running
- **THEN** in-flight HTTP requests drain (10s), the process exits 0
