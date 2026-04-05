package tracker_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

func TestNormalizeIssue(t *testing.T) {
	raw := map[string]any{
		"id":          "issue-1",
		"identifier":  "MT-101",
		"title":       "Fix tracker",
		"description": "desc",
		"priority":    2.0,
		"branchName":  "feat/branch",
		"url":         "https://linear.app/issue/MT-101",
		"createdAt":   "2026-03-07T12:00:00Z",
		"updatedAt":   "2026-03-07T13:00:00Z",
		"state": map[string]any{
			"name": "In Progress",
		},
		"labels": map[string]any{
			"nodes": []any{
				map[string]any{"name": "Backend"},
				map[string]any{"name": "Urgent"},
			},
		},
		"relations": map[string]any{
			"nodes": []any{
				map[string]any{
					"type": "blocks",
					"issue": map[string]any{
						"id":         "blocker-1",
						"identifier": "MT-099",
						"state":      map[string]any{"name": "Todo"},
					},
				},
			},
		},
	}

	issue, err := tracker.NormalizeIssue(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if issue.Identifier != "MT-101" {
		t.Fatalf("identifier = %q", issue.Identifier)
	}
	if got, want := issue.Labels, []string{"backend", "urgent"}; len(got) != len(want) ||
		got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("labels = %#v want %#v", got, want)
	}
	if issue.Priority == nil || *issue.Priority != 2 {
		t.Fatalf("priority = %#v", issue.Priority)
	}
	if issue.CreatedAt == nil || issue.CreatedAt.Format(time.RFC3339) != "2026-03-07T12:00:00Z" {
		t.Fatalf("created_at = %#v", issue.CreatedAt)
	}
	if len(issue.BlockedBy) != 1 || issue.BlockedBy[0].Identifier == nil || *issue.BlockedBy[0].Identifier != "MT-099" {
		t.Fatalf("blocked_by = %#v", issue.BlockedBy)
	}
}

func TestFetchCandidatesPaginates(t *testing.T) {
	var calls int
	client := tracker.Client{
		Endpoint:    "https://linear.invalid/graphql",
		APIKey:      "test-token",
		ProjectSlug: "proj",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if got := r.Header.Get("Authorization"); got != "test-token" {
					t.Fatalf("authorization header = %q", got)
				}

				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				query, _ := req["query"].(string)
				if !strings.Contains(query, "slugId") {
					t.Fatalf("query missing slugId filter: %s", query)
				}
				vars, _ := req["variables"].(map[string]any)
				if vars["projectSlug"] != "proj" {
					t.Fatalf("projectSlug = %#v", vars["projectSlug"])
				}

				if calls == 1 {
					return jsonResponse(t, http.StatusOK, map[string]any{
						"data": map[string]any{
							"issues": map[string]any{
								"nodes": []any{
									issueNode("issue-1", "MT-1", "Todo"),
								},
								"pageInfo": map[string]any{
									"hasNextPage": true,
									"endCursor":   "cursor-1",
								},
							},
						},
					}), nil
				}

				if vars["after"] != "cursor-1" {
					t.Fatalf("after = %#v", vars["after"])
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"data": map[string]any{
						"issues": map[string]any{
							"nodes": []any{
								issueNode("issue-2", "MT-2", "In Progress"),
							},
							"pageInfo": map[string]any{
								"hasNextPage": false,
								"endCursor":   nil,
							},
						},
					},
				}), nil
			}),
		},
	}

	issues, err := client.FetchCandidates(context.Background(), []string{"Todo", "In Progress"})
	if err != nil {
		t.Fatalf("fetch candidates: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
	if len(issues) != 2 {
		t.Fatalf("issues len = %d", len(issues))
	}
}

func TestFetchStatesByIDsEmptySkipsRequest(t *testing.T) {
	var calls int
	client := tracker.Client{
		Endpoint: "https://linear.invalid/graphql",
		APIKey:   "test-token",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return jsonResponse(t, http.StatusInternalServerError, map[string]any{"error": "unexpected"}), nil
			}),
		},
	}

	issues, err := client.FetchStatesByIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("fetch states by ids: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues len = %d", len(issues))
	}
	if calls != 0 {
		t.Fatalf("calls = %d", calls)
	}
}

func TestFetchCandidatesNon200ReturnsStatusError(t *testing.T) {
	client := tracker.Client{
		Endpoint:    "https://linear.invalid/graphql",
		APIKey:      "test-token",
		ProjectSlug: "proj",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader("bad gateway")),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	_, err := client.FetchCandidates(context.Background(), []string{"Todo"})
	assertErrorKind(t, err, tracker.ErrLinearAPIStatus)
}

func TestFetchCandidatesGraphQLErrors(t *testing.T) {
	client := tracker.Client{
		Endpoint:    "https://linear.invalid/graphql",
		APIKey:      "test-token",
		ProjectSlug: "proj",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(t, http.StatusOK, map[string]any{
					"errors": []any{
						map[string]any{"message": "boom"},
					},
				}), nil
			}),
		},
	}

	_, err := client.FetchCandidates(context.Background(), []string{"Todo"})
	assertErrorKind(t, err, tracker.ErrLinearGraphQLErrors)
}

func issueNode(id, identifier, state string) map[string]any {
	return map[string]any{
		"id":         id,
		"identifier": identifier,
		"title":      identifier,
		"state":      map[string]any{"name": state},
		"labels":     map[string]any{"nodes": []any{}},
		"relations":  map[string]any{"nodes": []any{}},
	}
}

func jsonResponse(t *testing.T, status int, payload map[string]any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode response: %v", err)
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}
}

func assertErrorKind(t *testing.T, err error, kind string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error kind %q, got nil", kind)
	}
	te, ok := err.(*tracker.Error)
	if !ok {
		t.Fatalf("expected *tracker.Error, got %T", err)
	}
	if te.Kind != kind {
		t.Fatalf("kind = %q want %q", te.Kind, kind)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestRawQueryReturnsFullPayloadIncludingGraphQLErrors(t *testing.T) {
	// RawQuery must return the full payload even when errors are present;
	// the caller decides what to do with them.
	client := tracker.Client{
		Endpoint: "https://linear.invalid/graphql",
		APIKey:   "test-key",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(t, http.StatusOK, map[string]any{
					"errors": []any{map[string]any{"message": "not found"}},
					"data":   nil,
				}), nil
			}),
		},
	}

	payload, err := client.RawQuery(context.Background(), "query { viewer { id } }", nil)
	if err != nil {
		t.Fatalf("RawQuery returned unexpected error: %v", err)
	}
	if _, ok := payload["errors"]; !ok {
		t.Fatalf("RawQuery dropped errors field from payload: %#v", payload)
	}
}

func TestRawQueryReturnsPayloadOnSuccess(t *testing.T) {
	client := tracker.Client{
		Endpoint: "https://linear.invalid/graphql",
		APIKey:   "test-key",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if got := r.Header.Get("Authorization"); got != "test-key" {
					t.Errorf("authorization header = %q", got)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"data": map[string]any{"viewer": map[string]any{"id": "u-1"}},
				}), nil
			}),
		},
	}

	payload, err := client.RawQuery(context.Background(), "query { viewer { id } }", nil)
	if err != nil {
		t.Fatalf("RawQuery error: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data in payload: %#v", payload)
	}
}

func TestRawQueryFailsOnNon200Status(t *testing.T) {
	client := tracker.Client{
		Endpoint: "https://linear.invalid/graphql",
		APIKey:   "test-key",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(t, http.StatusUnauthorized, map[string]any{"error": "unauthorized"}), nil
			}),
		},
	}

	_, err := client.RawQuery(context.Background(), "query { viewer { id } }", nil)
	assertErrorKind(t, err, tracker.ErrLinearAPIStatus)
}
