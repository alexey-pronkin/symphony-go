package insights

import (
	"context"
	"testing"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/orchestrator"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

func TestServiceBuildsDeliveryReport(t *testing.T) {
	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	doneUpdated := now.Add(-24 * time.Hour)
	createdAt := now.Add(-72 * time.Hour)
	blockerState := "Todo"
	service := NewService(Options{
		Tasks: fakeTaskProvider{tasks: []tracker.Issue{
			{
				ID:         "task-1",
				Identifier: "SYM-1",
				Title:      "Active",
				State:      "In Progress",
				CreatedAt:  &createdAt,
				BlockedBy: []tracker.BlockerRef{{
					Identifier: strPtr("SYM-0"),
					State:      &blockerState,
				}},
			},
			{
				ID:         "task-2",
				Identifier: "SYM-2",
				Title:      "Done",
				State:      "Done",
				UpdatedAt:  &doneUpdated,
			},
		}},
		Runtime: fakeRuntimeProvider{
			snapshot: orchestrator.Snapshot{
				Counts: orchestrator.SnapshotCounts{Running: 2, Retrying: 1},
				CodexTotals: orchestrator.SnapshotTotals{
					TotalTokens: 42,
				},
			},
		},
		Inspector: fakeInspector{
			metrics: SCMSourceMetrics{
				Kind:             "github",
				Name:             "origin",
				RepoPath:         "/tmp/repo",
				MainBranch:       "main",
				Branches:         3,
				UnmergedBranches: 2,
				StaleBranches:    1,
				DriftCommits:     4,
				AheadCommits:     3,
				MaxAgeHours:      96,
				MergeReadiness:   61,
			},
		},
		Sources: []SourceConfig{{
			Kind:       "github",
			Name:       "origin",
			RepoPath:   "/tmp/repo",
			MainBranch: "main",
		}},
		Now: nowFunc(now),
	})

	report, err := service.Delivery(context.Background())
	if err != nil {
		t.Fatalf("Delivery: %v", err)
	}
	if report.Tracker.BlockedTasks != 1 {
		t.Fatalf("blocked tasks = %d want 1", report.Tracker.BlockedTasks)
	}
	if report.Tracker.DoneLastWindow != 1 {
		t.Fatalf("done last window = %d want 1", report.Tracker.DoneLastWindow)
	}
	if report.SCM.ActiveSources != 1 {
		t.Fatalf("active sources = %d want 1", report.SCM.ActiveSources)
	}
	if report.Summary.DeliveryHealth.Score <= 0 {
		t.Fatalf("delivery health score = %d", report.Summary.DeliveryHealth.Score)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("warnings = %#v want none", report.Warnings)
	}
}

func TestServiceWarnsWhenSCMSourcesMissing(t *testing.T) {
	service := NewService(Options{
		Now: nowFunc(time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)),
	})

	report, err := service.Delivery(context.Background())
	if err != nil {
		t.Fatalf("Delivery: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected warnings")
	}
	if report.SCM.ActiveSources != 0 {
		t.Fatalf("active sources = %d want 0", report.SCM.ActiveSources)
	}
}

type fakeTaskProvider struct {
	tasks []tracker.Issue
}

func (f fakeTaskProvider) ListTasks(context.Context) ([]tracker.Issue, error) {
	return f.tasks, nil
}

type fakeRuntimeProvider struct {
	snapshot orchestrator.Snapshot
}

func (f fakeRuntimeProvider) Snapshot() orchestrator.Snapshot {
	return f.snapshot
}

type fakeInspector struct {
	metrics SCMSourceMetrics
	err     error
}

func (f fakeInspector) Inspect(context.Context, SourceConfig, time.Duration, time.Time) (SCMSourceMetrics, error) {
	return f.metrics, f.err
}

func nowFunc(now time.Time) func() time.Time {
	return func() time.Time { return now }
}

func strPtr(value string) *string {
	return &value
}
