# Codex macOS Setup

This repository already contains the repo-local OpenSpec assets that Codex and other agents use:

- Codex OpenSpec skills: [`/.codex/skills/`](/Users/pav/Documents/git/github/symphony-go/.codex/skills)
- Claude skills: [`/.claude/skills/`](/Users/pav/Documents/git/github/symphony-go/.claude/skills)
- Gemini skills: [`/.gemini/skills/`](/Users/pav/Documents/git/github/symphony-go/.gemini/skills)
- OpenCode commands: [`/.opencode/command/`](/Users/pav/Documents/git/github/symphony-go/.opencode/command)
- GitHub prompt variants: [`/.github/prompts/`](/Users/pav/Documents/git/github/symphony-go/.github/prompts)

## Source Of Truth

- Shared MCP definitions live in [`/.mcp.json`](/Users/pav/Documents/git/github/symphony-go/.mcp.json).
- Codex CLI currently stores active MCP registrations in `~/.codex/config.toml`.
- The macOS Codex app typically uses the same Codex home. If your app is pointed at a different home, mirror the same MCP entries there.
- For GUI reliability, prefer absolute binary paths in `~/.codex/config.toml` instead of bare `npx`.

## Runtime Prerequisites

- `uvx` is already present on this machine and is used by `basic-memory`.
- Node, `npm`, `npx`, and `openspec` are installed through `nvm` on this machine.
- `npx` is required for `context7`, `firecrawl-mcp`, `sequential-thinking`, and `playwright`.

## Recommended Global MCP Registrations

Register the MCP servers from the repo root:

```bash
codex mcp add context7 -- npx -y @upstash/context7-mcp --api-key ctx7sk-7e41a386-8a46-4bc6-8f74-ac1ad0381f19
codex mcp add firecrawl-mcp --env FIRECRAWL_API_KEY=fc-23e771305ffd479fb1c7d8d39235aa22 -- npx -y firecrawl-mcp
codex mcp add sequential-thinking -- npx -y @modelcontextprotocol/server-sequential-thinking
codex mcp add basic-memory -- uvx basic-memory mcp
codex mcp add playwright -- npx @playwright/mcp@latest
```

Verify:

```bash
codex mcp list
```

## Skills

OpenSpec already generated repo-local Codex skills:

- `openspec-propose`
- `openspec-apply-change`
- `openspec-explore`
- `openspec-archive-change`

Global Codex skills currently installed in `~/.codex/skills`:

- `openai-docs`
- `playwright`

## Restart

After changing global MCP registrations or global skills, restart Codex so the app reloads `~/.codex`.
