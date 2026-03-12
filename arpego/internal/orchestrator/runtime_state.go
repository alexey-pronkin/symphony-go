package orchestrator

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/agent"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type RuntimeStateStore interface {
	EnsureSchema(context.Context) error
	LoadState(context.Context) (PersistedRuntimeState, error)
	UpsertRetry(context.Context, PersistedRetryEntry) error
	DeleteRetry(context.Context, string) error
	UpsertRunning(context.Context, PersistedRunningEntry) error
	DeleteRunning(context.Context, string) error
}

type PersistedRuntimeState struct {
	Retrying []PersistedRetryEntry
	Running  []PersistedRunningEntry
}

type PersistedRetryEntry struct {
	IssueID       string
	Identifier    string
	Attempt       int
	DueAt         time.Time
	Error         string
	Continuation  bool
	RestartCount  int
	RetryAttempt  int
	WorkspacePath string
}

type PersistedRunningEntry struct {
	IssueID         string
	Identifier      string
	State           string
	WorkspacePath   string
	StartedAt       time.Time
	LastEventAt     *time.Time
	LastEvent       string
	LastMessage     string
	SessionID       string
	ThreadID        string
	TurnID          string
	TurnCount       int
	RetryAttempt    int
	SessionLog      string
	InputTokens     int
	OutputTokens    int
	TotalTokens     int
	RecentEvents    []IssueEvent
	TrackedMetadata map[string]any
}

type RuntimeStateStatus struct {
	Enabled   bool    `json:"enabled"`
	Status    string  `json:"status"`
	LastError *string `json:"last_error,omitempty"`
}

func (s RuntimeStateStatus) clone() *RuntimeStateStatus {
	if !s.Enabled && s.Status == "" && s.LastError == nil {
		return nil
	}
	out := s
	if s.LastError != nil {
		value := *s.LastError
		out.LastError = &value
	}
	return &out
}

type PostgresRuntimeStateStore struct {
	db      *gorm.DB
	closer  io.Closer
	project string
}

type postgresRetryRecord struct {
	ProjectSlug   string    `gorm:"primaryKey;column:project_slug"`
	IssueID       string    `gorm:"primaryKey;column:issue_id"`
	Identifier    string    `gorm:"column:identifier;not null"`
	Attempt       int       `gorm:"column:attempt;not null"`
	DueAt         time.Time `gorm:"column:due_at;not null"`
	Error         string    `gorm:"column:error_message"`
	Continuation  bool      `gorm:"column:is_continuation;not null"`
	RestartCount  int       `gorm:"column:restart_count;not null"`
	RetryAttempt  int       `gorm:"column:retry_attempt;not null"`
	WorkspacePath string    `gorm:"column:workspace_path"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (postgresRetryRecord) TableName() string {
	return "runtime_retry_entries"
}

type postgresRunningRecord struct {
	ProjectSlug   string     `gorm:"primaryKey;column:project_slug"`
	IssueID       string     `gorm:"primaryKey;column:issue_id"`
	Identifier    string     `gorm:"column:identifier;not null"`
	State         string     `gorm:"column:state_name;not null"`
	WorkspacePath string     `gorm:"column:workspace_path;not null"`
	StartedAt     time.Time  `gorm:"column:started_at;not null"`
	LastEventAt   *time.Time `gorm:"column:last_event_at"`
	LastEvent     string     `gorm:"column:last_event"`
	LastMessage   string     `gorm:"column:last_message"`
	SessionID     string     `gorm:"column:session_id"`
	ThreadID      string     `gorm:"column:thread_id"`
	TurnID        string     `gorm:"column:turn_id"`
	TurnCount     int        `gorm:"column:turn_count;not null"`
	RetryAttempt  int        `gorm:"column:retry_attempt;not null"`
	SessionLog    string     `gorm:"column:session_log"`
	InputTokens   int        `gorm:"column:input_tokens;not null"`
	OutputTokens  int        `gorm:"column:output_tokens;not null"`
	TotalTokens   int        `gorm:"column:total_tokens;not null"`
	RecentEvents  []byte     `gorm:"column:recent_events;type:jsonb;not null"`
	TrackedJSON   []byte     `gorm:"column:tracked;type:jsonb;not null"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (postgresRunningRecord) TableName() string {
	return "runtime_running_entries"
}

func OpenPostgresRuntimeStateStore(ctx context.Context, dsn, project string) (*PostgresRuntimeStateStore, error) {
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
	store := NewPostgresRuntimeStateStore(db, project)
	store.closer = sqlDB
	if err := store.EnsureSchema(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return store, nil
}

func NewPostgresRuntimeStateStore(db *gorm.DB, project string) *PostgresRuntimeStateStore {
	return &PostgresRuntimeStateStore{db: db, project: project}
}

func (s *PostgresRuntimeStateStore) Close() error {
	if s == nil || s.closer == nil {
		return nil
	}
	return s.closer.Close()
}

func (s *PostgresRuntimeStateStore) EnsureSchema(ctx context.Context) error {
	return s.db.WithContext(ctx).AutoMigrate(&postgresRetryRecord{}, &postgresRunningRecord{})
}

func (s *PostgresRuntimeStateStore) LoadState(ctx context.Context) (PersistedRuntimeState, error) {
	var retries []postgresRetryRecord
	if err := s.db.WithContext(ctx).
		Where("project_slug = ?", s.project).
		Order("due_at, issue_id").
		Find(&retries).Error; err != nil {
		return PersistedRuntimeState{}, err
	}
	var running []postgresRunningRecord
	if err := s.db.WithContext(ctx).
		Where("project_slug = ?", s.project).
		Order("started_at, issue_id").
		Find(&running).Error; err != nil {
		return PersistedRuntimeState{}, err
	}
	out := PersistedRuntimeState{
		Retrying: make([]PersistedRetryEntry, 0, len(retries)),
		Running:  make([]PersistedRunningEntry, 0, len(running)),
	}
	for _, record := range retries {
		out.Retrying = append(out.Retrying, PersistedRetryEntry{
			IssueID:       record.IssueID,
			Identifier:    record.Identifier,
			Attempt:       record.Attempt,
			DueAt:         record.DueAt,
			Error:         record.Error,
			Continuation:  record.Continuation,
			RestartCount:  record.RestartCount,
			RetryAttempt:  record.RetryAttempt,
			WorkspacePath: record.WorkspacePath,
		})
	}
	for _, record := range running {
		entry, err := decodePersistedRunning(record)
		if err != nil {
			return PersistedRuntimeState{}, err
		}
		out.Running = append(out.Running, entry)
	}
	return out, nil
}

func (s *PostgresRuntimeStateStore) UpsertRetry(ctx context.Context, entry PersistedRetryEntry) error {
	record := postgresRetryRecord{
		ProjectSlug:   s.project,
		IssueID:       entry.IssueID,
		Identifier:    entry.Identifier,
		Attempt:       entry.Attempt,
		DueAt:         entry.DueAt,
		Error:         entry.Error,
		Continuation:  entry.Continuation,
		RestartCount:  entry.RestartCount,
		RetryAttempt:  entry.RetryAttempt,
		WorkspacePath: entry.WorkspacePath,
	}
	return s.db.WithContext(ctx).Save(&record).Error
}

func (s *PostgresRuntimeStateStore) DeleteRetry(ctx context.Context, issueID string) error {
	return s.db.WithContext(ctx).
		Delete(&postgresRetryRecord{}, "project_slug = ? AND issue_id = ?", s.project, issueID).Error
}

func (s *PostgresRuntimeStateStore) UpsertRunning(ctx context.Context, entry PersistedRunningEntry) error {
	record, err := encodePersistedRunning(s.project, entry)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Save(&record).Error
}

func (s *PostgresRuntimeStateStore) DeleteRunning(ctx context.Context, issueID string) error {
	return s.db.WithContext(ctx).
		Delete(&postgresRunningRecord{}, "project_slug = ? AND issue_id = ?", s.project, issueID).Error
}

func encodePersistedRunning(project string, entry PersistedRunningEntry) (postgresRunningRecord, error) {
	recentEvents, err := json.Marshal(entry.RecentEvents)
	if err != nil {
		return postgresRunningRecord{}, err
	}
	tracked := entry.TrackedMetadata
	if tracked == nil {
		tracked = map[string]any{}
	}
	trackedJSON, err := json.Marshal(tracked)
	if err != nil {
		return postgresRunningRecord{}, err
	}
	return postgresRunningRecord{
		ProjectSlug:   project,
		IssueID:       entry.IssueID,
		Identifier:    entry.Identifier,
		State:         entry.State,
		WorkspacePath: entry.WorkspacePath,
		StartedAt:     entry.StartedAt,
		LastEventAt:   entry.LastEventAt,
		LastEvent:     entry.LastEvent,
		LastMessage:   entry.LastMessage,
		SessionID:     entry.SessionID,
		ThreadID:      entry.ThreadID,
		TurnID:        entry.TurnID,
		TurnCount:     entry.TurnCount,
		RetryAttempt:  entry.RetryAttempt,
		SessionLog:    entry.SessionLog,
		InputTokens:   entry.InputTokens,
		OutputTokens:  entry.OutputTokens,
		TotalTokens:   entry.TotalTokens,
		RecentEvents:  recentEvents,
		TrackedJSON:   trackedJSON,
	}, nil
}

func decodePersistedRunning(record postgresRunningRecord) (PersistedRunningEntry, error) {
	var recentEvents []IssueEvent
	if len(record.RecentEvents) > 0 {
		if err := json.Unmarshal(record.RecentEvents, &recentEvents); err != nil {
			return PersistedRunningEntry{}, err
		}
	}
	var tracked map[string]any
	if len(record.TrackedJSON) > 0 {
		if err := json.Unmarshal(record.TrackedJSON, &tracked); err != nil {
			return PersistedRunningEntry{}, err
		}
	}
	return PersistedRunningEntry{
		IssueID:         record.IssueID,
		Identifier:      record.Identifier,
		State:           record.State,
		WorkspacePath:   record.WorkspacePath,
		StartedAt:       record.StartedAt,
		LastEventAt:     record.LastEventAt,
		LastEvent:       record.LastEvent,
		LastMessage:     record.LastMessage,
		SessionID:       record.SessionID,
		ThreadID:        record.ThreadID,
		TurnID:          record.TurnID,
		TurnCount:       record.TurnCount,
		RetryAttempt:    record.RetryAttempt,
		SessionLog:      record.SessionLog,
		InputTokens:     record.InputTokens,
		OutputTokens:    record.OutputTokens,
		TotalTokens:     record.TotalTokens,
		RecentEvents:    recentEvents,
		TrackedMetadata: tracked,
	}, nil
}

func runtimeStateContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func (o *Orchestrator) markRuntimeStateError(err error) {
	if err == nil {
		o.state.RuntimeState.Status = "ok"
		o.state.RuntimeState.LastError = nil
		return
	}
	o.state.RuntimeState.Enabled = o.runtimeState != nil
	o.state.RuntimeState.Status = "degraded"
	message := err.Error()
	o.state.RuntimeState.LastError = &message
}

func (o *Orchestrator) persistRetryLocked(entry RetryEntry) {
	if o.runtimeState == nil {
		return
	}
	err := o.runtimeState.UpsertRetry(runtimeStateContext(o.ctx), PersistedRetryEntry{
		IssueID:       entry.IssueID,
		Identifier:    entry.Identifier,
		Attempt:       entry.Attempt,
		DueAt:         entry.DueAt,
		Error:         entry.Error,
		Continuation:  entry.Continuation,
		RestartCount:  entry.RestartCount,
		RetryAttempt:  entry.RetryAttempt,
		WorkspacePath: entry.WorkspacePath,
	})
	o.markRuntimeStateError(err)
}

func (o *Orchestrator) deleteRetryLocked(issueID string) {
	if o.runtimeState == nil {
		return
	}
	o.markRuntimeStateError(o.runtimeState.DeleteRetry(runtimeStateContext(o.ctx), issueID))
}

func (o *Orchestrator) persistRunningLocked(entry *RunningEntry) {
	if o.runtimeState == nil || entry == nil {
		return
	}
	o.markRuntimeStateError(o.runtimeState.UpsertRunning(runtimeStateContext(o.ctx), persistedRunningFromEntry(entry)))
}

func (o *Orchestrator) deleteRunningLocked(issueID string) {
	if o.runtimeState == nil {
		return
	}
	o.markRuntimeStateError(o.runtimeState.DeleteRunning(runtimeStateContext(o.ctx), issueID))
}

func persistedRunningFromEntry(entry *RunningEntry) PersistedRunningEntry {
	if entry == nil {
		return PersistedRunningEntry{}
	}
	return PersistedRunningEntry{
		IssueID:       entry.Issue.ID,
		Identifier:    entry.Issue.Identifier,
		State:         entry.Issue.State,
		WorkspacePath: entry.WorkspacePath,
		StartedAt:     entry.StartedAt,
		LastEventAt:   entry.LastEventAt,
		LastEvent:     entry.LastEvent,
		LastMessage:   entry.LastMessage,
		SessionID:     entry.SessionID,
		ThreadID:      entry.ThreadID,
		TurnID:        entry.TurnID,
		TurnCount:     entry.TurnCount,
		RetryAttempt:  entry.RetryAttempt,
		SessionLog:    entry.SessionLog,
		InputTokens:   entry.CurrentUsage.InputTokens,
		OutputTokens:  entry.CurrentUsage.OutputTokens,
		TotalTokens:   entry.CurrentUsage.TotalTokens,
		RecentEvents:  append([]IssueEvent(nil), entry.RecentEvents...),
		TrackedMetadata: map[string]any{
			"thread_id": entry.ThreadID,
			"turn_id":   entry.TurnID,
		},
	}
}

func restoreEntryFromPersisted(entry PersistedRunningEntry) *RunningEntry {
	return &RunningEntry{
		Issue:         trackerIssueFromPersisted(entry),
		WorkspacePath: entry.WorkspacePath,
		StartedAt:     entry.StartedAt,
		LastEventAt:   entry.LastEventAt,
		LastEvent:     entry.LastEvent,
		LastMessage:   entry.LastMessage,
		SessionID:     entry.SessionID,
		ThreadID:      entry.ThreadID,
		TurnID:        entry.TurnID,
		TurnCount:     entry.TurnCount,
		RetryAttempt:  entry.RetryAttempt,
		CurrentUsage:  agentUsageFromPersisted(entry),
		RecentEvents:  append([]IssueEvent(nil), entry.RecentEvents...),
		SessionLog:    entry.SessionLog,
	}
}

func trackerIssueFromPersisted(entry PersistedRunningEntry) tracker.Issue {
	return tracker.Issue{
		ID:         entry.IssueID,
		Identifier: entry.Identifier,
		State:      entry.State,
	}
}

func agentUsageFromPersisted(entry PersistedRunningEntry) agent.Usage {
	return agent.Usage{
		InputTokens:  entry.InputTokens,
		OutputTokens: entry.OutputTokens,
		TotalTokens:  entry.TotalTokens,
	}
}

func (o *Orchestrator) restoreRuntimeState(ctx context.Context) error {
	if o.runtimeState == nil {
		return nil
	}
	state, err := o.runtimeState.LoadState(ctx)
	if err != nil {
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.state.RuntimeState.Enabled = true
	o.state.RuntimeState.Status = "ok"
	o.state.RuntimeState.LastError = nil
	for _, retry := range state.Retrying {
		entry := RetryEntry{
			IssueID:       retry.IssueID,
			Identifier:    retry.Identifier,
			Attempt:       retry.Attempt,
			DueAt:         retry.DueAt,
			Error:         retry.Error,
			Continuation:  retry.Continuation,
			RestartCount:  retry.RestartCount,
			RetryAttempt:  retry.RetryAttempt,
			WorkspacePath: retry.WorkspacePath,
		}
		o.installRetryTimerLocked(&entry)
		o.state.RetryAttempts[retry.IssueID] = entry
		o.state.Claimed[retry.IssueID] = struct{}{}
	}
	for _, running := range state.Running {
		entry := restoreEntryFromPersisted(running)
		o.state.Running[running.IssueID] = entry
		o.state.Claimed[running.IssueID] = struct{}{}
	}
	return nil
}

func (o *Orchestrator) installRetryTimerLocked(retry *RetryEntry) {
	if retry == nil {
		return
	}
	delay := time.Until(retry.DueAt)
	if o.now != nil {
		delay = retry.DueAt.Sub(o.now())
	}
	if delay < 0 {
		delay = 0
	}
	retry.Timer = o.afterFunc(delay, func() {
		select {
		case o.retryCh <- retry.IssueID:
		default:
			go func() { o.retryCh <- retry.IssueID }()
		}
	})
}
