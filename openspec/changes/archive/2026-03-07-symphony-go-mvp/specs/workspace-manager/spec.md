## ADDED Requirements

### Requirement: Deterministic workspace path per issue
The system SHALL compute a workspace path as `<workspace.root>/<sanitized_identifier>` where the sanitized identifier replaces any character outside `[A-Za-z0-9._-]` with `_`.

#### Scenario: Path is deterministic for same identifier
- **WHEN** `ensure_workspace` is called twice for the same identifier
- **THEN** both calls return the same path

#### Scenario: Special characters are replaced with underscore
- **WHEN** an identifier contains `/` or spaces
- **THEN** those characters become `_` in the workspace directory name

### Requirement: Create or reuse workspace directory
The system SHALL create the directory if absent (`created_now=true`) or reuse it if present (`created_now=false`).

#### Scenario: New workspace triggers after_create hook
- **WHEN** the workspace directory does not exist before the call
- **THEN** `created_now` is true and `after_create` hook runs if configured

#### Scenario: Existing workspace does not rerun after_create
- **WHEN** the workspace directory already exists
- **THEN** `created_now` is false and `after_create` hook does not run

### Requirement: Root containment safety invariant
The system SHALL validate that the resolved workspace path has `workspace.root` as a directory prefix. Paths outside the root SHALL be rejected before any agent is launched.

#### Scenario: Path inside root is accepted
- **WHEN** workspace path starts with the configured workspace root
- **THEN** the workspace is valid and the agent may be launched

#### Scenario: Path outside root is rejected
- **WHEN** a computed path does not start with the workspace root (e.g., via symlink traversal)
- **THEN** the system returns an error and does not launch the agent

### Requirement: Lifecycle hooks execution
The system SHALL execute `after_create`, `before_run`, `after_run`, and `before_remove` hooks via `bash -lc <script>` with the workspace as cwd and `hooks.timeout_ms` timeout.

#### Scenario: before_run failure aborts attempt
- **WHEN** the `before_run` hook exits non-zero
- **THEN** the run attempt is aborted with an error

#### Scenario: after_run failure is logged and ignored
- **WHEN** the `after_run` hook exits non-zero
- **THEN** the failure is logged and the worker continues to completion

#### Scenario: Hook timeout enforced
- **WHEN** a hook runs longer than `hooks.timeout_ms`
- **THEN** the hook process is killed and the timeout error is handled per hook failure semantics
