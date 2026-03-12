package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPostgresRuntimeStateStoreRoundTrip(t *testing.T) {
	db := openRuntimeStateORM(t)
	store := NewPostgresRuntimeStateStore(db, "sym")
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	now := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)
	lastEventAt := now.Add(-time.Minute)
	if err := store.UpsertRetry(context.Background(), PersistedRetryEntry{
		IssueID:       "issue-1",
		Identifier:    "SYM-1",
		Attempt:       2,
		DueAt:         now.Add(30 * time.Second),
		Error:         "retry me",
		Continuation:  true,
		RestartCount:  2,
		RetryAttempt:  2,
		WorkspacePath: "/tmp/SYM-1",
	}); err != nil {
		t.Fatalf("UpsertRetry: %v", err)
	}
	if err := store.UpsertRunning(context.Background(), PersistedRunningEntry{
		IssueID:       "issue-2",
		Identifier:    "SYM-2",
		State:         "In Progress",
		WorkspacePath: "/tmp/SYM-2",
		StartedAt:     now,
		LastEventAt:   &lastEventAt,
		LastEvent:     "thread/message",
		LastMessage:   "working",
		SessionID:     "thread-1-turn-1",
		ThreadID:      "thread-1",
		TurnID:        "turn-1",
		TurnCount:     1,
		RetryAttempt:  3,
		SessionLog:    "/tmp/session.log",
		InputTokens:   10,
		OutputTokens:  5,
		TotalTokens:   15,
		RecentEvents:  []IssueEvent{{At: now, Event: "thread/message", Message: "working"}},
		TrackedMetadata: map[string]any{
			"thread_id": "thread-1",
		},
	}); err != nil {
		t.Fatalf("UpsertRunning: %v", err)
	}

	state, err := store.LoadState(context.Background())
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.Retrying) != 1 || state.Retrying[0].Identifier != "SYM-1" {
		t.Fatalf("retrying = %#v", state.Retrying)
	}
	if len(state.Running) != 1 || state.Running[0].SessionID != "thread-1-turn-1" {
		t.Fatalf("running = %#v", state.Running)
	}

	if err := store.DeleteRetry(context.Background(), "issue-1"); err != nil {
		t.Fatalf("DeleteRetry: %v", err)
	}
	if err := store.DeleteRunning(context.Background(), "issue-2"); err != nil {
		t.Fatalf("DeleteRunning: %v", err)
	}
	state, err = store.LoadState(context.Background())
	if err != nil {
		t.Fatalf("LoadState after delete: %v", err)
	}
	if len(state.Retrying) != 0 || len(state.Running) != 0 {
		t.Fatalf("state after delete = %#v", state)
	}
}

func TestRestoreRuntimeStatePopulatesSnapshotAndIssueDetail(t *testing.T) {
	now := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)
	store := &fakeRuntimeStateStore{
		loadState: PersistedRuntimeState{
			Retrying: []PersistedRetryEntry{{
				IssueID:       "retry-1",
				Identifier:    "SYM-1",
				Attempt:       2,
				DueAt:         now.Add(10 * time.Second),
				Error:         "waiting",
				WorkspacePath: "/tmp/SYM-1",
			}},
			Running: []PersistedRunningEntry{{
				IssueID:       "run-1",
				Identifier:    "SYM-2",
				State:         "In Progress",
				WorkspacePath: "/tmp/SYM-2",
				StartedAt:     now.Add(-time.Minute),
				SessionID:     "thread-2-turn-1",
				ThreadID:      "thread-2",
				TurnID:        "turn-1",
				TurnCount:     1,
				RetryAttempt:  1,
				SessionLog:    "/tmp/run.log",
				TotalTokens:   11,
				RecentEvents:  []IssueEvent{{At: now, Event: "thread/message", Message: "working"}},
			}},
		},
	}
	orc := testOrchestrator(t, testDeps{
		cfg: config.New(map[string]any{
			"workspace": map[string]any{"root": t.TempDir()},
		}),
		now:          func() time.Time { return now },
		runtimeState: store,
	})

	if err := orc.restoreRuntimeState(context.Background()); err != nil {
		t.Fatalf("restoreRuntimeState: %v", err)
	}

	snapshot := orc.Snapshot()
	if snapshot.Counts.Running != 1 || snapshot.Counts.Retrying != 1 {
		t.Fatalf("snapshot counts = %#v", snapshot.Counts)
	}
	if snapshot.RuntimeState == nil || snapshot.RuntimeState.Status != "ok" {
		t.Fatalf("runtime state = %#v", snapshot.RuntimeState)
	}
	detail, ok := orc.Issue("SYM-2")
	if !ok {
		t.Fatal("expected restored running issue detail")
	}
	if detail.Running == nil || detail.Running.SessionID != "thread-2-turn-1" {
		t.Fatalf("detail running = %#v", detail.Running)
	}
	if detail.RuntimeState == nil || detail.RuntimeState.Status != "ok" {
		t.Fatalf("detail runtime state = %#v", detail.RuntimeState)
	}
}

func TestRuntimeStatePersistenceDegradesGracefully(t *testing.T) {
	now := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)
	store := &fakeRuntimeStateStore{
		upsertRetryErr: errors.New("db down"),
	}
	orc := testOrchestrator(t, testDeps{
		cfg: config.New(map[string]any{
			"workspace": map[string]any{"root": t.TempDir()},
		}),
		now:          func() time.Time { return now },
		runtimeState: store,
	})

	orc.mu.Lock()
	orc.scheduleRetry(tracker.Issue{ID: "issue-1", Identifier: "SYM-1"}, 1, false, "failed")
	orc.mu.Unlock()

	retry, ok := orc.state.RetryAttempts["issue-1"]
	if !ok || retry.Attempt != 1 {
		t.Fatalf("retry = %#v", retry)
	}
	snapshot := orc.Snapshot()
	if snapshot.RuntimeState == nil || snapshot.RuntimeState.Status != "degraded" {
		t.Fatalf("runtime state = %#v", snapshot.RuntimeState)
	}
	if snapshot.RuntimeState.LastError == nil || *snapshot.RuntimeState.LastError != "db down" {
		t.Fatalf("runtime state error = %#v", snapshot.RuntimeState)
	}
}

func openRuntimeStateORM(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db
}

type fakeRuntimeStateStore struct {
	loadState      PersistedRuntimeState
	loadErr        error
	upsertRetryErr error
	deleteRetryErr error
	upsertRunErr   error
	deleteRunErr   error
}

func (f *fakeRuntimeStateStore) EnsureSchema(context.Context) error { return nil }

func (f *fakeRuntimeStateStore) LoadState(context.Context) (PersistedRuntimeState, error) {
	return f.loadState, f.loadErr
}

func (f *fakeRuntimeStateStore) UpsertRetry(context.Context, PersistedRetryEntry) error {
	return f.upsertRetryErr
}

func (f *fakeRuntimeStateStore) DeleteRetry(context.Context, string) error {
	return f.deleteRetryErr
}

func (f *fakeRuntimeStateStore) UpsertRunning(context.Context, PersistedRunningEntry) error {
	return f.upsertRunErr
}

func (f *fakeRuntimeStateStore) DeleteRunning(context.Context, string) error {
	return f.deleteRunErr
}
