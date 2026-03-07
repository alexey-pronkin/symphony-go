## ADDED Requirements

### Requirement: Typed config getters with defaults
The system SHALL expose typed getters for all SPEC.md Section 6.4 fields. When a WORKFLOW.md value is absent, the getter SHALL return the specified default.

#### Scenario: Missing poll interval defaults to 30000
- **WHEN** `polling.interval_ms` is absent from front matter
- **THEN** the config layer returns 30000

#### Scenario: Missing workspace root defaults to system temp
- **WHEN** `workspace.root` is absent
- **THEN** the config layer returns `<os.TempDir()>/symphony_workspaces`

#### Scenario: Present value overrides default
- **WHEN** `agent.max_concurrent_agents` is set to 5 in front matter
- **THEN** the config layer returns 5

### Requirement: $VAR environment variable resolution
The system SHALL resolve values of the form `$VAR_NAME` in tracker API key and path fields by reading the named environment variable.

#### Scenario: $VAR resolves to env value
- **WHEN** `tracker.api_key` is `$LINEAR_API_KEY` and the env var is set
- **THEN** the resolved value is the env var content

#### Scenario: Empty $VAR treats key as missing
- **WHEN** `$VAR_NAME` resolves to empty string
- **THEN** the config layer treats the field as missing (absent)

### Requirement: ~ home directory expansion
The system SHALL expand a leading `~` in path fields to the current user home directory.

#### Scenario: Tilde expanded in workspace root
- **WHEN** `workspace.root` is `~/workspaces`
- **THEN** the resolved path is `<home>/workspaces`

### Requirement: Dispatch preflight validation
The system SHALL validate the config before starting dispatch each tick. Validation SHALL check: workflow loadable, `tracker.kind` present and supported, `tracker.api_key` non-empty after resolution, `tracker.project_slug` present, `codex.command` non-empty.

#### Scenario: Valid config passes preflight
- **WHEN** all required fields are present and valid
- **THEN** validation returns success

#### Scenario: Missing API key fails preflight
- **WHEN** `tracker.api_key` is absent or resolves to empty
- **THEN** validation returns a `missing_tracker_api_key` error

#### Scenario: Unsupported tracker kind fails preflight
- **WHEN** `tracker.kind` is not `linear`
- **THEN** validation returns an `unsupported_tracker_kind` error
