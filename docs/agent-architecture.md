# Agent Architecture

## Goals

- Keep agent workflows token-efficient.
- Reuse shared capabilities across Codex, Claude, Gemini, and OpenCode.
- Make both code and specs retrievable with narrow filters before large context is loaded.

## Canonical Sources

Use these sources in this order:

1. Specs in `openspec/` and other repo docs
2. Project memory in `basic-memory`
3. Code graph and LSP-backed code navigation
4. MCP tools for current external documentation

The rule is simple:

- specs are the source of truth for intended behavior
- project memory is the source of truth for durable operating knowledge
- code is the source of truth for current implementation

## Retrieval Layers

### Spec-RAG

Specs and notes should be stored as Markdown with YAML front matter so agents can filter before reading.

Recommended metadata fields:

```yaml
title: Orchestrator scheduling
kind: spec
area: orchestrator
component: scheduler
language: go
framework: stdlib
service: symphony
status: active
tags:
  - retries
  - polling
  - concurrency
updated: 2026-03-06
```

Use these fields to answer questions like:

- only show Go backend specs
- only show React frontend notes
- only show Python scripting conventions
- only show scheduler-related docs

### Code-RAG

Use structural retrieval rather than dumping files:

- OpenCode LSP for symbols, references, definitions, and workspace navigation
- ripgrep for text retrieval
- focused file reads for implementation details
- summaries in memory/spec docs for top-level retrieval

The intended query flow is:

1. filter to relevant docs and memory by metadata
2. locate code symbols via LSP or graph
3. read only the files and functions needed for the task

## Stack

- Backend: Go in `arpego/`
- Scripting: Python with `uv` in `scripts/`
- Frontend: Vite + React in `libretto/`
- Elixir is legacy reference material only

Implications:

- Specs should tag backend notes with `language: go`
- Script and tooling notes should tag with `language: python` and `tooling: uv`
- Frontend notes should tag with `language: typescript` and `framework: react` or `tooling: vite`

## Documentation Rules

- Prefer Mermaid for diagrams in Markdown.
- Use PlantUML for richer UML when Mermaid is insufficient.
- Keep every long doc front-loaded with:
  - one-paragraph summary
  - explicit metadata
  - links to adjacent docs

## Practical Rule

Before adding a new note or spec, ask:

- Can this be a focused document instead of a long chapter?
- Can an agent filter to it with metadata?
- Can another agent reuse it without restating it?

If not, restructure it.
