package orchestrator

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/agent"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

func TestSortIssuesForDispatch(t *testing.T) {
	createdOld := mustTime(t, "2026-03-01T10:00:00Z")
	createdNew := mustTime(t, "2026-03-02T10:00:00Z")
	priorityOne := 1
	priorityThree := 3

	issues := []tracker.Issue{
		{ID: "3", Identifier: "MT-30", Title: "Nil priority", State: "Todo", CreatedAt: &createdOld},
		{ID: "2", Identifier: "MT-20", Title: "Later", State: "Todo", Priority: &priorityOne, CreatedAt: &createdNew},
		{ID: "4", Identifier: "MT-05", Title: "Tie breaker", State: "Todo", Priority: &priorityOne, CreatedAt: &createdNew},
		{
			ID:         "1",
			Identifier: "MT-10",
			Title:      "Lower priority",
			State:      "Todo",
			Priority:   &priorityThree,
			CreatedAt:  &createdOld,
		},
	}

	got := sortIssuesForDispatch(issues)
	want := []string{"MT-05", "MT-20", "MT-10", "MT-30"}
	for i, identifier := range want {
		if got[i].Identifier != identifier {
			t.Fatalf("index %d identifier = %q want %q", i, got[i].Identifier, identifier)
		}
	}
}

func TestDispatchEligibleRespectsClaimsBlockersAndPerStateConcurrency(t *testing.T) {
	priority := 1
	cfg := config.New(map[string]any{
		"tracker": map[string]any{
			"active_states":   []any{"Todo", "In Progress"},
			"terminal_states": []any{"Done"},
		},
		"agent": map[string]any{
			"max_concurrent_agents":          3,
			"max_concurrent_agents_by_state": map[string]any{"todo": 1},
		},
	})
	state := State{
		Running: map[string]*RunningEntry{
			"running-todo": {
				Issue: tracker.Issue{ID: "running-todo", Identifier: "MT-1", Title: "Existing", State: "Todo"},
			},
		},
		Claimed: map[string]struct{}{
			"claimed": {},
		},
	}

	claimed := tracker.Issue{ID: "claimed", Identifier: "MT-2", Title: "Claimed", State: "Todo", Priority: &priority}
	if dispatchEligible(claimed, state, cfg) {
		t.Fatal("claimed issue should not be eligible")
	}

	blocked := tracker.Issue{
		ID:         "blocked",
		Identifier: "MT-3",
		Title:      "Blocked",
		State:      "Todo",
		Priority:   &priority,
		BlockedBy: []tracker.BlockerRef{
			{State: stringPtr("In Progress")},
		},
	}
	if dispatchEligible(blocked, state, cfg) {
		t.Fatal("todo issue with non-terminal blocker should not be eligible")
	}

	saturated := tracker.Issue{ID: "saturated", Identifier: "MT-4", Title: "Todo", State: "Todo", Priority: &priority}
	if dispatchEligible(saturated, state, cfg) {
		t.Fatal("per-state saturated issue should not be eligible")
	}

	eligible := tracker.Issue{
		ID:         "eligible",
		Identifier: "MT-5",
		Title:      "In Progress",
		State:      "In Progress",
		Priority:   &priority,
	}
	if !dispatchEligible(eligible, state, cfg) {
		t.Fatal("in-progress issue should be eligible when global slots remain")
	}
}

func TestRetryDelayUsesContinuationAndFailureBackoffCap(t *testing.T) {
	if got := retryDelay(1, true, 5*time.Minute); got != time.Second {
		t.Fatalf("continuation delay = %s want 1s", got)
	}
	if got := retryDelay(1, false, 5*time.Minute); got != 10*time.Second {
		t.Fatalf("attempt1 delay = %s want 10s", got)
	}
	if got := retryDelay(6, false, 90*time.Second); got != 90*time.Second {
		t.Fatalf("capped delay = %s want 90s", got)
	}
}

func TestStartupCleanupRemovesTerminalWorkspaces(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "MT-1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	orc := testOrchestrator(t, testDeps{
		cfg: config.New(map[string]any{
			"workspace": map[string]any{"root": root},
			"tracker":   map[string]any{"terminal_states": []any{"Done"}},
		}),
		tracker: &fakeTracker{
			byStates: func(_ context.Context, states []string) ([]tracker.Issue, error) {
				if len(states) != 1 || states[0] != "Done" {
					t.Fatalf("terminal states = %#v", states)
				}
				return []tracker.Issue{{ID: "1", Identifier: "MT-1", Title: "done", State: "Done"}}, nil
			},
		},
	})

	orc.runStartupCleanup(context.Background())

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("workspace still exists, err = %v", err)
	}
}

func TestReconcileRunningTerminalAndNonActiveStates(t *testing.T) {
	root := t.TempDir()
	terminalPath := filepath.Join(root, "MT-10")
	pausedPath := filepath.Join(root, "MT-11")
	for _, path := range []string{terminalPath, pausedPath} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", path, err)
		}
	}

	var terminalCancelled atomic.Int32
	var pausedCancelled atomic.Int32

	orc := testOrchestrator(t, testDeps{
		cfg: config.New(map[string]any{
			"workspace": map[string]any{"root": root},
			"tracker": map[string]any{
				"active_states":   []any{"Todo", "In Progress"},
				"terminal_states": []any{"Done"},
			},
		}),
		tracker: &fakeTracker{
			byIDs: func(_ context.Context, ids []string) ([]tracker.Issue, error) {
				if len(ids) != 2 {
					t.Fatalf("ids = %#v", ids)
				}
				return []tracker.Issue{
					{ID: "terminal", Identifier: "MT-10", Title: "Done", State: "Done"},
					{ID: "paused", Identifier: "MT-11", Title: "Paused", State: "Backlog"},
				}, nil
			},
		},
	})
	orc.state = State{
		Running: map[string]*RunningEntry{
			"terminal": {
				Issue:         tracker.Issue{ID: "terminal", Identifier: "MT-10", Title: "Done", State: "In Progress"},
				WorkspacePath: terminalPath,
				cancel:        func() { terminalCancelled.Add(1) },
				StartedAt:     time.Now().Add(-time.Minute),
			},
			"paused": {
				Issue:         tracker.Issue{ID: "paused", Identifier: "MT-11", Title: "Paused", State: "In Progress"},
				WorkspacePath: pausedPath,
				cancel:        func() { pausedCancelled.Add(1) },
				StartedAt:     time.Now().Add(-time.Minute),
			},
		},
		Claimed: map[string]struct{}{"terminal": {}, "paused": {}},
	}

	orc.reconcileRunning(context.Background())

	if terminalCancelled.Load() != 1 {
		t.Fatalf("terminal cancel count = %d want 1", terminalCancelled.Load())
	}
	if pausedCancelled.Load() != 1 {
		t.Fatalf("paused cancel count = %d want 1", pausedCancelled.Load())
	}
	if len(orc.state.Running) != 0 {
		t.Fatalf("running entries = %#v want empty", orc.state.Running)
	}
	if _, ok := orc.state.Claimed["terminal"]; ok {
		t.Fatal("terminal claim should be released")
	}
	if _, ok := orc.state.Claimed["paused"]; ok {
		t.Fatal("non-active claim should be released")
	}
	if _, err := os.Stat(terminalPath); !os.IsNotExist(err) {
		t.Fatalf("terminal workspace still exists, err = %v", err)
	}
	if _, err := os.Stat(pausedPath); err != nil {
		t.Fatalf("paused workspace should remain: %v", err)
	}
}

func TestReconcileRunningStalledSchedulesRetry(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "MT-20")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	var cancelled atomic.Int32
	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	orc := testOrchestrator(t, testDeps{
		cfg: config.New(map[string]any{
			"workspace": map[string]any{"root": root},
			"tracker": map[string]any{
				"active_states":   []any{"Todo", "In Progress"},
				"terminal_states": []any{"Done"},
			},
			"codex": map[string]any{"stall_timeout_ms": 1_000},
			"agent": map[string]any{"max_retry_backoff_ms": 300_000},
		}),
		now: func() time.Time { return now },
		tracker: &fakeTracker{
			byIDs: func(_ context.Context, ids []string) ([]tracker.Issue, error) {
				if len(ids) != 0 {
					t.Fatalf("expected no state refresh after stall, got %#v", ids)
				}
				return nil, nil
			},
		},
	})
	orc.state = State{
		Running: map[string]*RunningEntry{
			"stall": {
				Issue:         tracker.Issue{ID: "stall", Identifier: "MT-20", Title: "Stalled", State: "In Progress"},
				WorkspacePath: path,
				cancel:        func() { cancelled.Add(1) },
				StartedAt:     now.Add(-5 * time.Second),
			},
		},
		Claimed: map[string]struct{}{"stall": {}},
	}

	orc.reconcileRunning(context.Background())

	if cancelled.Load() != 1 {
		t.Fatalf("cancel count = %d want 1", cancelled.Load())
	}
	if len(orc.state.Running) != 0 {
		t.Fatalf("running entries = %#v want empty", orc.state.Running)
	}
	retry := orc.state.RetryAttempts["stall"]
	if retry.Attempt != 1 {
		t.Fatalf("retry attempt = %d want 1", retry.Attempt)
	}
	if retry.Identifier != "MT-20" {
		t.Fatalf("retry identifier = %q", retry.Identifier)
	}
	if retry.Error == "" {
		t.Fatal("retry error should be populated")
	}
	if _, ok := orc.state.Claimed["stall"]; !ok {
		t.Fatal("stalled issue claim should remain for retry")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("workspace should remain after stall retry: %v", err)
	}
}

func TestWorkerAccountingTracksSessionIDTokensAndRuntime(t *testing.T) {
	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	orc := testOrchestrator(t, testDeps{
		cfg: config.New(map[string]any{
			"agent": map[string]any{"max_retry_backoff_ms": 300_000},
		}),
		now: func() time.Time { return now },
	})
	orc.state = State{
		Running: map[string]*RunningEntry{
			"issue-1": {
				Issue:        tracker.Issue{ID: "issue-1", Identifier: "MT-1", Title: "Tracked", State: "In Progress"},
				StartedAt:    now.Add(-15 * time.Second),
				RetryAttempt: 0,
			},
		},
		Claimed: map[string]struct{}{"issue-1": {}},
	}

	orc.recordSession(
		tracker.Issue{ID: "issue-1", Identifier: "MT-1"},
		agent.SessionStarted{ThreadID: "thread-1", TurnID: "turn-2"},
	)
	entry := orc.state.Running["issue-1"]
	if entry.SessionID != "thread-1-turn-2" {
		t.Fatalf("session id = %q", entry.SessionID)
	}

	orc.recordEvent("issue-1", agent.Event{
		Method:  "thread/tokenUsage/updated",
		Usage:   &agent.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
		Payload: map[string]any{"rate_limits": map[string]any{"remaining": 42}},
	})
	orc.recordEvent("issue-1", agent.Event{
		Method: "thread/tokenUsage/updated",
		Usage:  &agent.Usage{InputTokens: 14, OutputTokens: 9, TotalTokens: 23},
	})

	entry = orc.state.Running["issue-1"]
	if entry.CurrentUsage.TotalTokens != 23 {
		t.Fatalf("current total tokens = %d want 23", entry.CurrentUsage.TotalTokens)
	}
	if orc.state.CodexRateLimits["remaining"] != 42 {
		t.Fatalf("rate limits = %#v", orc.state.CodexRateLimits)
	}

	now = now.Add(5 * time.Second)
	orc.handleWorkerResult(workerResult{
		IssueID: "issue-1",
		Result:  agent.RunnerResult{Completed: true, Usage: entry.CurrentUsage},
	})

	if _, ok := orc.state.Running["issue-1"]; ok {
		t.Fatal("running entry should be removed after worker result")
	}
	if _, ok := orc.state.Completed["issue-1"]; !ok {
		t.Fatal("completed set should include issue")
	}
	if orc.state.CodexTotals.TotalTokens != 23 {
		t.Fatalf("aggregate total tokens = %d want 23", orc.state.CodexTotals.TotalTokens)
	}
	if orc.state.CodexTotals.SecondsRunning != 20 {
		t.Fatalf("runtime seconds = %d want 20", orc.state.CodexTotals.SecondsRunning)
	}
	if retry := orc.state.RetryAttempts["issue-1"]; retry.Attempt != 1 {
		t.Fatalf("continuation retry attempt = %d want 1", retry.Attempt)
	}
}

type testDeps struct {
	cfg     config.Config
	tracker *fakeTracker
	now     func() time.Time
}

func testOrchestrator(t *testing.T, deps testDeps) *Orchestrator {
	t.Helper()
	var sink bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&sink, nil))
	orc := New(Options{
		Config:  deps.cfg,
		Logger:  logger,
		Tracker: deps.tracker,
		Now:     deps.now,
		AfterFunc: func(time.Duration, func()) timerHandle {
			return noopTimer{}
		},
	})
	if orc.now == nil {
		t.Fatal("orchestrator clock must be set")
	}
	return orc
}

type noopTimer struct{}

func (noopTimer) Stop() bool { return true }

type fakeTracker struct {
	candidates func(context.Context, []string) ([]tracker.Issue, error)
	byStates   func(context.Context, []string) ([]tracker.Issue, error)
	byIDs      func(context.Context, []string) ([]tracker.Issue, error)
}

func (f *fakeTracker) FetchCandidates(ctx context.Context, states []string) ([]tracker.Issue, error) {
	if f.candidates == nil {
		return nil, nil
	}
	return f.candidates(ctx, states)
}

func (f *fakeTracker) FetchByStates(ctx context.Context, states []string) ([]tracker.Issue, error) {
	if f.byStates == nil {
		return nil, nil
	}
	return f.byStates(ctx, states)
}

func (f *fakeTracker) FetchStatesByIDs(ctx context.Context, ids []string) ([]tracker.Issue, error) {
	if f.byIDs == nil {
		return nil, nil
	}
	return f.byIDs(ctx, ids)
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time.Parse(%q): %v", value, err)
	}
	return ts
}

func stringPtr(value string) *string {
	return &value
}
