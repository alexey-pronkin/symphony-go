# Symphony

This repository has two primary roles:

- `SPEC.md` is the language-agnostic contract for Symphony.
- `elixir/` is a legacy reference implementation and inspiration source, not the default target for new work.

## Working Rules

- Keep implementation changes aligned with [`SPEC.md`](/Users/pav/Documents/git/github/symphony-go/SPEC.md).
- An implementation may extend the spec, but it must not conflict with it.
- If behavior changes meaningfully affect orchestration, workflow config, workspace lifecycle, agent execution, tracker integration, or observability, update the spec in the same change where practical.
- Prefer narrow, behavior-focused changes over broad refactors.

## OpenSpec

- OpenSpec is initialized in [`openspec/`](/Users/pav/Documents/git/github/symphony-go/openspec).
- Project context for OpenSpec lives in [`openspec/config.yaml`](/Users/pav/Documents/git/github/symphony-go/openspec/config.yaml).
- Repo-local OpenSpec integrations already exist for Codex, Claude, Gemini, OpenCode, GitHub prompts, Agent, KiloCode, and Qwen.
- Treat the generated wrappers under `.codex/skills/openspec-*`, `.opencode/command/`, `.claude/skills/`, `.gemini/skills/`, and `.github/prompts/` as generated integration assets. Do not hand-edit them unless you are intentionally updating the OpenSpec tool wiring.
- Use OpenSpec as the default artifact-guided workflow for feature and change work.
- OpenSpec philosophy for this repo:
  - fluid, not rigid
  - iterative, not waterfall
  - easy, not complex
  - built for brownfield, not just greenfield
  - scalable from personal projects to enterprises
- Default command flow:
  - `/opsx:propose "idea"`
  - `/opsx:apply`
  - `/opsx:archive`
- The active change folder and its artifacts are the primary scoped context for implementation.
- Codex-specific OpenSpec notes live in [`CODEX.md`](/Users/pav/Documents/git/github/symphony-go/CODEX.md).

## Codex And MCP

- Shared MCP server definitions for this repo live in [`/.mcp.json`](/Users/pav/Documents/git/github/symphony-go/.mcp.json).
- Codex currently reads active MCP registrations from the user-level Codex home, not from the repo. Use the shared manifest as the source of truth and sync it into `~/.codex/config.toml` or via `codex mcp add`.
- Repo-specific Codex setup notes live in [`docs/codex-macos.md`](/Users/pav/Documents/git/github/symphony-go/docs/codex-macos.md).

## Memory And Docs

- `basic-memory` is the current shared project memory system and should be treated as the source of truth for durable cross-session knowledge.
- Agent-specific memory features are secondary caches, not the canonical record.
- Put durable decisions, workflow conventions, debugging findings, and architecture notes into project memory in Markdown.
- Record OpenSpec workflow conventions and change-level lessons in project memory so future agents can reuse them without rereading full chats.
- For structured metadata and filtering, use YAML front matter on Markdown notes and spec artifacts.
- Prefer YAML over TOML for specs and memory metadata because it handles nested structures, lists, and multiline text more naturally.
- Use TOML for tool configuration only, such as Codex or Gemini config files.
- Use Mermaid for diagrams embedded in Markdown docs and memory notes.
- Use PlantUML when Mermaid is not expressive enough for sequence, class, or deployment diagrams.

## Target Stack

- Primary backend language: Go
- Scripting and automation: Python via `uv`
- Frontend: Vite + React
- Implementation roots:
  - `arpego/` for the Go backend
  - `libretto/` for the frontend
  - `scripts/` for Python tooling
- OpenCode LSP and related editor/runtime features should be reused where they improve navigation, code intelligence, and implementation speed.

## Token And Cache Efficiency

- Optimize prompts, memory, and docs for low token usage and high reuse.
- Prefer retrieval over repetition. Store durable knowledge in specs and project memory, then retrieve only the relevant subset for the current task.
- Keep canonical facts in one place. Link or reference them instead of duplicating them across prompts, docs, and agent memories.
- Prefer compact, structured metadata over long prose when the goal is filtering or routing context.
- Agents should reuse shared project capabilities where practical:
  - `basic-memory` for durable memory
  - OpenSpec artifacts for scoped implementation context
  - OpenCode LSP for symbol navigation and code intelligence
  - MCP tools for current docs, research, and browser automation

## Retrieval Conventions

- Design specs and memory notes so they are filterable before they are readable.
- Use YAML front matter on Markdown notes and spec artifacts for retrieval keys such as:
  - `area`
  - `component`
  - `language`
  - `framework`
  - `service`
  - `kind`
  - `status`
  - `tags`
- For language- or tool-specific material, set `language` and `framework` explicitly so agents can pull only the relevant context.
- Put summaries at the top of long docs so retrieval can stop early when the summary is sufficient.
- Prefer small, focused notes over monolithic design documents.

## Implementation Notes

- Default to `arpego/`, `libretto/`, and `scripts/` for new implementation work.
- Use [`elixir/`](/Users/pav/Documents/git/github/symphony-go/elixir) as reference material only unless a task explicitly targets it.
- If you are working in [`elixir/`](/Users/pav/Documents/git/github/symphony-go/elixir), also follow [`elixir/AGENTS.md`](/Users/pav/Documents/git/github/symphony-go/elixir/AGENTS.md).
- Workspace safety is a core invariant. Agent execution must stay inside issue workspaces, not the source repo.
- Preserve orchestrator semantics around claiming, retries, reconciliation, cleanup, and dynamic `WORKFLOW.md` reloads.

## Multi-Agent Coordination

Claude, Codex, and Gemini CLI work on this repo concurrently. Conventions:

- `Claude Code` — architecture review, spec alignment, PR review, research via MCP tools, and structured execution with OpenSpec artifacts.
- `Codex` — implementation tasks and repo-wide edits.
- `Gemini CLI` — broad codebase exploration and large-context analysis.

Coordination rules:
- One agent per branch at a time. Do not push to a branch another agent is actively working on.
- Use Linear ticket state as the coordination signal: `In Progress` = claimed, `Merging` = PR open.
- Run the validation commands that match the touched stack before creating a PR.
- Commit messages must include a `Co-authored-by` trailer identifying the agent.

## Knowledge Refresh (1-day cadence)

Before starting work each session, check whether agent tool docs are more than 1 day old.
If stale, fetch updates from:

- Claude Code: `https://code.claude.com/docs/llms.txt`
- Codex CLI: `https://raw.githubusercontent.com/openai/codex/main/README.md`
- Gemini CLI: `https://raw.githubusercontent.com/google-gemini/gemini-cli/main/README.md`
- OpenSpec: `https://raw.githubusercontent.com/Fission-AI/OpenSpec/main/README.md`

Save key behavior changes to project memory first, then mirror only short operational summaries into agent-local memory if useful.
Do not assume tool behavior is stable across versions — check if in doubt.

## Validation

- For spec-only or tooling-only changes, validate the affected docs and config files directly.
- For implementation changes, run the narrowest useful checks first, then the project gate for the touched stack.
- The legacy Elixir reference implementation still uses:

```bash
cd elixir
make all
```
