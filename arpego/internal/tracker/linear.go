package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	Endpoint    string
	APIKey      string
	ProjectSlug string
	HTTPClient  *http.Client
}

func (c Client) FetchCandidates(ctx context.Context, activeStates []string) ([]Issue, error) {
	var issues []Issue
	var after any

	for {
		payload, err := c.query(ctx, candidateQuery, map[string]any{
			"projectSlug": c.ProjectSlug,
			"states":      activeStates,
			"after":       after,
		})
		if err != nil {
			return nil, err
		}

		page, ok := digMap(payload, "data", "issues")
		if !ok {
			return nil, &Error{Kind: ErrLinearUnknownPayload, Message: "missing data.issues"}
		}
		nodes, _ := page["nodes"].([]any)
		for _, node := range nodes {
			nodeMap, _ := node.(map[string]any)
			issue, err := NormalizeIssue(nodeMap)
			if err != nil {
				return nil, err
			}
			issues = append(issues, issue)
		}

		pageInfo, _ := page["pageInfo"].(map[string]any)
		hasNext, _ := pageInfo["hasNextPage"].(bool)
		if !hasNext {
			return issues, nil
		}
		after, ok = pageInfo["endCursor"]
		if !ok || after == nil || after == "" {
			return nil, &Error{Kind: ErrLinearUnknownPayload, Message: "missing endCursor for next page"}
		}
	}
}

func (c Client) FetchByStates(ctx context.Context, states []string) ([]Issue, error) {
	payload, err := c.query(ctx, statesQuery, map[string]any{
		"projectSlug": c.ProjectSlug,
		"states":      states,
	})
	if err != nil {
		return nil, err
	}
	return parseIssues(payload)
}

func (c Client) FetchStatesByIDs(ctx context.Context, ids []string) ([]Issue, error) {
	if len(ids) == 0 {
		return []Issue{}, nil
	}
	payload, err := c.query(ctx, idsQuery, map[string]any{
		"ids": ids,
	})
	if err != nil {
		return nil, err
	}
	return parseIssues(payload)
}

// RawQuery executes a GraphQL request and returns the full decoded response payload.
// Unlike query(), it does NOT treat GraphQL errors in the body as a failure — the
// caller receives the full payload and can inspect payload["errors"] itself.
// Only transport-level failures (network errors, non-200 HTTP status, decode failures)
// return a non-nil error.
func (c Client) RawQuery(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return nil, &Error{Kind: ErrLinearUnknownPayload, Message: "marshal GraphQL request", Cause: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, &Error{Kind: ErrLinearAPIRequest, Message: "build request", Cause: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, &Error{Kind: ErrLinearAPIRequest, Message: "perform request", Cause: err}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &Error{
			Kind:    ErrLinearAPIStatus,
			Message: fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(body)),
		}
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, &Error{Kind: ErrLinearUnknownPayload, Message: "decode response", Cause: err}
	}
	return payload, nil
}

func (c Client) query(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	payload, err := c.RawQuery(ctx, query, variables)
	if err != nil {
		return nil, err
	}
	if errs, ok := payload["errors"]; ok && errs != nil {
		return nil, &Error{Kind: ErrLinearGraphQLErrors, Message: "graphql errors returned"}
	}
	return payload, nil
}

func parseIssues(payload map[string]any) ([]Issue, error) {
	page, ok := digMap(payload, "data", "issues")
	if !ok {
		return nil, &Error{Kind: ErrLinearUnknownPayload, Message: "missing data.issues"}
	}
	nodes, _ := page["nodes"].([]any)
	issues := make([]Issue, 0, len(nodes))
	for _, node := range nodes {
		nodeMap, _ := node.(map[string]any)
		issue, err := NormalizeIssue(nodeMap)
		if err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

func digMap(root map[string]any, keys ...string) (map[string]any, bool) {
	current := root
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

const candidateQuery = `
query CandidateIssues($projectSlug: String!, $states: [String!], $after: String) {
  issues(
    filter: {
      project: { slugId: { eq: $projectSlug } }
      state: { name: { in: $states } }
    }
    first: 50
    after: $after
  ) {
    nodes {
      id
      identifier
      title
      description
      priority
      branchName
      url
      createdAt
      updatedAt
      state { name }
      labels { nodes { name } }
      relations { nodes { type issue { id identifier state { name } } } }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}`

const statesQuery = `
query IssuesByStates($projectSlug: String!, $states: [String!]) {
  issues(
    filter: {
      project: { slugId: { eq: $projectSlug } }
      state: { name: { in: $states } }
    }
    first: 50
  ) {
    nodes {
      id
      identifier
      title
      description
      priority
      branchName
      url
      createdAt
      updatedAt
      state { name }
      labels { nodes { name } }
      relations { nodes { type issue { id identifier state { name } } } }
    }
  }
}`

const idsQuery = `
query IssuesByIDs($ids: [ID!]) {
  issues(filter: { id: { in: $ids } }, first: 50) {
    nodes {
      id
      identifier
      title
      description
      priority
      branchName
      url
      createdAt
      updatedAt
      state { name }
      labels { nodes { name } }
      relations { nodes { type issue { id identifier state { name } } } }
    }
  }
}`
