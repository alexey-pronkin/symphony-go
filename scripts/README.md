# Scripts

`scripts/` is the Python automation root for Symphony.

## Role

- small tooling
- maintenance scripts
- repository automation
- support utilities that do not belong in the Go runtime or React frontend

## Notes For Agents

- Use `uv` for environment and dependency management.
- Keep scripts small, explicit, and task-focused.
- Prefer calling into shared APIs or documented interfaces instead of duplicating business logic here.

## Starter Commands

```bash
cd scripts
uv run python -V
```
