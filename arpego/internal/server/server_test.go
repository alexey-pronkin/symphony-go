package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/orchestrator"
)

func TestStateEndpointReturnsSummary(t *testing.T) {
	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	runtime := &fakeRuntime{
		snapshot: orchestrator.Snapshot{
			GeneratedAt: now,
			Counts:      orchestrator.SnapshotCounts{Running: 1, Retrying: 1},
			Running: []orchestrator.RunningStatus{{
				IssueID:         "issue-1",
				IssueIdentifier: "MT-649",
				State:           "In Progress",
				SessionID:       "thread-1-turn-1",
				TurnCount:       1,
			}},
			Retrying: []orchestrator.RetryStatus{{
				IssueID:         "issue-2",
				IssueIdentifier: "MT-650",
				Attempt:         2,
			}},
			CodexTotals: orchestrator.SnapshotTotals{TotalTokens: 23},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
	rec := httptest.NewRecorder()
	NewHandler(runtime).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	counts := payload["counts"].(map[string]any)
	if counts["running"] != float64(1) || counts["retrying"] != float64(1) {
		t.Fatalf("counts = %#v", counts)
	}
	running := payload["running"].([]any)
	if len(running) != 1 {
		t.Fatalf("running len = %d", len(running))
	}
}

func TestIssueEndpointReturnsDetailOr404(t *testing.T) {
	runtime := &fakeRuntime{
		issues: map[string]orchestrator.IssueDetail{
			"MT-649": {
				IssueIdentifier: "MT-649",
				IssueID:         "issue-1",
				Status:          "running",
				Workspace:       orchestrator.WorkspaceInfo{Path: "/tmp/MT-649"},
				Running: &orchestrator.RunningStatus{
					SessionID: "thread-1-turn-1",
					Tokens:    orchestrator.RunningStatus{}.Tokens,
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/MT-649", nil)
	rec := httptest.NewRecorder()
	NewHandler(runtime).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("known status = %d want 200", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/MT-999", nil)
	rec = httptest.NewRecorder()
	NewHandler(runtime).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown status = %d want 404", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	errBody := payload["error"].(map[string]any)
	if errBody["code"] != "issue_not_found" {
		t.Fatalf("error body = %#v", errBody)
	}
}

func TestRefreshEndpointQueuesPollAndMethodNotAllowed(t *testing.T) {
	runtime := &fakeRuntime{}
	handler := NewHandler(runtime)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("refresh status = %d want 202", rec.Code)
	}
	if runtime.refreshCalls != 1 {
		t.Fatalf("refresh calls = %d want 1", runtime.refreshCalls)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/state", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("state wrong method status = %d want 405", rec.Code)
	}
}

type fakeRuntime struct {
	snapshot     orchestrator.Snapshot
	issues       map[string]orchestrator.IssueDetail
	refreshCalls int
}

func (f *fakeRuntime) Snapshot() orchestrator.Snapshot {
	return f.snapshot
}

func (f *fakeRuntime) Issue(identifier string) (orchestrator.IssueDetail, bool) {
	detail, ok := f.issues[identifier]
	return detail, ok
}

func (f *fakeRuntime) Refresh() {
	f.refreshCalls++
}
