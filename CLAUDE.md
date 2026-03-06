# Symphony — Claude Code

@AGENTS.md

## Claude-Specific Setup

### MCP Tools

Project MCPs are configured in `.mcp.json` (gitignored, local only):

- **context7** — fetch up-to-date library and framework docs by library ID
- **firecrawl-mcp** — web research, crawl docs sites, extract structured content
- **sequential-thinking** — stepwise reasoning for complex architectural decisions
- **basic-memory** — persistent cross-session notes (separate from Claude auto-memory)
- **playwright** — browser automation for testing dashboards and HTTP endpoints

Use context7 before implementing anything that touches Elixir library APIs, Phoenix LiveView, or
the Codex app-server protocol. For new work in this repo, prioritize Go, React, Vite, and Python
tooling docs first. Prefer authoritative docs over training-data guesses.

### Memory

`basic-memory` is the canonical shared memory system for this project.
Claude auto-memory is useful, but it is not the source of truth.

Key memory locations:

- `~/.claude/projects/.../memory/MEMORY.md` — loaded every session (keep < 200 lines)
- Topic files (e.g. `debugging.md`, `patterns.md`) — loaded on demand
- Shared repo guidance: [`docs/memory.md`](/Users/pav/Documents/git/github/symphony-go/docs/memory.md)

When you discover something worth remembering (a tricky OTP pattern, a dialyzer workaround,
a test fixture convention), save it to project memory first and mirror only the short operational summary into Claude auto-memory if needed.

Use Markdown for notes, YAML front matter for filterable metadata, Mermaid for most diagrams, and PlantUML when a richer UML notation is needed.

Implementation defaults:

- `arpego/` for backend work
- `libretto/` for frontend work
- `scripts/` for Python tooling
- `elixir/` only for reference or explicit maintenance tasks

### Skills

OpenSpec skills are in `.claude/skills/` and auto-load when relevant:

- `/opsx:propose` — draft a spec-aligned change proposal before implementing
- `/opsx:apply` — execute an approved proposal's task list
- `/opsx:explore` — explore the codebase for a given concern
- `/opsx:archive` — archive a completed change

Codex skills are in `.codex/skills/` (used by Codex, not Claude directly, but readable for context):
`commit`, `land`, `debug`, `pull`, `push`, `linear`.

### Permissions

`.claude/settings.local.json` grants pre-approved `WebFetch` access to:
`raw.githubusercontent.com`, `docs.anthropic.com`, `code.claude.com`

For other domains needed during research, use firecrawl-mcp instead.

### PR Convention

PR body must follow `.github/pull_request_template.md` exactly.
Validate with: `mix pr_body.check --file /path/to/pr_body.md`
