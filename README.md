# Symphony

Symphony turns project work into isolated, autonomous implementation runs, allowing teams to manage
work instead of supervising coding agents.

## Current Direction

This repository is using the language-agnostic [`SPEC.md`](SPEC.md) as the contract while moving
the active implementation toward:

- `arpego/` for the Go backend
- `libretto/` for the Vite + React frontend
- `scripts/` for Python automation via `uv`

The `elixir/` tree remains useful as a legacy reference and source of ideas, but it is not the
default implementation target for new work.

[![Symphony demo video preview](.github/media/symphony-demo-poster.jpg)](.github/media/symphony-demo.mp4)

_In this [demo video](.github/media/symphony-demo.mp4), Symphony monitors a Linear board for work and spawns agents to handle the tasks. The agents complete the tasks and provide proof of work: CI status, PR review feedback, complexity analysis, and walkthrough videos. When accepted, the agents land the PR safely. Engineers do not need to supervise Codex; they can manage the work at a higher level._

> [!WARNING]
> Symphony is a low-key engineering preview for testing in trusted environments.

## Running Symphony

### Requirements

Symphony works best in codebases that have adopted
[harness engineering](https://openai.com/index/harness-engineering/). Symphony is the next step --
moving from managing coding agents to managing work that needs to get done.

### Option 1. Make your own

Tell your favorite coding agent to build Symphony in a programming language of your choice:

> Implement Symphony according to the following spec:
> https://github.com/openai/symphony/blob/main/SPEC.md

### Option 2. Use our experimental reference implementation

Check out [elixir/README.md](elixir/README.md) for instructions on how to set up your environment
and run the Elixir-based Symphony implementation. You can also ask your favorite coding agent to
help with the setup:

> Set up Symphony for my repository based on
> https://github.com/openai/symphony/blob/main/elixir/README.md

### Project Layout

- [`arpego/`](arpego/README.md): Go-first implementation root
- [`libretto/`](libretto/README.md): Vite + React frontend
- [`scripts/`](scripts/README.md): Python scripting and automation via `uv`
- [`docs/memory.md`](docs/memory.md): memory and retrieval conventions
- [`docs/agent-architecture.md`](docs/agent-architecture.md): token-efficient multi-agent design
- [`docs/dev-tooling.md`](docs/dev-tooling.md): shared lint, format, and git hook workflow
- [`docs/nx.md`](docs/nx.md): Nx workspace and monorepo commands
- [`docs/docker-security.md`](docs/docker-security.md): Docker, Traefik, CrowdSec, and security scanning

---

## License

This project is licensed under the [Apache License 2.0](LICENSE).
