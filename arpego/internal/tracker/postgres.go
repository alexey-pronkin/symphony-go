package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresPlatform struct {
	db     *gorm.DB
	closer io.Closer
	prefix string
}

type postgresTaskSequence struct {
	Prefix    string `gorm:"primaryKey"`
	LastValue int64  `gorm:"not null"`
}

func (postgresTaskSequence) TableName() string {
	return "task_sequences"
}

type postgresTaskRecord struct {
	ID          string `gorm:"primaryKey"`
	Identifier  string `gorm:"uniqueIndex;not null"`
	Title       string `gorm:"not null"`
	Description *string
	Priority    *int
	State       string  `gorm:"not null"`
	BranchName  *string `gorm:"column:branch_name"`
	URL         *string `gorm:"column:url"`
	LabelsJSON  []byte  `gorm:"column:labels;type:jsonb;not null"`
	BlockedJSON []byte  `gorm:"column:blocked_by;type:jsonb;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (postgresTaskRecord) TableName() string {
	return "tasks"
}

func OpenPostgresPlatform(ctx context.Context, dsn, prefix string) (*PostgresPlatform, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
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
	platform := NewPostgresPlatform(db, prefix)
	platform.closer = sqlDB
	if err := platform.EnsureSchema(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return platform, nil
}

func NewPostgresPlatform(db *gorm.DB, prefix string) *PostgresPlatform {
	prefix = normalizeIdentifierPrefix(prefix)
	if prefix == "" {
		prefix = "SYM"
	}
	return &PostgresPlatform{db: db, prefix: prefix}
}

func (p *PostgresPlatform) Close() error {
	if p == nil || p.closer == nil {
		return nil
	}
	return p.closer.Close()
}

func (p *PostgresPlatform) EnsureSchema(ctx context.Context) error {
	return p.db.WithContext(ctx).AutoMigrate(&postgresTaskSequence{}, &postgresTaskRecord{})
}

func (p *PostgresPlatform) FetchCandidates(ctx context.Context, activeStates []string) ([]Issue, error) {
	return p.fetchByField(ctx, "state", activeStates)
}

func (p *PostgresPlatform) FetchByStates(ctx context.Context, states []string) ([]Issue, error) {
	return p.fetchByField(ctx, "state", states)
}

func (p *PostgresPlatform) FetchStatesByIDs(ctx context.Context, ids []string) ([]Issue, error) {
	return p.fetchByField(ctx, "id", ids)
}

func (p *PostgresPlatform) ListTasks(ctx context.Context) ([]Issue, error) {
	var records []postgresTaskRecord
	err := p.baseQuery(ctx).
		Order("COALESCE(priority, 5), created_at, identifier").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return decodeTaskRecords(records)
}

func (p *PostgresPlatform) CreateTask(ctx context.Context, input CreateTaskInput) (Issue, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Issue{}, &TaskError{Code: ErrTaskValidation, Message: "title is required"}
	}

	now := time.Now().UTC()
	issue := Issue{
		Title:       title,
		Description: normalizeOptionalString(input.Description),
		Priority:    input.Priority,
		State:       defaultTaskState(input.State),
		Labels:      normalizeLabels(input.Labels),
		BlockedBy:   []BlockerRef{},
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&postgresTaskSequence{Prefix: p.prefix, LastValue: 0}).Error; err != nil {
			return err
		}

		var seq postgresTaskSequence
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("prefix = ?", p.prefix).
			First(&seq).Error; err != nil {
			return err
		}

		seq.LastValue++
		if err := tx.Model(&seq).Update("last_value", seq.LastValue).Error; err != nil {
			return err
		}

		issue.ID = fmt.Sprintf("task-%s-%d", strings.ToLower(p.prefix), seq.LastValue)
		issue.Identifier = fmt.Sprintf("%s-%d", p.prefix, seq.LastValue)

		record, err := encodeTaskRecord(issue)
		if err != nil {
			return err
		}
		return tx.Create(&record).Error
	})
	if err != nil {
		return Issue{}, err
	}
	return issue, nil
}

func (p *PostgresPlatform) UpdateTask(ctx context.Context, identifier string, input UpdateTaskInput) (Issue, error) {
	var updated Issue
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record postgresTaskRecord
		if err := tx.Where("identifier = ?", identifier).First(&record).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return &TaskError{Code: ErrTaskNotFound, Message: "task not found"}
			}
			return err
		}

		issue, err := decodeTaskRecord(record)
		if err != nil {
			return err
		}

		if input.Title != nil {
			title := strings.TrimSpace(*input.Title)
			if title == "" {
				return &TaskError{Code: ErrTaskValidation, Message: "title must not be empty"}
			}
			issue.Title = title
		}
		if input.Description != nil {
			issue.Description = normalizeOptionalString(input.Description)
		}
		if input.State != nil {
			issue.State = defaultTaskState(*input.State)
		}
		if input.Priority != nil {
			issue.Priority = input.Priority
		}
		if input.Labels != nil {
			issue.Labels = normalizeLabels(*input.Labels)
		}
		now := time.Now().UTC()
		issue.UpdatedAt = &now

		record, err = encodeTaskRecord(issue)
		if err != nil {
			return err
		}
		if err := tx.Save(&record).Error; err != nil {
			return err
		}
		updated = issue
		return nil
	})
	if err != nil {
		return Issue{}, err
	}
	return updated, nil
}

func (p *PostgresPlatform) fetchByField(ctx context.Context, field string, values []string) ([]Issue, error) {
	if len(values) == 0 {
		return []Issue{}, nil
	}
	var records []postgresTaskRecord
	err := p.baseQuery(ctx).
		Where(fmt.Sprintf("%s IN ?", field), values).
		Order("COALESCE(priority, 5), created_at, identifier").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return decodeTaskRecords(records)
}

func (p *PostgresPlatform) baseQuery(ctx context.Context) *gorm.DB {
	return p.db.WithContext(ctx).Model(&postgresTaskRecord{})
}

func encodeTaskRecord(issue Issue) (postgresTaskRecord, error) {
	labelsJSON, err := json.Marshal(issue.Labels)
	if err != nil {
		return postgresTaskRecord{}, err
	}
	blockedJSON, err := json.Marshal(issue.BlockedBy)
	if err != nil {
		return postgresTaskRecord{}, err
	}
	record := postgresTaskRecord{
		ID:          issue.ID,
		Identifier:  issue.Identifier,
		Title:       issue.Title,
		Description: issue.Description,
		Priority:    issue.Priority,
		State:       issue.State,
		BranchName:  issue.BranchName,
		URL:         issue.URL,
		LabelsJSON:  labelsJSON,
		BlockedJSON: blockedJSON,
	}
	if issue.CreatedAt != nil {
		record.CreatedAt = *issue.CreatedAt
	}
	if issue.UpdatedAt != nil {
		record.UpdatedAt = *issue.UpdatedAt
	}
	return record, nil
}

func decodeTaskRecords(records []postgresTaskRecord) ([]Issue, error) {
	issues := make([]Issue, 0, len(records))
	for _, record := range records {
		issue, err := decodeTaskRecord(record)
		if err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

func decodeTaskRecord(record postgresTaskRecord) (Issue, error) {
	issue := Issue{
		ID:          record.ID,
		Identifier:  record.Identifier,
		Title:       record.Title,
		Description: record.Description,
		Priority:    record.Priority,
		State:       record.State,
		BranchName:  record.BranchName,
		URL:         record.URL,
		CreatedAt:   &record.CreatedAt,
		UpdatedAt:   &record.UpdatedAt,
	}
	if err := decodeJSONSlice(record.LabelsJSON, &issue.Labels); err != nil {
		return Issue{}, err
	}
	if err := decodeJSONSlice(record.BlockedJSON, &issue.BlockedBy); err != nil {
		return Issue{}, err
	}
	if issue.Labels == nil {
		issue.Labels = []string{}
	}
	if issue.BlockedBy == nil {
		issue.BlockedBy = []BlockerRef{}
	}
	return issue, nil
}

func decodeJSONSlice(data []byte, out any) error {
	if len(data) == 0 {
		data = []byte("[]")
	}
	return json.Unmarshal(data, out)
}
