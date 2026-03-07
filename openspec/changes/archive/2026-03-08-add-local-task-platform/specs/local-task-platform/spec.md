## Requirement: Local tracker mode loads tasks from a file-backed store

When `tracker.kind` is `local`, the system SHALL read tasks from a local task store file instead of querying Linear.

#### Scenario: Local tracker uses YAML task storage
- **GIVEN** `tracker.kind` is `local`
- **AND** `tracker.file` points to a valid task store
- **WHEN** the orchestrator fetches candidate issues
- **THEN** it loads tasks from the file
- **AND** returns normalized issue records using the same dispatch fields used for Linear-backed issues

#### Scenario: Local tracker defaults the store path
- **GIVEN** `tracker.kind` is `local`
- **AND** `tracker.file` is omitted
- **WHEN** the service starts
- **THEN** it uses `TASKS.yaml` next to the selected `WORKFLOW.md`

## Requirement: Local task platform exposes task CRUD APIs

The HTTP server SHALL expose task list and mutation endpoints when the active tracker is local.

#### Scenario: List local tasks
- **GIVEN** the active tracker is local
- **WHEN** a client sends `GET /api/v1/tasks`
- **THEN** the server returns all known tasks as normalized task records
- **AND** includes summary counts by state

#### Scenario: Create a local task
- **GIVEN** the active tracker is local
- **WHEN** a client sends `POST /api/v1/tasks` with a valid title and initial task fields
- **THEN** the server persists a new task in the local task store
- **AND** returns the created normalized task record

#### Scenario: Update a local task
- **GIVEN** the active tracker is local
- **WHEN** a client sends `PATCH /api/v1/tasks/MT-101` with updated state or content fields
- **THEN** the server persists the change to the local task store
- **AND** returns the updated normalized task record

#### Scenario: Task CRUD is unavailable for non-local trackers
- **GIVEN** the active tracker is not local
- **WHEN** a client calls a local task platform endpoint
- **THEN** the server returns a JSON error response explaining that the local task platform is unavailable
