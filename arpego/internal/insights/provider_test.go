package insights

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultInspectorGitHubProviderMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/org/repo/pulls":
			writeJSON(t, w, []map[string]any{
				{
					"number":     12,
					"draft":      false,
					"updated_at": "2026-03-04T12:00:00Z",
					"head":       map[string]any{"sha": "abc123"},
				},
			})
		case "/repos/org/repo/pulls/12/reviews":
			writeJSON(t, w, []map[string]any{{"state": "APPROVED"}})
		case "/repos/org/repo/commits/abc123/check-runs":
			writeJSON(t, w, map[string]any{
				"check_runs": []map[string]any{{"conclusion": "failure"}},
			})
		default:
			t.Fatalf("unexpected github path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	metrics, err := DefaultInspector{Client: server.Client()}.Inspect(
		context.Background(),
		SourceConfig{
			Kind:       "github",
			Name:       "origin",
			Repository: "org/repo",
			APIURL:     server.URL,
		},
		72*time.Hour,
		time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if metrics.OpenChangeRequests != 1 {
		t.Fatalf("open change requests = %d want 1", metrics.OpenChangeRequests)
	}
	if metrics.ApprovedChangeRequests != 1 {
		t.Fatalf("approved change requests = %d want 1", metrics.ApprovedChangeRequests)
	}
	if metrics.FailingChangeRequests != 1 {
		t.Fatalf("failing change requests = %d want 1", metrics.FailingChangeRequests)
	}
	if metrics.StaleChangeRequests != 1 {
		t.Fatalf("stale change requests = %d want 1", metrics.StaleChangeRequests)
	}
}

func TestDefaultInspectorGitLabProviderMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/group/project/merge_requests":
			writeJSON(t, w, []map[string]any{
				{
					"iid":        8,
					"updated_at": "2026-03-08T06:00:00Z",
					"draft":      false,
					"head_pipeline": map[string]any{
						"status": "failed",
					},
				},
			})
		case "/api/v4/projects/group/project/merge_requests/8/approvals":
			writeJSON(t, w, map[string]any{"approved": true})
		default:
			t.Fatalf("unexpected gitlab path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	metrics, err := DefaultInspector{Client: server.Client()}.Inspect(
		context.Background(),
		SourceConfig{
			Kind:      "gitlab",
			Name:      "internal",
			ProjectID: "group%2Fproject",
			APIURL:    server.URL + "/api/v4",
		},
		24*time.Hour,
		time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if metrics.OpenChangeRequests != 1 {
		t.Fatalf("open change requests = %d want 1", metrics.OpenChangeRequests)
	}
	if metrics.ApprovedChangeRequests != 1 {
		t.Fatalf("approved change requests = %d want 1", metrics.ApprovedChangeRequests)
	}
	if metrics.FailingChangeRequests != 1 {
		t.Fatalf("failing change requests = %d want 1", metrics.FailingChangeRequests)
	}
	if metrics.StaleChangeRequests != 0 {
		t.Fatalf("stale change requests = %d want 0", metrics.StaleChangeRequests)
	}
}

func TestDefaultInspectorGitVerseReturnsGracefulWarning(t *testing.T) {
	metrics, err := DefaultInspector{}.Inspect(
		context.Background(),
		SourceConfig{
			Kind: "gitverse",
			Name: "gitverse",
		},
		24*time.Hour,
		time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatalf("expected degraded gitverse error, metrics = %#v", metrics)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("Encode: %v", err)
	}
}
