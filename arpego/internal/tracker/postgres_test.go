package tracker

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPostgresPlatformCRUDAndTrackerQueries(t *testing.T) {
	db := openTaskORM(t)
	platform := NewPostgresPlatform(db, "SYM")
	if err := platform.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	created, err := platform.CreateTask(context.Background(), CreateTaskInput{
		Title:       "Persisted task",
		Description: strPtr("Stored in postgres-style repo"),
		State:       "Todo",
		Priority:    intPtr(2),
		Labels:      []string{"Platform", "Urgent"},
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if created.Identifier != "SYM-1" {
		t.Fatalf("identifier = %q want SYM-1", created.Identifier)
	}

	listed, err := platform.ListTasks(context.Background())
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(listed) != 1 || listed[0].Identifier != "SYM-1" {
		t.Fatalf("listed = %#v", listed)
	}

	done := "Done"
	updated, err := platform.UpdateTask(context.Background(), "SYM-1", UpdateTaskInput{State: &done})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.State != "Done" {
		t.Fatalf("state = %q want Done", updated.State)
	}

	candidates, err := platform.FetchCandidates(context.Background(), []string{"Done"})
	if err != nil {
		t.Fatalf("FetchCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].State != "Done" {
		t.Fatalf("candidates = %#v", candidates)
	}

	refreshed, err := platform.FetchStatesByIDs(context.Background(), []string{created.ID})
	if err != nil {
		t.Fatalf("FetchStatesByIDs: %v", err)
	}
	if len(refreshed) != 1 || refreshed[0].Identifier != created.Identifier {
		t.Fatalf("refreshed = %#v", refreshed)
	}
}

func TestPostgresPlatformUpdateTaskReturnsNotFound(t *testing.T) {
	db := openTaskORM(t)
	platform := NewPostgresPlatform(db, "SYM")
	if err := platform.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	_, err := platform.UpdateTask(context.Background(), "SYM-404", UpdateTaskInput{})
	if err == nil {
		t.Fatal("expected not found error")
	}
	taskErr, ok := err.(*TaskError)
	if !ok || taskErr.Code != ErrTaskNotFound {
		t.Fatalf("err = %#v", err)
	}
}

func openTaskORM(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db
}

func strPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}
