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

func TestServiceAggregatesProviderChangeRequestTotals(t *testing.T) {
	service := NewService(Options{
		Inspector: fakeInspector{
			metrics: SCMSourceMetrics{
				Kind:                   "github",
				Name:                   "origin",
				OpenChangeRequests:     3,
				ApprovedChangeRequests: 2,
				FailingChangeRequests:  1,
				StaleChangeRequests:    1,
			},
		},
		Sources: []SourceConfig{{
			Kind:       "github",
			Name:       "origin",
			Repository: "org/repo",
		}},
		Now: nowFunc(time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)),
	})

	report, err := service.Delivery(context.Background())
	if err != nil {
		t.Fatalf("Delivery: %v", err)
	}
	if report.SCM.Totals.OpenChangeRequests != 3 {
		t.Fatalf("open change requests = %d want 3", report.SCM.Totals.OpenChangeRequests)
	}
	if report.SCM.Totals.ApprovedChangeRequests != 2 {
		t.Fatalf("approved change requests = %d want 2", report.SCM.Totals.ApprovedChangeRequests)
	}
	if report.Summary.MergeReadiness.Score <= 0 {
		t.Fatalf("merge readiness score = %d", report.Summary.MergeReadiness.Score)
	}
}

func TestServiceDegradesWhenProviderSourceFails(t *testing.T) {
	service := NewService(Options{
		Inspector: fakeInspector{err: context.DeadlineExceeded},
		Sources: []SourceConfig{{
			Kind:       "gitverse",
			Name:       "gitverse",
			Repository: "team/repo",
		}},
		Now: nowFunc(time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)),
	})

	report, err := service.Delivery(context.Background())
	if err != nil {
		t.Fatalf("Delivery: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected warnings")
	}
	if len(report.SCM.Sources) != 1 || len(report.SCM.Sources[0].Warnings) == 0 {
		t.Fatalf("source warnings = %#v", report.SCM.Sources)
	}
}

func TestServicePersistsDeliveryTrendSnapshots(t *testing.T) {
	store := &fakeTrendStore{}
	service := NewService(Options{
		Tasks: fakeTaskProvider{tasks: []tracker.Issue{
			{ID: "task-1", Identifier: "SYM-1", Title: "Active", State: "In Progress"},
		}},
		Inspector: fakeInspector{},
		Trends:    store,
		Now:       nowFunc(time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)),
	})

	report, err := service.Delivery(context.Background())
	if err != nil {
		t.Fatalf("Delivery: %v", err)
	}
	if len(store.appended) != 1 {
		t.Fatalf("appended snapshots = %d want 1", len(store.appended))
	}
	if store.appended[0].DeliveryHealth != report.Summary.DeliveryHealth.Score {
		t.Fatalf("stored health = %d want %d", store.appended[0].DeliveryHealth, report.Summary.DeliveryHealth.Score)
	}
}

func TestServiceReturnsDeliveryTrends(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	store := &fakeTrendStore{
		listed: []DeliveryTrendPoint{{
			CapturedAt:     now.Add(-24 * time.Hour),
			DeliveryHealth: 66,
		}, {
			CapturedAt:     now.Add(-12 * time.Hour),
			DeliveryHealth: 77,
		}},
	}
	service := NewService(Options{
		Trends: store,
		Now:    nowFunc(now),
	})

	report, err := service.Trends(context.Background(), DeliveryTrendQuery{Window: "7d", Limit: 8})
	if err != nil {
		t.Fatalf("Trends: %v", err)
	}
	if !report.Available {
		t.Fatal("expected available trends")
	}
	if len(report.Points) != 2 {
		t.Fatalf("trend points = %d want 2", len(report.Points))
	}
	if report.Points[1].DeliveryHealth != 77 {
		t.Fatalf("latest trend health = %d want 77", report.Points[1].DeliveryHealth)
	}
	if report.Rollups.HealthDelta != 11 {
		t.Fatalf("health delta = %d want 11", report.Rollups.HealthDelta)
	}
}

func TestServiceDegradesDeliveryTrendsWhenStoreMissing(t *testing.T) {
	service := NewService(Options{
		Now: nowFunc(time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)),
	})

	report, err := service.Trends(context.Background(), DeliveryTrendQuery{})
	if err != nil {
		t.Fatalf("Trends: %v", err)
	}
	if report.Available {
		t.Fatal("expected unavailable trends")
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected trend warning")
	}
}

func TestServiceRejectsInvalidTrendWindows(t *testing.T) {
	service := NewService(Options{
		Now: nowFunc(time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)),
	})

	_, err := service.Trends(context.Background(), DeliveryTrendQuery{Window: "365d"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceBuildsTrendAlertsAndRollups(t *testing.T) {
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	service := NewService(Options{
		Trends: &fakeTrendStore{
			listed: []DeliveryTrendPoint{
				{
					CapturedAt:          now.Add(-48 * time.Hour),
					DeliveryHealth:      78,
					FlowEfficiency:      72,
					MergeReadiness:      74,
					Predictability:      73,
					BlockedTasks:        1,
					FailingChangeChecks: 0,
					WarningCount:        1,
				},
				{
					CapturedAt:          now.Add(-24 * time.Hour),
					DeliveryHealth:      63,
					FlowEfficiency:      67,
					MergeReadiness:      68,
					Predictability:      66,
					BlockedTasks:        3,
					FailingChangeChecks: 1,
					WarningCount:        2,
				},
				{
					CapturedAt:          now.Add(-12 * time.Hour),
					DeliveryHealth:      57,
					FlowEfficiency:      61,
					MergeReadiness:      62,
					Predictability:      60,
					BlockedTasks:        4,
					FailingChangeChecks: 2,
					WarningCount:        2,
				},
			},
		},
		Now: nowFunc(now),
	})

	report, err := service.Trends(context.Background(), DeliveryTrendQuery{Window: "7d", Limit: 12})
	if err != nil {
		t.Fatalf("Trends: %v", err)
	}
	if report.Rollups.HealthAverage != 66 {
		t.Fatalf("health average = %d want 66", report.Rollups.HealthAverage)
	}
	if report.Rollups.HealthDelta != -21 {
		t.Fatalf("health delta = %d want -21", report.Rollups.HealthDelta)
	}
	if report.Rollups.WarningPressure <= 1.0 {
		t.Fatalf("warning pressure = %v want > 1", report.Rollups.WarningPressure)
	}
	if len(report.Alerts) < 3 {
		t.Fatalf("alerts len = %d want at least 3", len(report.Alerts))
	}
}

func TestServiceTrendRollupsDegradeForSinglePoint(t *testing.T) {
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	service := NewService(Options{
		Trends: &fakeTrendStore{
			listed: []DeliveryTrendPoint{{
				CapturedAt:     now.Add(-12 * time.Hour),
				DeliveryHealth: 70,
			}},
		},
		Now: nowFunc(now),
	})

	report, err := service.Trends(context.Background(), DeliveryTrendQuery{Window: "24h", Limit: 12})
	if err != nil {
		t.Fatalf("Trends: %v", err)
	}
	if !report.Rollups.InsufficientSamples {
		t.Fatal("expected insufficient samples")
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

type fakeTrendStore struct {
	appended []DeliveryTrendPoint
	listed   []DeliveryTrendPoint
	err      error
}

func (f fakeInspector) Inspect(context.Context, SourceConfig, time.Duration, time.Time) (SCMSourceMetrics, error) {
	return f.metrics, f.err
}

func (f *fakeTrendStore) AppendDeliverySnapshot(_ context.Context, point DeliveryTrendPoint) error {
	f.appended = append(f.appended, point)
	return f.err
}

func (f *fakeTrendStore) ListDeliverySnapshots(
	_ context.Context,
	_ DeliveryTrendQuery,
	_ time.Time,
) ([]DeliveryTrendPoint, error) {
	return f.listed, f.err
}

func nowFunc(now time.Time) func() time.Time {
	return func() time.Time { return now }
}

func strPtr(value string) *string {
	return &value
}
