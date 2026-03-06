# OpenSpec Workflow

OpenSpec is the default planning and implementation workflow for this repository.

## Philosophy

- fluid, not rigid
- iterative, not waterfall
- easy, not complex
- built for brownfield, not just greenfield
- scalable from personal projects to enterprises

## Default Flow

```text
/opsx:propose "your idea"
/opsx:apply
/opsx:archive
```

Expected result:

- `proposal.md` for why and scope
- `specs/` for requirements and scenarios
- `design.md` for technical approach
- `tasks.md` for implementation checklist

## How To Use It Here

- Use OpenSpec artifacts as the primary implementation context.
- Update artifacts iteratively; do not treat them as rigid waterfall gates.
- For brownfield changes, focus artifacts on the delta from the existing codebase.
- Keep artifacts compact, structured, and filterable so multiple agents can reuse them efficiently.

## Expanded Workflow

If enabled through OpenSpec profile configuration, the expanded workflow may include:

- `/opsx:new`
- `/opsx:continue`
- `/opsx:ff`
- `/opsx:verify`
- `/opsx:sync`
- `/opsx:bulk-archive`
- `/opsx:onboard`

## Operational Notes

- OpenSpec works best with clean context and high-reasoning models.
- Refresh generated agent guidance with `openspec update` when the global OpenSpec package changes.
- OpenSpec supports multiple package manager ecosystems; this repo currently uses `pnpm` and `uv`.
