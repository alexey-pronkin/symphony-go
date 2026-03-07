package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
)

func cfg(raw map[string]any) config.Config {
	return config.New(raw)
}

func nested(pairs ...any) map[string]any {
	m := map[string]any{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i].(string)] = pairs[i+1]
	}
	return m
}

// --- defaults ---

func TestDefaults_NilRaw(t *testing.T) {
	c := config.New(nil)
	if c.PollIntervalMs() != 30_000 {
		t.Fatalf("expected 30000, got %d", c.PollIntervalMs())
	}
	if c.MaxConcurrentAgents() != 10 {
		t.Fatalf("expected 10, got %d", c.MaxConcurrentAgents())
	}
	if c.CodexCommand() != "codex app-server" {
		t.Fatalf("unexpected codex.command: %q", c.CodexCommand())
	}
	if !strings.HasSuffix(c.WorkspaceRoot(), "symphony_workspaces") {
		t.Fatalf("unexpected workspace root: %q", c.WorkspaceRoot())
	}
	if c.HookTimeoutMs() != 60_000 {
		t.Fatalf("expected 60000, got %d", c.HookTimeoutMs())
	}
	if c.MaxRetryBackoffMs() != 300_000 {
		t.Fatalf("expected 300000, got %d", c.MaxRetryBackoffMs())
	}
}

func TestDefaults_ActiveStates(t *testing.T) {
	c := cfg(nil)
	states := c.TrackerActiveStates()
	if len(states) != 2 || states[0] != "Todo" || states[1] != "In Progress" {
		t.Fatalf("unexpected active states: %v", states)
	}
}

func TestDefaults_TerminalStates(t *testing.T) {
	c := cfg(nil)
	states := c.TrackerTerminalStates()
	want := []string{"Closed", "Cancelled", "Canceled", "Duplicate", "Done"}
	for i, w := range want {
		if i >= len(states) || states[i] != w {
			t.Fatalf("unexpected terminal states: %v", states)
		}
	}
}

// --- present values override defaults ---

func TestPresentValue_PollInterval(t *testing.T) {
	c := cfg(map[string]any{"polling": nested("interval_ms", 5000)})
	if c.PollIntervalMs() != 5000 {
		t.Fatalf("expected 5000, got %d", c.PollIntervalMs())
	}
}

func TestPresentValue_MaxConcurrentAgents(t *testing.T) {
	c := cfg(map[string]any{"agent": nested("max_concurrent_agents", 5)})
	if c.MaxConcurrentAgents() != 5 {
		t.Fatalf("expected 5, got %d", c.MaxConcurrentAgents())
	}
}

func TestPresentValue_StringInt(t *testing.T) {
	// SPEC.md allows string integers.
	c := cfg(map[string]any{"polling": nested("interval_ms", "15000")})
	if c.PollIntervalMs() != 15_000 {
		t.Fatalf("expected 15000, got %d", c.PollIntervalMs())
	}
}

// --- $VAR resolution ---

func TestVarResolution_APIKey(t *testing.T) {
	t.Setenv("MY_LINEAR_KEY", "tok-abc")
	c := cfg(map[string]any{"tracker": nested("api_key", "$MY_LINEAR_KEY")})
	if c.TrackerAPIKey() != "tok-abc" {
		t.Fatalf("expected resolved key, got %q", c.TrackerAPIKey())
	}
}

func TestVarResolution_EmptyEnv_TreatedAsMissing(t *testing.T) {
	t.Setenv("EMPTY_KEY", "")
	c := cfg(map[string]any{"tracker": nested("api_key", "$EMPTY_KEY")})
	if c.TrackerAPIKey() != "" {
		t.Fatalf("expected empty, got %q", c.TrackerAPIKey())
	}
}

func TestVarResolution_LiteralKey(t *testing.T) {
	c := cfg(map[string]any{"tracker": nested("api_key", "literal-token")})
	if c.TrackerAPIKey() != "literal-token" {
		t.Fatalf("expected literal-token, got %q", c.TrackerAPIKey())
	}
}

func TestStoragePostgresDSN_ResolvesEnvDefault(t *testing.T) {
	t.Setenv("SYMPHONY_POSTGRES_DSN", "postgres://example")
	c := cfg(nil)
	if c.StoragePostgresDSN() != "postgres://example" {
		t.Fatalf("expected env postgres dsn, got %q", c.StoragePostgresDSN())
	}
}

func TestTrackerStorage_DefaultsToFileForLocal(t *testing.T) {
	c := cfg(map[string]any{"tracker": nested("kind", "local")})
	if c.TrackerStorage() != "file" {
		t.Fatalf("expected file storage, got %q", c.TrackerStorage())
	}
}

func TestInsightsSCMSources_NormalizesAndExpands(t *testing.T) {
	home, _ := os.UserHomeDir()
	c := cfg(map[string]any{
		"insights": nested("scm_sources", []any{
			map[string]any{
				"kind":        "GitHub",
				"repo_path":   "~/work/symphony",
				"main_branch": "",
			},
			map[string]any{
				"kind":      "gitlab",
				"name":      "internal",
				"repo_path": "/srv/git/internal",
			},
		}),
	})

	sources := c.InsightsSCMSources()
	if len(sources) != 2 {
		t.Fatalf("sources len = %d want 2", len(sources))
	}
	if sources[0].Kind != "github" {
		t.Fatalf("first kind = %q want github", sources[0].Kind)
	}
	if sources[0].RepoPath != filepath.Join(home, "work", "symphony") {
		t.Fatalf("first repo path = %q", sources[0].RepoPath)
	}
	if sources[0].MainBranch != "main" {
		t.Fatalf("first main branch = %q want main", sources[0].MainBranch)
	}
	if sources[1].Name != "internal" {
		t.Fatalf("second name = %q want internal", sources[1].Name)
	}
}

// --- ~ path expansion ---

func TestTildeExpansion_WorkspaceRoot(t *testing.T) {
	c := cfg(map[string]any{"workspace": nested("root", "~/myworkspaces")})
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "myworkspaces")
	if c.WorkspaceRoot() != expected {
		t.Fatalf("expected %q, got %q", expected, c.WorkspaceRoot())
	}
}

// --- per-state concurrency map ---

func TestMaxConcurrentByState_Normalized(t *testing.T) {
	c := cfg(map[string]any{
		"agent": nested("max_concurrent_agents_by_state", map[string]any{
			"In Progress": 3,
			"Todo":        1,
			"Bad":         -1, // invalid, ignored
			"Also Bad":    "nope",
		}),
	})
	m := c.MaxConcurrentAgentsByState()
	if m["in progress"] != 3 {
		t.Fatalf("expected 3, got %d", m["in progress"])
	}
	if m["todo"] != 1 {
		t.Fatalf("expected 1, got %d", m["todo"])
	}
	if _, ok := m["bad"]; ok {
		t.Fatal("negative value should be ignored")
	}
}

// --- string list parsing ---

func TestActiveStates_CommaSeparatedString(t *testing.T) {
	c := cfg(map[string]any{"tracker": nested("active_states", "Todo, In Progress, Review")})
	states := c.TrackerActiveStates()
	if len(states) != 3 || states[2] != "Review" {
		t.Fatalf("unexpected states: %v", states)
	}
}

func TestActiveStates_List(t *testing.T) {
	c := cfg(map[string]any{"tracker": nested("active_states", []any{"A", "B"})})
	states := c.TrackerActiveStates()
	if len(states) != 2 || states[0] != "A" {
		t.Fatalf("unexpected states: %v", states)
	}
}

// --- HookTimeoutMs non-positive fallback ---

func TestHookTimeoutMs_NonPositive_FallsBack(t *testing.T) {
	c := cfg(map[string]any{"hooks": nested("timeout_ms", 0)})
	if c.HookTimeoutMs() != 60_000 {
		t.Fatalf("expected 60000 fallback, got %d", c.HookTimeoutMs())
	}
}

// --- ValidateDispatch ---

func TestValidate_Valid(t *testing.T) {
	c := cfg(map[string]any{
		"tracker": nested("kind", "linear", "api_key", "tok", "project_slug", "my-proj"),
	})
	if err := config.ValidateDispatch(c); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidate_MissingKind(t *testing.T) {
	c := cfg(nil)
	assertValidationKind(t, config.ValidateDispatch(c), config.ErrUnsupportedTrackerKind)
}

func TestValidate_UnsupportedKind(t *testing.T) {
	c := cfg(map[string]any{"tracker": nested("kind", "jira")})
	assertValidationKind(t, config.ValidateDispatch(c), config.ErrUnsupportedTrackerKind)
}

func TestValidate_LocalTrackerDoesNotRequireLinearCredentials(t *testing.T) {
	c := cfg(map[string]any{"tracker": nested("kind", "local")})
	if err := config.ValidateDispatch(c); err != nil {
		t.Fatalf("expected valid local config, got %v", err)
	}
}

func TestValidate_LocalTrackerWithUnsupportedStorage(t *testing.T) {
	c := cfg(map[string]any{"tracker": nested("kind", "local", "storage", "redis")})
	assertValidationKind(t, config.ValidateDispatch(c), config.ErrUnsupportedTrackerStorage)
}

func TestValidate_LocalPostgresStorageRequiresDSN(t *testing.T) {
	t.Setenv("SYMPHONY_POSTGRES_DSN", "")
	c := cfg(map[string]any{"tracker": nested("kind", "local", "storage", "postgres")})
	assertValidationKind(t, config.ValidateDispatch(c), config.ErrMissingPostgresDSN)
}

func TestValidate_LocalPostgresStorageWithDSN(t *testing.T) {
	t.Setenv("SYMPHONY_POSTGRES_DSN", "postgres://example")
	c := cfg(map[string]any{"tracker": nested("kind", "local", "storage", "postgres")})
	if err := config.ValidateDispatch(c); err != nil {
		t.Fatalf("expected valid postgres local config, got %v", err)
	}
}

func TestValidate_MissingAPIKey(t *testing.T) {
	c := cfg(map[string]any{"tracker": nested("kind", "linear", "project_slug", "p")})
	assertValidationKind(t, config.ValidateDispatch(c), config.ErrMissingTrackerAPIKey)
}

func TestValidate_EmptyEnvAPIKey(t *testing.T) {
	if err := os.Unsetenv("UNSET_KEY"); err != nil {
		t.Fatalf("unset env: %v", err)
	}
	c := cfg(map[string]any{"tracker": nested("kind", "linear", "api_key", "$UNSET_KEY", "project_slug", "p")})
	assertValidationKind(t, config.ValidateDispatch(c), config.ErrMissingTrackerAPIKey)
}

func TestValidate_MissingProjectSlug(t *testing.T) {
	c := cfg(map[string]any{"tracker": nested("kind", "linear", "api_key", "tok")})
	assertValidationKind(t, config.ValidateDispatch(c), config.ErrMissingTrackerProjectSlug)
}

func assertValidationKind(t *testing.T, err error, kind string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error %q, got nil", kind)
	}
	ve, ok := err.(*config.ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if ve.Kind != kind {
		t.Fatalf("expected kind %q, got %q", kind, ve.Kind)
	}
}
