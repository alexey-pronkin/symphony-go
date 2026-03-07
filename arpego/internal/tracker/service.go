package tracker

import (
	"context"
	"errors"
	"time"
)

var ErrTaskPlatformUnavailable = errors.New("task platform is unavailable")

type TaskRepository interface {
	CreateTask(context.Context, CreateTaskInput) (Issue, error)
	UpdateTask(context.Context, string, UpdateTaskInput) (Issue, error)
}

type TaskQueryPort interface {
	ListTasks(context.Context) ([]Issue, error)
	FetchCandidates(context.Context, []string) ([]Issue, error)
	FetchByStates(context.Context, []string) ([]Issue, error)
	FetchStatesByIDs(context.Context, []string) ([]Issue, error)
}

type RuntimeEvent struct {
	IssueID     string
	Identifier  string
	Name        string
	Message     string
	ObservedAt  time.Time
	SessionID   string
	Workspace   string
	MetadataRaw []byte
}

type RuntimeEventQuery struct {
	IssueID    string
	Identifier string
	Limit      int
}

type RuntimeEventSink interface {
	AppendRuntimeEvent(context.Context, RuntimeEvent) error
}

type RuntimeEventStore interface {
	ListRuntimeEvents(context.Context, RuntimeEventQuery) ([]RuntimeEvent, error)
}

type MetricSample struct {
	Name       string
	Value      float64
	ObservedAt time.Time
	Labels     map[string]string
}

type MetricsExporter interface {
	ExportMetric(context.Context, MetricSample) error
}

type TaskService struct {
	repo    TaskRepository
	queries TaskQueryPort
}

func NewTaskService(repo TaskRepository, queries TaskQueryPort) *TaskService {
	return &TaskService{repo: repo, queries: queries}
}

func (s *TaskService) ListTasks(ctx context.Context) ([]Issue, error) {
	if s == nil || s.queries == nil {
		return nil, ErrTaskPlatformUnavailable
	}
	return s.queries.ListTasks(ctx)
}

func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (Issue, error) {
	if s == nil || s.repo == nil {
		return Issue{}, ErrTaskPlatformUnavailable
	}
	return s.repo.CreateTask(ctx, input)
}

func (s *TaskService) UpdateTask(ctx context.Context, identifier string, input UpdateTaskInput) (Issue, error) {
	if s == nil || s.repo == nil {
		return Issue{}, ErrTaskPlatformUnavailable
	}
	return s.repo.UpdateTask(ctx, identifier, input)
}

func (s *TaskService) FetchCandidates(ctx context.Context, states []string) ([]Issue, error) {
	if s == nil || s.queries == nil {
		return nil, ErrTaskPlatformUnavailable
	}
	return s.queries.FetchCandidates(ctx, states)
}

func (s *TaskService) FetchByStates(ctx context.Context, states []string) ([]Issue, error) {
	if s == nil || s.queries == nil {
		return nil, ErrTaskPlatformUnavailable
	}
	return s.queries.FetchByStates(ctx, states)
}

func (s *TaskService) FetchStatesByIDs(ctx context.Context, ids []string) ([]Issue, error) {
	if s == nil || s.queries == nil {
		return nil, ErrTaskPlatformUnavailable
	}
	return s.queries.FetchStatesByIDs(ctx, ids)
}
