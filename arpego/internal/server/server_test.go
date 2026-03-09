package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/insights"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/orchestrator"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
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
	NewHandler(runtime, nil, nil, nil, "").ServeHTTP(rec, req)

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
	NewHandler(runtime, nil, nil, nil, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("known status = %d want 200", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/MT-999", nil)
	rec = httptest.NewRecorder()
	NewHandler(runtime, nil, nil, nil, "").ServeHTTP(rec, req)
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
	handler := NewHandler(runtime, nil, nil, nil, "")

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

func TestDashboardRootServesBuiltIndexWhenAvailable(t *testing.T) {
	dashboardDir := writeDashboardFiles(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	NewHandler(&fakeRuntime{}, nil, nil, nil, dashboardDir).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Symphony Dashboard") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestDashboardServesStaticAssetsAndSpaFallback(t *testing.T) {
	dashboardDir := writeDashboardFiles(t)
	handler := NewHandler(&fakeRuntime{}, nil, nil, nil, dashboardDir)

	assetReq := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	assetRec := httptest.NewRecorder()
	handler.ServeHTTP(assetRec, assetReq)
	if assetRec.Code != http.StatusOK {
		t.Fatalf("asset status = %d want 200", assetRec.Code)
	}
	if !strings.Contains(assetRec.Body.String(), "console.log") {
		t.Fatalf("asset body = %q", assetRec.Body.String())
	}

	routeReq := httptest.NewRequest(http.MethodGet, "/issues/MT-1", nil)
	routeRec := httptest.NewRecorder()
	handler.ServeHTTP(routeRec, routeReq)
	if routeRec.Code != http.StatusOK {
		t.Fatalf("route status = %d want 200", routeRec.Code)
	}
	if !strings.Contains(routeRec.Body.String(), "Symphony Dashboard") {
		t.Fatalf("route body = %q", routeRec.Body.String())
	}
}

func TestDashboardMissingAssetReturns404AndFallbackPlaceholder(t *testing.T) {
	dashboardDir := writeDashboardFiles(t)
	handler := NewHandler(&fakeRuntime{}, nil, nil, nil, dashboardDir)

	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing asset status = %d want 404", rec.Code)
	}

	fallbackReq := httptest.NewRequest(http.MethodGet, "/", nil)
	fallbackRec := httptest.NewRecorder()
	NewHandler(&fakeRuntime{}, nil, nil, nil, "").ServeHTTP(fallbackRec, fallbackReq)
	if fallbackRec.Code != http.StatusOK {
		t.Fatalf("fallback status = %d want 200", fallbackRec.Code)
	}
	if !strings.Contains(fallbackRec.Body.String(), "Arpego") {
		t.Fatalf("fallback body = %q", fallbackRec.Body.String())
	}
}

func TestTaskPlatformEndpointsListCreateUpdateAndUnavailable(t *testing.T) {
	platform := &fakeTaskPlatform{
		listTasks: []tracker.Issue{
			{ID: "task-1", Identifier: "SYM-1", Title: "Local task", State: "Todo"},
		},
		createdTask: tracker.Issue{ID: "task-2", Identifier: "SYM-2", Title: "Created", State: "Todo"},
		updatedTask: tracker.Issue{ID: "task-1", Identifier: "SYM-1", Title: "Local task", State: "Done"},
	}
	handler := NewHandler(&fakeRuntime{}, platform, nil, nil, "")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d want 200", listRec.Code)
	}
	var listPayload map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("Unmarshal list: %v", err)
	}
	tasks := listPayload["tasks"].([]any)
	if len(tasks) != 1 {
		t.Fatalf("tasks len = %d want 1", len(tasks))
	}

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks",
		strings.NewReader(`{"title":"Created","state":"Todo"}`),
	)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d want 201", createRec.Code)
	}
	if platform.lastCreate.Title != "Created" {
		t.Fatalf("last create = %#v", platform.lastCreate)
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/SYM-1", strings.NewReader(`{"state":"Done"}`))
	updateRec := httptest.NewRecorder()
	handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d want 200", updateRec.Code)
	}
	if platform.lastUpdateIdentifier != "SYM-1" {
		t.Fatalf("last update identifier = %q", platform.lastUpdateIdentifier)
	}

	unavailable := NewHandler(&fakeRuntime{}, nil, nil, nil, "")
	unavailableReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	unavailableRec := httptest.NewRecorder()
	unavailable.ServeHTTP(unavailableRec, unavailableReq)
	if unavailableRec.Code != http.StatusConflict {
		t.Fatalf("unavailable status = %d want 409", unavailableRec.Code)
	}
}

func TestMetricsEndpointExportsRuntimeAndTaskCounts(t *testing.T) {
	platform := &fakeTaskPlatform{
		listTasks: []tracker.Issue{
			{ID: "task-1", Identifier: "SYM-1", Title: "One", State: "Todo"},
			{ID: "task-2", Identifier: "SYM-2", Title: "Two", State: "Done"},
		},
	}
	runtime := &fakeRuntime{
		snapshot: orchestrator.Snapshot{
			Counts: orchestrator.SnapshotCounts{Running: 2, Retrying: 1},
			CodexTotals: orchestrator.SnapshotTotals{
				TotalTokens:    42,
				SecondsRunning: 12,
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	NewHandler(runtime, platform, nil, nil, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "symphony_running_sessions 2") {
		t.Fatalf("metrics body missing running count: %q", body)
	}
	if !strings.Contains(body, "symphony_tasks_total 2") {
		t.Fatalf("metrics body missing task count: %q", body)
	}
}

func TestDeliveryInsightsEndpointReturnsReport(t *testing.T) {
	handler := NewHandler(&fakeRuntime{}, nil, fakeDeliveryInsights{
		report: insights.DeliveryReport{
			Summary: insights.DeliverySummary{
				DeliveryHealth: insights.IntegralMetric{Score: 81, Status: "strong"},
			},
			Warnings: []string{"scm metrics degraded"},
		},
	}, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/insights/delivery", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	summary := payload["summary"].(map[string]any)
	health := summary["delivery_health"].(map[string]any)
	if health["score"] != float64(81) {
		t.Fatalf("delivery health = %#v", health)
	}
}

func TestDeliveryTrendEndpointReturnsTrendReport(t *testing.T) {
	handler := NewHandler(&fakeRuntime{}, nil, fakeDeliveryInsights{
		trends: insights.DeliveryTrendReport{
			Window:    "7d",
			Limit:     12,
			Available: true,
			Points: []insights.DeliveryTrendPoint{{
				CapturedAt:     time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC),
				DeliveryHealth: 78,
			}},
		},
	}, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/insights/delivery/trends?window=7d&limit=12", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if payload["window"] != "7d" {
		t.Fatalf("window = %#v want 7d", payload["window"])
	}
	points := payload["points"].([]any)
	if len(points) != 1 {
		t.Fatalf("points len = %d want 1", len(points))
	}
}

func TestDeliveryTrendEndpointRejectsInvalidWindow(t *testing.T) {
	handler := NewHandler(&fakeRuntime{}, nil, fakeDeliveryInsights{
		trendErr: insights.ErrInvalidTrendWindow,
	}, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/insights/delivery/trends?window=365d", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400", rec.Code)
	}
}

func TestIssueEndpointEnrichesDetailWithPersistedRuntimeEvents(t *testing.T) {
	runtime := &fakeRuntime{
		issues: map[string]orchestrator.IssueDetail{
			"MT-649": {
				IssueIdentifier: "MT-649",
				IssueID:         "issue-1",
				Status:          "running",
				Workspace:       orchestrator.WorkspaceInfo{Path: "/tmp/MT-649"},
				Logs:            orchestrator.IssueLogs{},
				RecentEvents: []orchestrator.IssueEvent{{
					At:      time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC),
					Event:   "turn.started",
					Message: "turn started",
				}},
			},
		},
	}
	observability := fakeObservability{
		events: []tracker.RuntimeEvent{
			{
				IssueID:    "issue-1",
				Identifier: "MT-649",
				Name:       "session.started",
				Message:    "session started",
				ObservedAt: time.Date(2026, 3, 7, 11, 59, 0, 0, time.UTC),
				LogPath:    "/tmp/MT-649/.symphony/session.jsonl",
				SessionID:  "thread-1-turn-1",
			},
			{
				IssueID:    "issue-1",
				Identifier: "MT-649",
				Name:       "turn.completed",
				Message:    "turn completed",
				ObservedAt: time.Date(2026, 3, 7, 12, 1, 0, 0, time.UTC),
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/MT-649", nil)
	rec := httptest.NewRecorder()
	NewHandler(runtime, nil, nil, observability, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	recentEvents := payload["recent_events"].([]any)
	if len(recentEvents) != 3 {
		t.Fatalf("recent events len = %d want 3", len(recentEvents))
	}
	logs := payload["logs"].(map[string]any)["codex_session_logs"].([]any)
	if len(logs) != 1 {
		t.Fatalf("log refs len = %d want 1", len(logs))
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

type fakeTaskPlatform struct {
	listTasks            []tracker.Issue
	createdTask          tracker.Issue
	updatedTask          tracker.Issue
	lastCreate           tracker.CreateTaskInput
	lastUpdate           tracker.UpdateTaskInput
	lastUpdateIdentifier string
}

type fakeDeliveryInsights struct {
	report   insights.DeliveryReport
	err      error
	trends   insights.DeliveryTrendReport
	trendErr error
}

type fakeObservability struct {
	events []tracker.RuntimeEvent
	err    error
}

func (f fakeDeliveryInsights) Delivery(context.Context) (insights.DeliveryReport, error) {
	return f.report, f.err
}

func (f fakeDeliveryInsights) Trends(
	context.Context,
	insights.DeliveryTrendQuery,
) (insights.DeliveryTrendReport, error) {
	return f.trends, f.trendErr
}

func (f fakeObservability) ListRuntimeEvents(
	context.Context,
	tracker.RuntimeEventQuery,
) ([]tracker.RuntimeEvent, error) {
	return f.events, f.err
}

func (f *fakeTaskPlatform) ListTasks(context.Context) ([]tracker.Issue, error) {
	return f.listTasks, nil
}

func (f *fakeTaskPlatform) CreateTask(_ context.Context, input tracker.CreateTaskInput) (tracker.Issue, error) {
	f.lastCreate = input
	return f.createdTask, nil
}

func (f *fakeTaskPlatform) UpdateTask(
	_ context.Context,
	identifier string,
	input tracker.UpdateTaskInput,
) (tracker.Issue, error) {
	f.lastUpdateIdentifier = identifier
	f.lastUpdate = input
	return f.updatedTask, nil
}

func writeDashboardFiles(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	assetsDir := filepath.Join(dir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "index.html"),
		[]byte("<!doctype html><html><body>Symphony Dashboard</body></html>"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(assetsDir, "app.js"),
		[]byte("console.log('dashboard')"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile asset: %v", err)
	}
	return dir
}
