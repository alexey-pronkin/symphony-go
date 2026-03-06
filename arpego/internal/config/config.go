// Package config provides typed access to WORKFLOW.md front matter per SPEC.md §6.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config is a typed view over a WORKFLOW.md front matter map.
// All getters apply defaults and env resolution as specified in SPEC.md §6.4.
type Config struct {
	raw map[string]any
}

// New wraps a parsed front matter map.
// raw may be nil (treated as empty map).
func New(raw map[string]any) Config {
	if raw == nil {
		raw = map[string]any{}
	}
	return Config{raw: raw}
}

// --- tracker ---

func (c Config) TrackerKind() string {
	return getString(c.section("tracker"), "kind", "")
}

func (c Config) TrackerEndpoint() string {
	def := "https://api.linear.app/graphql"
	return getString(c.section("tracker"), "endpoint", def)
}

func (c Config) TrackerAPIKey() string {
	raw := getString(c.section("tracker"), "api_key", "")
	return resolveVar(raw)
}

func (c Config) TrackerProjectSlug() string {
	return getString(c.section("tracker"), "project_slug", "")
}

func (c Config) TrackerActiveStates() []string {
	return getStringList(c.section("tracker"), "active_states", []string{"Todo", "In Progress"})
}

func (c Config) TrackerTerminalStates() []string {
	return getStringList(c.section("tracker"), "terminal_states",
		[]string{"Closed", "Cancelled", "Canceled", "Duplicate", "Done"})
}

// --- polling ---

func (c Config) PollIntervalMs() int64 {
	return getInt64(c.section("polling"), "interval_ms", 30_000)
}

// --- workspace ---

func (c Config) WorkspaceRoot() string {
	raw := getString(c.section("workspace"), "root", "")
	if raw == "" {
		return filepath.Join(os.TempDir(), "symphony_workspaces")
	}
	return expandPath(raw)
}

// --- hooks ---

func (c Config) HookAfterCreate() string  { return getString(c.section("hooks"), "after_create", "") }
func (c Config) HookBeforeRun() string    { return getString(c.section("hooks"), "before_run", "") }
func (c Config) HookAfterRun() string     { return getString(c.section("hooks"), "after_run", "") }
func (c Config) HookBeforeRemove() string { return getString(c.section("hooks"), "before_remove", "") }

func (c Config) HookTimeoutMs() int64 {
	v := getInt64(c.section("hooks"), "timeout_ms", 60_000)
	if v <= 0 {
		return 60_000
	}
	return v
}

// --- agent ---

func (c Config) MaxConcurrentAgents() int {
	return int(getInt64(c.section("agent"), "max_concurrent_agents", 10))
}

func (c Config) MaxTurns() int {
	return int(getInt64(c.section("agent"), "max_turns", 20))
}

func (c Config) MaxRetryBackoffMs() int64 {
	return getInt64(c.section("agent"), "max_retry_backoff_ms", 300_000)
}

// MaxConcurrentAgentsByState returns the per-state concurrency map.
// State keys are normalized (trimmed + lowercased). Invalid entries are ignored.
func (c Config) MaxConcurrentAgentsByState() map[string]int {
	raw, _ := c.section("agent")["max_concurrent_agents_by_state"].(map[string]any)
	out := make(map[string]int, len(raw))
	for k, v := range raw {
		n := toInt64(v, -1)
		if n > 0 {
			out[normalizeState(k)] = int(n)
		}
	}
	return out
}

// --- codex ---

func (c Config) CodexCommand() string {
	return getString(c.section("codex"), "command", "codex app-server")
}

func (c Config) CodexApprovalPolicy() string {
	return getString(c.section("codex"), "approval_policy", "")
}

func (c Config) CodexThreadSandbox() string {
	return getString(c.section("codex"), "thread_sandbox", "")
}

func (c Config) CodexTurnSandboxPolicy() string {
	return getString(c.section("codex"), "turn_sandbox_policy", "")
}

func (c Config) CodexTurnTimeoutMs() int64 {
	return getInt64(c.section("codex"), "turn_timeout_ms", 3_600_000)
}

func (c Config) CodexReadTimeoutMs() int64 {
	return getInt64(c.section("codex"), "read_timeout_ms", 5_000)
}

func (c Config) CodexStallTimeoutMs() int64 {
	return getInt64(c.section("codex"), "stall_timeout_ms", 300_000)
}

// --- server (extension) ---

func (c Config) ServerPort() int {
	return int(getInt64(c.section("server"), "port", 0))
}

// --- helpers ---

func (c Config) section(name string) map[string]any {
	m, _ := c.raw[name].(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func getString(m map[string]any, key, def string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

func getInt64(m map[string]any, key string, def int64) int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	return toInt64(v, def)
}

func toInt64(v any, def int64) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil {
			return def
		}
		return i
	}
	return def
}

func getStringList(m map[string]any, key string, def []string) []string {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case string:
		parts := strings.Split(t, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return def
}

// resolveVar expands $VAR_NAME env references. Returns empty string if the
// var is unset or empty (treated as missing per SPEC.md §5.3.1).
func resolveVar(s string) string {
	if strings.HasPrefix(s, "$") {
		name := s[1:]
		return os.Getenv(name)
	}
	return s
}

// expandPath expands ~ and $VAR in path-like values (SPEC.md §6.1).
// URIs and bare commands are left unchanged.
func expandPath(s string) string {
	s = resolveVar(s)
	if strings.HasPrefix(s, "~/") || s == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			s = filepath.Join(home, s[1:])
		}
	}
	return s
}

func normalizeState(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
