# Dev Tooling

This repository uses version-controlled git hooks plus per-stack lint and format commands.

## Install Hooks

```bash
make hooks-install
```

This configures `git` to use [`.githooks/`](/Users/pav/Documents/git/github/symphony-go/.githooks).

## Repo Commands

```bash
make format
make format-check
make lint
```

## Stack Commands

### Go (`arpego/`)

```bash
make format-go
make lint-go
```

- Formatter: `gofmt`
- Linter: `golangci-lint`, with `go vet` fallback when `golangci-lint` is unavailable

### Frontend (`libretto/`)

```bash
npm --prefix libretto run format
npm --prefix libretto run format:check
npm --prefix libretto run lint
```

- Formatter: `prettier`
- Linter: `eslint`

### Python (`scripts/`)

```bash
cd scripts
uvx ruff format .
uvx ruff check .
```

- Formatter: `ruff format` via `uvx`
- Linter: `ruff check` via `uvx`
