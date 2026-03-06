# Symphony — Codex

@AGENTS.md

## OpenSpec First

Use OpenSpec as the default workflow for planning and implementation in this repository.

OpenSpec philosophy:

- fluid, not rigid
- iterative, not waterfall
- easy, not complex
- built for brownfield, not just greenfield
- scalable from personal projects to enterprises

Default workflow:

1. `/opsx:propose "idea"` to create the change folder and artifacts
2. `/opsx:apply` to implement tasks from the approved artifacts
3. `/opsx:archive` to archive the completed change and sync specs

Default artifact set:

- `proposal.md`
- `specs/`
- `design.md`
- `tasks.md`

Treat these artifacts as the scoped implementation context. Prefer reading the active change
artifacts over reconstructing intent from chat history.

## Commands

Start with:

```text
/opsx:propose "your idea"
```

Expanded workflows are available if the project enables them through OpenSpec profile updates:

- `/opsx:new`
- `/opsx:continue`
- `/opsx:ff`
- `/opsx:verify`
- `/opsx:sync`
- `/opsx:bulk-archive`
- `/opsx:onboard`

## Practical Rules

- Do not treat OpenSpec as a rigid phase gate. Artifacts may be updated iteratively.
- Prefer updating the active proposal, specs, design, or tasks instead of carrying critical decisions only in chat.
- For brownfield changes, anchor work in the existing codebase and use specs to clarify deltas, not to restate the whole system.
- Keep artifact context compact and filterable so other agents can reuse it with low token cost.

## Project Defaults

- Backend: `arpego/`
- Frontend: `libretto/`
- Python tooling: `scripts/`
- Memory source of truth: `basic-memory`
- Legacy reference only: `elixir/`

## Quick Start

OpenSpec requires Node.js `20.19.0+`.

Typical setup:

```bash
openspec init
openspec update
```

In this repo, OpenSpec is already initialized. Use the generated `/opsx:*` commands and keep the
agent guidance fresh with `openspec update` when the OpenSpec package changes.
