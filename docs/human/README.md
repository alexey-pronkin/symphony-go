# Human Flow

This document describes the manual operator flow for running Symphony without
relying on tracker automation.

## Purpose

Use the human flow when:

- you want to validate a new workflow by hand
- you want to prepare a task before giving it to an agent
- you need a stable handoff format for a human contributor
- you want a repeatable path for local testing and demos

## Files

- [`HUMAN_TASK.md`](/Users/pav/Documents/git/github/symphony-go/HUMAN_TASK.md): task template for a single human-owned task
- `WORKFLOW.md`: runtime configuration and orchestration prompt

## Recommended Flow

1. Create or update `WORKFLOW.md`.
   Keep tracker, storage, workspace, and server settings aligned with the task you want to run.

2. Copy [`HUMAN_TASK.md`](/Users/pav/Documents/git/github/symphony-go/HUMAN_TASK.md) for the task you want to execute.
   Fill in the task identifier, goal, scope, constraints, validation plan, and handoff notes.

3. Prepare the environment.
   Start required dependencies such as PostgreSQL, ClickHouse, or monitoring services, and export any required environment variables.

4. Start Arpego.
   Example:
   ```bash
   cd arpego
   go run ./cmd/arpego ../WORKFLOW.md
   ```

5. Use the runtime API or Libretto to observe state.
   Useful endpoints:
   ```bash
   curl -sS http://127.0.0.1:18080/api/v1/state | jq
   curl -sS http://127.0.0.1:18080/api/v1/tasks | jq
   curl -sS http://127.0.0.1:18080/api/v1/<issue_identifier> | jq
   ```

6. Execute the task manually.
   Work inside the intended task workspace, not inside the source repo when the workflow expects workspace isolation.

7. Record results back into the task file.
   Update status, validation output, blockers, and follow-up work so the next human or agent gets a clean handoff.

8. Archive or promote the result.
   If the change is complete, archive the matching OpenSpec change. If more work is needed, keep the task file as the source of truth for the next iteration.

## Minimal Operator Checklist

- `WORKFLOW.md` is present and points at the intended services
- the task is written down in `HUMAN_TASK.md`
- required services and env vars are available
- runtime state is observable through API or dashboard
- validation commands are defined before implementation starts
- final outcome and evidence are written back to the task file

## Notes

- Prefer short, explicit task files over long prose.
- Keep acceptance criteria measurable.
- If a task changes orchestration, tracker behavior, workspace lifecycle, or observability, update `SPEC.md` and OpenSpec artifacts in the same change.
