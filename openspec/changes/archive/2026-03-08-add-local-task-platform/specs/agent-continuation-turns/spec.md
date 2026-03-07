## Requirement: Worker sessions reuse a live Codex thread for continuation turns

The system SHALL continue work on the same app-server process and `thread_id` for multiple successful turns inside one worker run.

#### Scenario: Successful turn continues on the same thread
- **GIVEN** a worker has started a Codex session for an active issue
- **WHEN** the current turn completes successfully
- **AND** the issue is still active
- **AND** the worker has not reached `agent.max_turns`
- **THEN** the worker starts another turn on the same `thread_id`
- **AND** it sends continuation guidance instead of resending the original workflow prompt

#### Scenario: Continuation stops when the issue becomes inactive
- **GIVEN** a worker completed a turn successfully
- **WHEN** the issue is refreshed and is no longer in an active state
- **THEN** the worker exits its in-process continuation loop without starting another turn

## Requirement: Prompt rendering receives the full normalized issue model

The workflow template SHALL receive the complete normalized issue object, not only a partial subset.

#### Scenario: Template can access optional issue fields
- **GIVEN** an issue includes description, priority, branch metadata, labels, blockers, and timestamps
- **WHEN** the workflow prompt is rendered
- **THEN** the template input includes those fields under `issue`
- **AND** lists and nested blocker objects remain iterable in the template
