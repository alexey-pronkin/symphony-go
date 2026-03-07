package tracker_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

func TestLocalPlatformFetchesAndPersistsTasks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASKS.yaml")
	platform := tracker.NewLocalPlatform(path, "SYM")

	created, err := platform.CreateTask(tracker.CreateTaskInput{
		Title:  "Build local tracker",
		State:  "Todo",
		Labels: []string{"Platform", "Urgent"},
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if created.Identifier != "SYM-1" {
		t.Fatalf("identifier = %q want SYM-1", created.Identifier)
	}
	if len(created.Labels) != 2 || created.Labels[0] != "platform" {
		t.Fatalf("labels = %#v", created.Labels)
	}

	candidates, err := platform.FetchCandidates(context.Background(), []string{"Todo"})
	if err != nil {
		t.Fatalf("FetchCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Identifier != "SYM-1" {
		t.Fatalf("candidates = %#v", candidates)
	}

	nextState := "In Progress"
	updated, err := platform.UpdateTask("SYM-1", tracker.UpdateTaskInput{State: &nextState})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.State != "In Progress" {
		t.Fatalf("state = %q", updated.State)
	}

	refreshed, err := platform.FetchStatesByIDs(context.Background(), []string{created.ID})
	if err != nil {
		t.Fatalf("FetchStatesByIDs: %v", err)
	}
	if len(refreshed) != 1 || refreshed[0].State != "In Progress" {
		t.Fatalf("refreshed = %#v", refreshed)
	}
}

func TestLocalPlatformDefaultsToEmptyStoreWhenFileIsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASKS.yaml")
	platform := tracker.NewLocalPlatform(path, "SYM")

	issues, err := platform.FetchCandidates(context.Background(), []string{"Todo"})
	if err != nil {
		t.Fatalf("FetchCandidates: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues len = %d want 0", len(issues))
	}
}
