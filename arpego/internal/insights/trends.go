package insights

import (
	"context"
	"io"
	"slices"
	"strings"
	"time"

	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

type ClickHouseTrendStore struct {
	db      *gorm.DB
	closer  io.Closer
	project string
}

type deliverySnapshotRecord struct {
	ProjectSlug         string    `gorm:"column:project_slug"`
	CapturedAt          time.Time `gorm:"column:captured_at"`
	DeliveryHealth      int       `gorm:"column:delivery_health"`
	FlowEfficiency      int       `gorm:"column:flow_efficiency"`
	MergeReadiness      int       `gorm:"column:merge_readiness"`
	Predictability      int       `gorm:"column:predictability"`
	ActiveTasks         int       `gorm:"column:active_tasks"`
	BlockedTasks        int       `gorm:"column:blocked_tasks"`
	DoneLastWindow      int       `gorm:"column:done_last_window"`
	WIPCount            int       `gorm:"column:wip_count"`
	OpenChangeRequests  int       `gorm:"column:open_change_requests"`
	FailingChangeChecks int       `gorm:"column:failing_change_checks"`
	WarningCount        int       `gorm:"column:warning_count"`
}

func (deliverySnapshotRecord) TableName() string {
	return "delivery_snapshots"
}

func OpenClickHouseTrendStore(
	ctx context.Context,
	dsn string,
	project string,
) (*ClickHouseTrendStore, error) {
	db, err := gorm.Open(clickhouse.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	store := NewClickHouseTrendStore(db, project)
	store.closer = sqlDB
	if err := store.EnsureSchema(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return store, nil
}

func NewClickHouseTrendStore(db *gorm.DB, project string) *ClickHouseTrendStore {
	return &ClickHouseTrendStore{
		db:      db,
		project: strings.TrimSpace(project),
	}
}

func (s *ClickHouseTrendStore) Close() error {
	if s == nil || s.closer == nil {
		return nil
	}
	return s.closer.Close()
}

func (s *ClickHouseTrendStore) EnsureSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	if s.db.Name() == "sqlite" {
		return s.db.WithContext(ctx).AutoMigrate(&deliverySnapshotRecord{})
	}
	return s.db.WithContext(ctx).Exec(`
CREATE TABLE IF NOT EXISTS delivery_snapshots (
	project_slug String,
	captured_at DateTime64(3, 'UTC'),
	delivery_health Int32,
	flow_efficiency Int32,
	merge_readiness Int32,
	predictability Int32,
	active_tasks Int32,
	blocked_tasks Int32,
	done_last_window Int32,
	wip_count Int32,
	open_change_requests Int32,
	failing_change_checks Int32,
	warning_count Int32
) ENGINE = MergeTree()
ORDER BY (project_slug, captured_at)
`).Error
}

func (s *ClickHouseTrendStore) AppendDeliverySnapshot(ctx context.Context, point DeliveryTrendPoint) error {
	if s == nil || s.db == nil {
		return nil
	}
	record := deliverySnapshotRecord{
		ProjectSlug:         s.project,
		CapturedAt:          point.CapturedAt.UTC(),
		DeliveryHealth:      point.DeliveryHealth,
		FlowEfficiency:      point.FlowEfficiency,
		MergeReadiness:      point.MergeReadiness,
		Predictability:      point.Predictability,
		ActiveTasks:         point.ActiveTasks,
		BlockedTasks:        point.BlockedTasks,
		DoneLastWindow:      point.DoneLastWindow,
		WIPCount:            point.WIPCount,
		OpenChangeRequests:  point.OpenChangeRequests,
		FailingChangeChecks: point.FailingChangeChecks,
		WarningCount:        point.WarningCount,
	}
	if record.CapturedAt.IsZero() {
		record.CapturedAt = time.Now().UTC()
	}
	return s.db.WithContext(ctx).Create(&record).Error
}

func (s *ClickHouseTrendStore) ListDeliverySnapshots(
	ctx context.Context,
	query DeliveryTrendQuery,
	now time.Time,
) ([]DeliveryTrendPoint, error) {
	if s == nil || s.db == nil {
		return []DeliveryTrendPoint{}, nil
	}
	since := now.UTC().Add(-trendWindowDuration(query.Window))
	var records []deliverySnapshotRecord
	if err := s.db.WithContext(ctx).
		Model(&deliverySnapshotRecord{}).
		Where("project_slug = ?", s.project).
		Where("captured_at >= ?", since).
		Order("captured_at DESC").
		Limit(query.Limit).
		Find(&records).Error; err != nil {
		return nil, err
	}
	points := make([]DeliveryTrendPoint, 0, len(records))
	for _, record := range records {
		points = append(points, DeliveryTrendPoint{
			CapturedAt:          record.CapturedAt.UTC(),
			DeliveryHealth:      record.DeliveryHealth,
			FlowEfficiency:      record.FlowEfficiency,
			MergeReadiness:      record.MergeReadiness,
			Predictability:      record.Predictability,
			ActiveTasks:         record.ActiveTasks,
			BlockedTasks:        record.BlockedTasks,
			DoneLastWindow:      record.DoneLastWindow,
			WIPCount:            record.WIPCount,
			OpenChangeRequests:  record.OpenChangeRequests,
			FailingChangeChecks: record.FailingChangeChecks,
			WarningCount:        record.WarningCount,
		})
	}
	slices.Reverse(points)
	return points, nil
}
