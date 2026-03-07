---
tracker:
  kind: local
  project_slug: sym
  file: /app/TASKS.yaml
polling:
  interval_ms: 15000
workspace:
  root: /tmp/symphony_workspaces
agent:
  max_concurrent_agents: 2
  max_retry_backoff_ms: 60000
codex:
  command: "sh -lc 'printf \"{\\\"id\\\":1,\\\"result\\\":{}}\\n{\\\"id\\\":2,\\\"result\\\":{\\\"thread\\\":{\\\"id\\\":\\\"thread-1\\\"}}}\\n{\\\"id\\\":3,\\\"result\\\":{\\\"turn\\\":{\\\"id\\\":\\\"turn-1\\\"}}}\\n{\\\"method\\\":\\\"turn/completed\\\",\\\"params\\\":{}}\\n\"'"
server:
  port: 18080
---
# Symphony Compose Workflow

This compose profile uses the local task platform so the stack can start without
external tracker credentials. Replace the `codex.command` and tracker settings
for real environments.
