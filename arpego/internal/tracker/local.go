package tracker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ErrTaskValidation = "task_validation_error"
	ErrTaskNotFound   = "task_not_found"
)

type TaskError struct {
	Code    string
	Message string
}

func (e *TaskError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type CreateTaskInput struct {
	Title       string   `json:"title"`
	Description *string  `json:"description,omitempty"`
	State       string   `json:"state,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

type UpdateTaskInput struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	State       *string   `json:"state,omitempty"`
	Priority    *int      `json:"priority,omitempty"`
	Labels      *[]string `json:"labels,omitempty"`
}

type localStore struct {
	Tasks []Issue `yaml:"tasks"`
}

type LocalPlatform struct {
	path   string
	prefix string
	mu     sync.Mutex
}

func NewLocalPlatform(path, prefix string) *LocalPlatform {
	prefix = normalizeIdentifierPrefix(prefix)
	if prefix == "" {
		prefix = "SYM"
	}
	return &LocalPlatform{path: path, prefix: prefix}
}

func (p *LocalPlatform) FetchCandidates(_ context.Context, activeStates []string) ([]Issue, error) {
	return p.filterByStates(activeStates)
}

func (p *LocalPlatform) FetchByStates(_ context.Context, states []string) ([]Issue, error) {
	return p.filterByStates(states)
}

func (p *LocalPlatform) FetchStatesByIDs(_ context.Context, ids []string) ([]Issue, error) {
	if len(ids) == 0 {
		return []Issue{}, nil
	}
	issues, err := p.ListTasks(context.Background())
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]bool, len(ids))
	for _, id := range ids {
		allowed[id] = true
	}
	filtered := make([]Issue, 0, len(ids))
	for _, issue := range issues {
		if allowed[issue.ID] {
			filtered = append(filtered, issue)
		}
	}
	return filtered, nil
}

func (p *LocalPlatform) ListTasks(_ context.Context) ([]Issue, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	store, err := p.loadLocked()
	if err != nil {
		return nil, err
	}
	issues := append([]Issue(nil), store.Tasks...)
	sortIssues(issues)
	return issues, nil
}

func (p *LocalPlatform) CreateTask(_ context.Context, input CreateTaskInput) (Issue, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Issue{}, &TaskError{Code: ErrTaskValidation, Message: "title is required"}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	store, err := p.loadLocked()
	if err != nil {
		return Issue{}, err
	}

	number := nextTaskNumber(store.Tasks, p.prefix)
	now := time.Now().UTC()
	issue := Issue{
		ID:          fmt.Sprintf("task-%d", number),
		Identifier:  fmt.Sprintf("%s-%d", p.prefix, number),
		Title:       title,
		Description: normalizeOptionalString(input.Description),
		Priority:    input.Priority,
		State:       defaultTaskState(input.State),
		Labels:      normalizeLabels(input.Labels),
		BlockedBy:   []BlockerRef{},
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	store.Tasks = append(store.Tasks, issue)
	if err := p.saveLocked(store); err != nil {
		return Issue{}, err
	}
	return issue, nil
}

func (p *LocalPlatform) UpdateTask(_ context.Context, identifier string, input UpdateTaskInput) (Issue, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	store, err := p.loadLocked()
	if err != nil {
		return Issue{}, err
	}
	for i := range store.Tasks {
		if store.Tasks[i].Identifier != identifier {
			continue
		}
		if input.Title != nil {
			title := strings.TrimSpace(*input.Title)
			if title == "" {
				return Issue{}, &TaskError{Code: ErrTaskValidation, Message: "title must not be empty"}
			}
			store.Tasks[i].Title = title
		}
		if input.Description != nil {
			store.Tasks[i].Description = normalizeOptionalString(input.Description)
		}
		if input.State != nil {
			store.Tasks[i].State = defaultTaskState(*input.State)
		}
		if input.Priority != nil {
			store.Tasks[i].Priority = input.Priority
		}
		if input.Labels != nil {
			store.Tasks[i].Labels = normalizeLabels(*input.Labels)
		}
		now := time.Now().UTC()
		store.Tasks[i].UpdatedAt = &now
		if err := p.saveLocked(store); err != nil {
			return Issue{}, err
		}
		return store.Tasks[i], nil
	}
	return Issue{}, &TaskError{Code: ErrTaskNotFound, Message: "task not found"}
}

func (p *LocalPlatform) filterByStates(states []string) ([]Issue, error) {
	issues, err := p.ListTasks(context.Background())
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]bool, len(states))
	for _, state := range states {
		allowed[normalizeTaskState(state)] = true
	}
	filtered := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		if allowed[normalizeTaskState(issue.State)] {
			filtered = append(filtered, issue)
		}
	}
	return filtered, nil
}

func (p *LocalPlatform) loadLocked() (localStore, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return localStore{Tasks: []Issue{}}, nil
		}
		return localStore{}, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return localStore{Tasks: []Issue{}}, nil
	}
	var store localStore
	if err := yaml.Unmarshal(data, &store); err != nil {
		return localStore{}, err
	}
	if store.Tasks == nil {
		store.Tasks = []Issue{}
	}
	return store, nil
}

func (p *LocalPlatform) saveLocked(store localStore) error {
	data, err := yaml.Marshal(store)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return err
	}
	tmpPath := p.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, p.path)
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeLabels(labels []string) []string {
	out := make([]string, 0, len(labels))
	seen := map[string]struct{}{}
	for _, label := range labels {
		normalized := strings.ToLower(strings.TrimSpace(label))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	slices.Sort(out)
	return out
}

func defaultTaskState(state string) string {
	state = strings.TrimSpace(state)
	if state == "" {
		return "Todo"
	}
	return state
}

func normalizeTaskState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}

func nextTaskNumber(tasks []Issue, prefix string) int {
	next := 1
	for _, task := range tasks {
		if value := parseIdentifierNumber(task.Identifier, prefix); value >= next {
			next = value + 1
		}
	}
	return next
}

var identifierNumberPattern = regexp.MustCompile(`^([A-Z0-9]+)-([0-9]+)$`)
var invalidPrefixChars = regexp.MustCompile(`[^A-Z0-9]+`)

func parseIdentifierNumber(identifier, prefix string) int {
	matches := identifierNumberPattern.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(identifier)))
	if len(matches) != 3 || matches[1] != prefix {
		return 0
	}
	var value int
	_, _ = fmt.Sscanf(matches[2], "%d", &value)
	return value
}

func normalizeIdentifierPrefix(prefix string) string {
	return invalidPrefixChars.ReplaceAllString(strings.ToUpper(strings.TrimSpace(prefix)), "")
}

func sortIssues(issues []Issue) {
	slices.SortStableFunc(issues, func(a, b Issue) int {
		aPriority := 5
		if a.Priority != nil && *a.Priority >= 1 && *a.Priority <= 4 {
			aPriority = *a.Priority
		}
		bPriority := 5
		if b.Priority != nil && *b.Priority >= 1 && *b.Priority <= 4 {
			bPriority = *b.Priority
		}
		switch {
		case aPriority != bPriority:
			return aPriority - bPriority
		case a.CreatedAt != nil && b.CreatedAt != nil && !a.CreatedAt.Equal(*b.CreatedAt):
			if a.CreatedAt.Before(*b.CreatedAt) {
				return -1
			}
			return 1
		case a.Identifier < b.Identifier:
			return -1
		case a.Identifier > b.Identifier:
			return 1
		default:
			return 0
		}
	})
}
