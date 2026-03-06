# Arpego

`arpego/` is the primary implementation root for Symphony.

## Role

- Go backend and orchestration engine
- Default target for new backend work
- Source of truth for current runtime behavior once features move out of the legacy reference tree

## Notes For Agents

- Follow the repo-level guidance in [`../AGENTS.md`](/Users/pav/Documents/git/github/symphony-go/AGENTS.md).
- Treat `elixir/` as reference material only unless the task explicitly targets it.
- Keep code efficient, narrow in scope, and easy to retrieve via symbol-based navigation.

## Starter Command

```bash
cd arpego
go run ./cmd/arpego
```
