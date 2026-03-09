package tracker

import (
	"context"
	"testing"
	"time"
)

func TestClickHouseObservabilityAppendsAndListsRuntimeEvents(t *testing.T) {
	db := openTaskORM(t)
	store := NewClickHouseObservability(db, "sym")
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	base := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	events := []RuntimeEvent{
		{
			IssueID:    "issue-1",
			Identifier: "SYM-1",
			Name:       "session.started",
			Message:    "session started",
			ObservedAt: base,
			SessionID:  "thread-1-turn-1",
			Workspace:  "/tmp/SYM-1",
			LogPath:    "/tmp/SYM-1/.symphony/session.jsonl",
		},
		{
			IssueID:     "issue-1",
			Identifier:  "SYM-1",
			Name:        "turn.completed",
			Message:     "turn completed",
			ObservedAt:  base.Add(2 * time.Second),
			SessionID:   "thread-1-turn-1",
			Workspace:   "/tmp/SYM-1",
			LogPath:     "/tmp/SYM-1/.symphony/session.jsonl",
			MetadataRaw: []byte(`{"turn":1}`),
		},
		{
			IssueID:    "issue-2",
			Identifier: "SYM-2",
			Name:       "session.started",
			Message:    "other issue",
			ObservedAt: base.Add(3 * time.Second),
		},
	}

	for _, event := range events {
		if err := store.AppendRuntimeEvent(context.Background(), event); err != nil {
			t.Fatalf("AppendRuntimeEvent(%s): %v", event.Name, err)
		}
	}

	listed, err := store.ListRuntimeEvents(context.Background(), RuntimeEventQuery{
		IssueID:    "issue-1",
		Identifier: "SYM-1",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("ListRuntimeEvents: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("len(listed) = %d want 2", len(listed))
	}
	if listed[0].Name != "session.started" || listed[1].Name != "turn.completed" {
		t.Fatalf("listed names = %#v", listed)
	}
	if listed[1].LogPath != "/tmp/SYM-1/.symphony/session.jsonl" {
		t.Fatalf("log path = %q", listed[1].LogPath)
	}
	if string(listed[1].MetadataRaw) != `{"turn":1}` {
		t.Fatalf("metadata = %q", string(listed[1].MetadataRaw))
	}
}

func TestClickHouseObservabilityLimitAndIdentifierFiltering(t *testing.T) {
	db := openTaskORM(t)
	store := NewClickHouseObservability(db, "sym")
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	base := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	for index := range 4 {
		if err := store.AppendRuntimeEvent(context.Background(), RuntimeEvent{
			IssueID:    "issue-3",
			Identifier: "SYM-3",
			Name:       "tick",
			Message:    "event",
			ObservedAt: base.Add(time.Duration(index) * time.Second),
		}); err != nil {
			t.Fatalf("AppendRuntimeEvent(%d): %v", index, err)
		}
	}

	listed, err := store.ListRuntimeEvents(context.Background(), RuntimeEventQuery{
		Identifier: "SYM-3",
		Limit:      2,
	})
	if err != nil {
		t.Fatalf("ListRuntimeEvents: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("len(listed) = %d want 2", len(listed))
	}
	if !listed[0].ObservedAt.Before(listed[1].ObservedAt) {
		t.Fatalf("events not returned oldest-to-newest within limit: %#v", listed)
	}
}
