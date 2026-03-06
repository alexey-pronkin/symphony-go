# Nx Workspace

This repository uses Nx as a mixed-stack task runner across:

- `arpego/` for Go
- `libretto/` for Vite + React
- `scripts/` for Python via `uv`

## Install

```bash
npm install
```

The global CLI is optional after setup, but supported:

```bash
npm add --global nx
```

## Common Commands

```bash
npm run graph
npm run build
npm run lint
npm run test
npm run format
npm run format:check
```

## Project Targets

```bash
npx nx show projects
npx nx run arpego:lint
npx nx run libretto:build
npx nx run scripts:format
```
