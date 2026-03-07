package tracker

import (
	"context"
	"io"
	"slices"
	"strings"
	"time"

	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

type ClickHouseObservability struct {
	db      *gorm.DB
	closer  io.Closer
	project string
}

type clickHouseRuntimeEventRecord struct {
	ProjectSlug string    `gorm:"column:project_slug"`
	IssueID     string    `gorm:"column:issue_id"`
	Identifier  string    `gorm:"column:identifier"`
	Name        string    `gorm:"column:event_name"`
	Message     string    `gorm:"column:message"`
	ObservedAt  time.Time `gorm:"column:observed_at"`
	SessionID   string    `gorm:"column:session_id"`
	Workspace   string    `gorm:"column:workspace"`
	LogPath     string    `gorm:"column:log_path"`
	Metadata    string    `gorm:"column:metadata_json"`
}

func (clickHouseRuntimeEventRecord) TableName() string {
	return "runtime_events"
}

func OpenClickHouseObservability(
	ctx context.Context,
	dsn string,
	project string,
) (*ClickHouseObservability, error) {
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
	store := NewClickHouseObservability(db, project)
	store.closer = sqlDB
	if err := store.EnsureSchema(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return store, nil
}

func NewClickHouseObservability(db *gorm.DB, project string) *ClickHouseObservability {
	return &ClickHouseObservability{
		db:      db,
		project: strings.TrimSpace(project),
	}
}

func (c *ClickHouseObservability) Close() error {
	if c == nil || c.closer == nil {
		return nil
	}
	return c.closer.Close()
}

func (c *ClickHouseObservability) EnsureSchema(ctx context.Context) error {
	if c == nil || c.db == nil {
		return nil
	}
	if c.db.Name() == "sqlite" {
		return c.db.WithContext(ctx).AutoMigrate(&clickHouseRuntimeEventRecord{})
	}
	return c.db.WithContext(ctx).Exec(`
CREATE TABLE IF NOT EXISTS runtime_events (
	project_slug String,
	issue_id String,
	identifier String,
	event_name String,
	message String,
	observed_at DateTime64(3, 'UTC'),
	session_id String,
	workspace String,
	log_path String,
	metadata_json String
) ENGINE = MergeTree()
ORDER BY (project_slug, identifier, observed_at, issue_id)
`).Error
}

func (c *ClickHouseObservability) AppendRuntimeEvent(ctx context.Context, event RuntimeEvent) error {
	if c == nil || c.db == nil {
		return nil
	}
	observedAt := event.ObservedAt.UTC()
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	record := clickHouseRuntimeEventRecord{
		ProjectSlug: c.project,
		IssueID:     strings.TrimSpace(event.IssueID),
		Identifier:  strings.TrimSpace(event.Identifier),
		Name:        strings.TrimSpace(event.Name),
		Message:     strings.TrimSpace(event.Message),
		ObservedAt:  observedAt,
		SessionID:   strings.TrimSpace(event.SessionID),
		Workspace:   strings.TrimSpace(event.Workspace),
		LogPath:     strings.TrimSpace(event.LogPath),
		Metadata:    strings.TrimSpace(string(event.MetadataRaw)),
	}
	return c.db.WithContext(ctx).Create(&record).Error
}

func (c *ClickHouseObservability) ListRuntimeEvents(
	ctx context.Context,
	query RuntimeEventQuery,
) ([]RuntimeEvent, error) {
	if c == nil || c.db == nil {
		return []RuntimeEvent{}, nil
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	db := c.db.WithContext(ctx).Model(&clickHouseRuntimeEventRecord{}).
		Where("project_slug = ?", c.project).
		Limit(limit).
		Order("observed_at DESC")
	if query.IssueID != "" {
		db = db.Where("issue_id = ?", query.IssueID)
	}
	if query.Identifier != "" {
		db = db.Where("identifier = ?", query.Identifier)
	}
	var records []clickHouseRuntimeEventRecord
	if err := db.Find(&records).Error; err != nil {
		return nil, err
	}
	events := make([]RuntimeEvent, 0, len(records))
	for _, record := range records {
		events = append(events, RuntimeEvent{
			IssueID:     record.IssueID,
			Identifier:  record.Identifier,
			Name:        record.Name,
			Message:     record.Message,
			ObservedAt:  record.ObservedAt.UTC(),
			SessionID:   record.SessionID,
			Workspace:   record.Workspace,
			LogPath:     record.LogPath,
			MetadataRaw: []byte(record.Metadata),
		})
	}
	slices.Reverse(events)
	return events, nil
}
