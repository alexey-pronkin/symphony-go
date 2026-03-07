package tracker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

func TestTaskServiceDelegatesToPorts(t *testing.T) {
	queryIssues := []tracker.Issue{{ID: "task-1", Identifier: "SYM-1", Title: "One", State: "Todo"}}
	repo := &fakeTaskRepository{
		created: tracker.Issue{ID: "task-2", Identifier: "SYM-2", Title: "Created", State: "Todo"},
		updated: tracker.Issue{ID: "task-1", Identifier: "SYM-1", Title: "One", State: "Done"},
	}
	queries := &fakeTaskQueryPort{
		list:       queryIssues,
		candidates: queryIssues,
		byStates:   queryIssues,
		byIDs:      queryIssues,
	}
	service := tracker.NewTaskService(repo, queries)

	listed, err := service.ListTasks(context.Background())
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(listed) != 1 || listed[0].Identifier != "SYM-1" {
		t.Fatalf("listed = %#v", listed)
	}

	created, err := service.CreateTask(context.Background(), tracker.CreateTaskInput{Title: "Created"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if created.Identifier != "SYM-2" || repo.lastCreate.Title != "Created" {
		t.Fatalf("created = %#v lastCreate = %#v", created, repo.lastCreate)
	}

	state := "Done"
	updated, err := service.UpdateTask(context.Background(), "SYM-1", tracker.UpdateTaskInput{State: &state})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.State != "Done" || repo.lastIdentifier != "SYM-1" {
		t.Fatalf("updated = %#v lastIdentifier = %q", updated, repo.lastIdentifier)
	}

	candidates, err := service.FetchCandidates(context.Background(), []string{"Todo"})
	if err != nil {
		t.Fatalf("FetchCandidates: %v", err)
	}
	if len(candidates) != 1 || queries.lastCandidateStates[0] != "Todo" {
		t.Fatalf("candidates = %#v states = %#v", candidates, queries.lastCandidateStates)
	}
}

func TestTaskServiceUnavailableWhenPortsMissing(t *testing.T) {
	service := tracker.NewTaskService(nil, nil)

	if _, err := service.ListTasks(context.Background()); !errors.Is(err, tracker.ErrTaskPlatformUnavailable) {
		t.Fatalf("ListTasks err = %v", err)
	}
	if _, err := service.CreateTask(
		context.Background(),
		tracker.CreateTaskInput{Title: "x"},
	); !errors.Is(err, tracker.ErrTaskPlatformUnavailable) {
		t.Fatalf("CreateTask err = %v", err)
	}
	if _, err := service.FetchCandidates(
		context.Background(),
		[]string{"Todo"},
	); !errors.Is(err, tracker.ErrTaskPlatformUnavailable) {
		t.Fatalf("FetchCandidates err = %v", err)
	}
}

type fakeTaskRepository struct {
	created        tracker.Issue
	updated        tracker.Issue
	lastCreate     tracker.CreateTaskInput
	lastUpdate     tracker.UpdateTaskInput
	lastIdentifier string
}

func (f *fakeTaskRepository) CreateTask(_ context.Context, input tracker.CreateTaskInput) (tracker.Issue, error) {
	f.lastCreate = input
	return f.created, nil
}

func (f *fakeTaskRepository) UpdateTask(
	_ context.Context,
	identifier string,
	input tracker.UpdateTaskInput,
) (tracker.Issue, error) {
	f.lastIdentifier = identifier
	f.lastUpdate = input
	return f.updated, nil
}

type fakeTaskQueryPort struct {
	list                []tracker.Issue
	candidates          []tracker.Issue
	byStates            []tracker.Issue
	byIDs               []tracker.Issue
	lastCandidateStates []string
}

func (f *fakeTaskQueryPort) ListTasks(context.Context) ([]tracker.Issue, error) {
	return f.list, nil
}

func (f *fakeTaskQueryPort) FetchCandidates(_ context.Context, states []string) ([]tracker.Issue, error) {
	f.lastCandidateStates = append([]string(nil), states...)
	return f.candidates, nil
}

func (f *fakeTaskQueryPort) FetchByStates(context.Context, []string) ([]tracker.Issue, error) {
	return f.byStates, nil
}

func (f *fakeTaskQueryPort) FetchStatesByIDs(context.Context, []string) ([]tracker.Issue, error) {
	return f.byIDs, nil
}
