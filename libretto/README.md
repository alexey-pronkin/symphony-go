# Libretto

`libretto/` is the frontend root for Symphony.

## Role

- Vite + React application
- Default target for new frontend work
- Human-facing interface layer for dashboards, control surfaces, and workflow views

## Notes For Agents

- Follow the repo-level guidance in [`../AGENTS.md`](/Users/pav/Documents/git/github/symphony-go/AGENTS.md).
- Keep UI work token-efficient by documenting component intent and behavior in focused notes instead of large prose dumps.
- Prefer React and Vite docs over legacy implementation references when starting new frontend work.

## Starter Commands

```bash
cd libretto
npm install
npm run dev
```

## Runtime Dashboard

Libretto now provides a Symphony operator dashboard for:

- runtime summary from `GET /api/v1/state`
- running and retrying issue queues
- selected issue detail from `GET /api/v1/{issue_identifier}`
- manual refresh via `POST /api/v1/refresh`
- delivery insights with a compact alert rollup for high-priority risks
- severity filters for the delivery alert rollup

## API Configuration

The frontend reads `VITE_SYMPHONY_API_BASE_URL`.

- unset: request same-origin `/api/v1/*`
- set: request the configured Symphony API origin

Example:

```bash
cd libretto
VITE_SYMPHONY_API_BASE_URL=http://127.0.0.1:18080 npm run dev
```

## Validation

```bash
cd libretto
npm test
npm run lint
npm run build
```
