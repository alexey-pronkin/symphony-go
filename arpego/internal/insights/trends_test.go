package insights

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestClickHouseTrendStoreAppendsAndListsSnapshots(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	store := NewClickHouseTrendStore(db, "symphony")
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	points := []DeliveryTrendPoint{
		{
			CapturedAt:          now.Add(-48 * time.Hour),
			DeliveryHealth:      61,
			FlowEfficiency:      58,
			MergeReadiness:      62,
			Predictability:      55,
			ActiveTasks:         8,
			BlockedTasks:        2,
			DoneLastWindow:      3,
			WIPCount:            5,
			OpenChangeRequests:  4,
			FailingChangeChecks: 1,
			WarningCount:        0,
		},
		{
			CapturedAt:          now.Add(-12 * time.Hour),
			DeliveryHealth:      74,
			FlowEfficiency:      71,
			MergeReadiness:      69,
			Predictability:      67,
			ActiveTasks:         6,
			BlockedTasks:        1,
			DoneLastWindow:      5,
			WIPCount:            4,
			OpenChangeRequests:  2,
			FailingChangeChecks: 0,
			WarningCount:        1,
		},
	}
	for _, point := range points {
		if err := store.AppendDeliverySnapshot(context.Background(), point); err != nil {
			t.Fatalf("AppendDeliverySnapshot: %v", err)
		}
	}

	got, err := store.ListDeliverySnapshots(context.Background(), DeliveryTrendQuery{
		Window: "7d",
		Limit:  10,
	}, now)
	if err != nil {
		t.Fatalf("ListDeliverySnapshots: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("points len = %d want 2", len(got))
	}
	if !got[0].CapturedAt.Equal(points[0].CapturedAt) {
		t.Fatalf("first captured_at = %v want %v", got[0].CapturedAt, points[0].CapturedAt)
	}
	if got[1].DeliveryHealth != 74 {
		t.Fatalf("latest delivery health = %d want 74", got[1].DeliveryHealth)
	}
}
