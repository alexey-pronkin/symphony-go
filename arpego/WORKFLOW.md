---
tracker:
  kind: linear
  endpoint: http://127.0.0.1:1/graphql
  api_key: sample-token
  project_slug: sample-project
  active_states:
    - Todo
    - In Progress
  terminal_states:
    - Done
    - Canceled
    - Cancelled
workspace:
  root: ./tmp/workspaces
agent:
  max_concurrent_agents: 2
  max_retry_backoff_ms: 300000
codex:
  command: printf ''
  read_timeout_ms: 250
  turn_timeout_ms: 1000
  stall_timeout_ms: 1000
server:
  port: 18080
---
Work on issue {{ .Issue.identifier }} titled {{ .Issue.title }}.
