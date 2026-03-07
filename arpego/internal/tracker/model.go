package tracker

import (
	"fmt"
	"strings"
	"time"
)

const (
	ErrLinearAPIRequest     = "linear_api_request"
	ErrLinearAPIStatus      = "linear_api_status"
	ErrLinearGraphQLErrors  = "linear_graphql_errors"
	ErrLinearUnknownPayload = "linear_unknown_payload"
)

type Error struct {
	Kind    string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

type BlockerRef struct {
	ID         *string `json:"id,omitempty" yaml:"id,omitempty"`
	Identifier *string `json:"identifier,omitempty" yaml:"identifier,omitempty"`
	State      *string `json:"state,omitempty" yaml:"state,omitempty"`
}

type Issue struct {
	ID          string       `json:"id" yaml:"id"`
	Identifier  string       `json:"identifier" yaml:"identifier"`
	Title       string       `json:"title" yaml:"title"`
	Description *string      `json:"description,omitempty" yaml:"description,omitempty"`
	Priority    *int         `json:"priority,omitempty" yaml:"priority,omitempty"`
	State       string       `json:"state" yaml:"state"`
	BranchName  *string      `json:"branch_name,omitempty" yaml:"branch_name,omitempty"`
	URL         *string      `json:"url,omitempty" yaml:"url,omitempty"`
	Labels      []string     `json:"labels" yaml:"labels"`
	BlockedBy   []BlockerRef `json:"blocked_by" yaml:"blocked_by"`
	CreatedAt   *time.Time   `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt   *time.Time   `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}

func NormalizeIssue(raw map[string]any) (Issue, error) {
	issue := Issue{
		ID:         asString(raw["id"]),
		Identifier: asString(raw["identifier"]),
		Title:      asString(raw["title"]),
		State:      extractStateName(raw["state"]),
		Labels:     extractLabels(raw["labels"]),
		BlockedBy:  extractBlockedBy(raw["relations"]),
	}

	if s := asOptionalString(raw["description"]); s != nil {
		issue.Description = s
	}
	if s := asOptionalString(raw["branchName"]); s != nil {
		issue.BranchName = s
	}
	if s := asOptionalString(raw["url"]); s != nil {
		issue.URL = s
	}
	if p := asOptionalInt(raw["priority"]); p != nil {
		issue.Priority = p
	}
	if ts, err := asOptionalTime(raw["createdAt"]); err != nil {
		return Issue{}, &Error{Kind: ErrLinearUnknownPayload, Message: "invalid createdAt", Cause: err}
	} else {
		issue.CreatedAt = ts
	}
	if ts, err := asOptionalTime(raw["updatedAt"]); err != nil {
		return Issue{}, &Error{Kind: ErrLinearUnknownPayload, Message: "invalid updatedAt", Cause: err}
	} else {
		issue.UpdatedAt = ts
	}

	if issue.ID == "" || issue.Identifier == "" || issue.Title == "" {
		return Issue{}, &Error{Kind: ErrLinearUnknownPayload, Message: "issue payload missing required fields"}
	}

	return issue, nil
}

func extractStateName(v any) string {
	m, _ := v.(map[string]any)
	return asString(m["name"])
}

func extractLabels(v any) []string {
	m, _ := v.(map[string]any)
	nodes, _ := m["nodes"].([]any)
	labels := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodeMap, _ := node.(map[string]any)
		name := strings.ToLower(strings.TrimSpace(asString(nodeMap["name"])))
		if name != "" {
			labels = append(labels, name)
		}
	}
	return labels
}

func extractBlockedBy(v any) []BlockerRef {
	m, _ := v.(map[string]any)
	nodes, _ := m["nodes"].([]any)
	blockers := make([]BlockerRef, 0, len(nodes))
	for _, node := range nodes {
		nodeMap, _ := node.(map[string]any)
		if asString(nodeMap["type"]) != "blocks" {
			continue
		}
		issueMap, _ := nodeMap["issue"].(map[string]any)
		stateName := extractStateName(issueMap["state"])
		blockers = append(blockers, BlockerRef{
			ID:         asOptionalString(issueMap["id"]),
			Identifier: asOptionalString(issueMap["identifier"]),
			State:      asOptionalString(stateName),
		})
	}
	return blockers
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asOptionalString(v any) *string {
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		return &t
	}
	return nil
}

func asOptionalInt(v any) *int {
	switch t := v.(type) {
	case int:
		return &t
	case int64:
		n := int(t)
		return &n
	case float64:
		n := int(t)
		return &n
	default:
		return nil
	}
}

func asOptionalTime(v any) (*time.Time, error) {
	s, _ := v.(string)
	if s == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &ts, nil
}
