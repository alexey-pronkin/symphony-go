## Context

Symphony already exposes runtime state, task CRUD, and Prometheus metrics, but operators still need a compact delivery-health view that combines task flow with gitflow health. The new metrics need to work across repositories hosted on GitHub, GitVerse, and self-hosted GitLab CE without forcing the first implementation to depend on provider APIs or network calls.

The current backend already has the right seams for this:

- task-platform services expose normalized tasks
- runtime snapshot data exposes current running and retrying work
- Libretto already polls a JSON API and can render richer dashboard sections

The main missing piece is a delivery-insights service that computes integral metrics from those inputs and exposes them in a dashboard-friendly shape.

## Goals / Non-Goals

**Goals:**

- add a backend delivery-insights service that combines task-flow metrics and SCM gitflow metrics
- support SCM source definitions labeled as `github`, `gitlab`, and `gitverse`
- implement the first SCM adapter by inspecting local git repositories so hosted and self-hosted remotes work the same way
- expose a compact JSON API with a few high-signal integral metrics plus supporting breakdowns
- render those metrics in Libretto with graceful degradation when SCM sources are missing or unavailable

**Non-Goals:**

- direct GitHub, GitLab, or GitVerse API integrations in this slice
- long-term analytics warehousing in ClickHouse for delivery metrics
- complex charts, historical trend storage, or per-user productivity scoring

## Decisions

### 1. Add a dedicated delivery-insights service in `arpego/internal/insights`

Why:

- keeps metric computation separate from server handlers and orchestration state
- allows task-flow and SCM metrics to evolve independently from the HTTP layer
- creates a clean place for future Bun/ClickHouse read models without changing the dashboard contract

Alternative considered:

- compute metrics directly in HTTP handlers
  - rejected because it would mix domain logic with transport code and make testing harder

### 2. Use provider-labeled local git sources instead of remote API adapters for the first slice

Why:

- works for GitHub, GitVerse, and self-hosted GitLab with the same implementation
- avoids coupling the first slice to provider tokens, rate limits, and API differences
- lets the dashboard stay useful in local, air-gapped, and self-hosted environments

Alternative considered:

- implement GitHub/GitLab provider APIs immediately
  - rejected because it adds auth and provider-specific complexity before the shared delivery-metrics model is stable

### 3. Expose a small set of integral metrics instead of raw dashboards first

Why:

- operators asked for a few strong metrics, not a wall of charts
- a compact score-based surface is faster to interpret during daily operation
- supporting detail metrics still remain available for context and debugging

Metrics chosen:

- `delivery_health_index`
- `flow_efficiency_index`
- `merge_readiness_index`
- `predictability_index`

Supporting breakdowns:

- agile task metrics: throughput, completion ratio, review load
- kanban task metrics: WIP, blocked ratio, aging work, backlog pressure
- SCM metrics: unmerged branches, stale branches, drift commits, ahead commits per source

### 4. Keep the dashboard API read-only and tolerant of missing sources

Why:

- delivery metrics are observability data, not transactional data
- missing SCM sources should degrade gracefully instead of breaking the dashboard
- this keeps the task platform and orchestration surfaces operational even when insights are partial

Alternative considered:

- fail the whole endpoint when one SCM source is unavailable
  - rejected because partial metrics are still useful

## Risks / Trade-offs

- [Local git inspection is not the same as remote PR/MR state] → document that this slice focuses on branch-flow health and keep provider API adapters as a follow-up
- [Integral scores can feel opaque if unsupported by detail] → include rationale/status labels and supporting breakdown metrics in the same payload
- [Task metrics are limited by the current task history model] → use current-state and timestamp signals now, and defer historical flow analytics to the ClickHouse slice
- [Reading multiple repositories can become slow] → keep source configuration explicit and collect only lightweight branch metadata in this slice
