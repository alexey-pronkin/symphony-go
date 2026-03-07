package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/orchestrator"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

type Runtime interface {
	Snapshot() orchestrator.Snapshot
	Issue(string) (orchestrator.IssueDetail, bool)
	Refresh()
}

type TaskPlatform interface {
	ListTasks() ([]tracker.Issue, error)
	CreateTask(tracker.CreateTaskInput) (tracker.Issue, error)
	UpdateTask(string, tracker.UpdateTaskInput) (tracker.Issue, error)
}

func NewHandler(runtime Runtime, tasks TaskPlatform, dashboardDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			if r.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			writeMetrics(runtime, tasks, w)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			handleAPI(runtime, tasks, w, r)
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		if tryServeDashboard(w, r, dashboardDir) {
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!doctype html><html><body><h1>Arpego</h1><p>Observability endpoint.</p></body></html>"))
	})
	return mux
}

func tryServeDashboard(w http.ResponseWriter, r *http.Request, dashboardDir string) bool {
	if dashboardDir == "" {
		return false
	}
	indexPath := filepath.Join(dashboardDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return false
	}

	cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if cleanPath == "." {
		http.ServeFile(w, r, indexPath)
		return true
	}

	targetPath := filepath.Join(dashboardDir, cleanPath)
	if fileExists(targetPath) {
		http.ServeFile(w, r, targetPath)
		return true
	}

	if hasExtension(cleanPath) {
		http.NotFound(w, r)
		return true
	}

	http.ServeFile(w, r, indexPath)
	return true
}

func handleAPI(runtime Runtime, tasks TaskPlatform, w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/v1/tasks":
		handleTasks(tasks, w, r)
	case strings.HasPrefix(r.URL.Path, "/api/v1/tasks/"):
		handleTaskByIdentifier(tasks, w, r)
	case r.URL.Path == "/api/v1/state":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, runtime.Snapshot())
	case r.URL.Path == "/api/v1/refresh":
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		runtime.Refresh()
		writeJSON(w, http.StatusAccepted, map[string]any{
			"queued":       true,
			"coalesced":    false,
			"requested_at": time.Now().UTC(),
			"operations":   []string{"poll", "reconcile"},
		})
	case strings.HasPrefix(r.URL.Path, "/api/v1/"):
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		identifier := strings.TrimPrefix(r.URL.Path, "/api/v1/")
		if identifier == "" {
			http.NotFound(w, r)
			return
		}
		detail, ok := runtime.Issue(identifier)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error": map[string]any{
					"code":    "issue_not_found",
					"message": "issue not found in current runtime state",
				},
			})
			return
		}
		writeJSON(w, http.StatusOK, detail)
	default:
		http.NotFound(w, r)
	}
}

func handleTasks(tasks TaskPlatform, w http.ResponseWriter, r *http.Request) {
	if tasks == nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": map[string]any{
				"code":    "task_platform_unavailable",
				"message": "local task platform is unavailable for the active tracker",
			},
		})
		return
	}
	switch r.Method {
	case http.MethodGet:
		issues, err := tasks.ListTasks()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"error": map[string]any{
					"code":    "task_list_failed",
					"message": err.Error(),
				},
			})
			return
		}
		byState := map[string]int{}
		for _, issue := range issues {
			byState[normalizeStateKey(issue.State)]++
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"tasks": issues,
			"counts": map[string]any{
				"total":    len(issues),
				"by_state": byState,
			},
		})
	case http.MethodPost:
		var input tracker.CreateTaskInput
		if err := readJSON(r.Body, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]any{
					"code":    "invalid_request",
					"message": err.Error(),
				},
			})
			return
		}
		issue, err := tasks.CreateTask(input)
		if err != nil {
			writeTaskError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, issue)
	default:
		methodNotAllowed(w, "GET, POST")
	}
}

func handleTaskByIdentifier(tasks TaskPlatform, w http.ResponseWriter, r *http.Request) {
	if tasks == nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": map[string]any{
				"code":    "task_platform_unavailable",
				"message": "local task platform is unavailable for the active tracker",
			},
		})
		return
	}
	if r.Method != http.MethodPatch {
		methodNotAllowed(w, http.MethodPatch)
		return
	}
	identifier := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	if identifier == "" {
		http.NotFound(w, r)
		return
	}
	var input tracker.UpdateTaskInput
	if err := readJSON(r.Body, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]any{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}
	issue, err := tasks.UpdateTask(identifier, input)
	if err != nil {
		writeTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, issue)
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
		"error": map[string]any{
			"code":    "method_not_allowed",
			"message": "method not allowed",
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func readJSON(body io.ReadCloser, out any) error {
	defer func() {
		_ = body.Close()
	}()
	return json.NewDecoder(body).Decode(out)
}

func writeTaskError(w http.ResponseWriter, err error) {
	var taskErr *tracker.TaskError
	if errors.As(err, &taskErr) {
		status := http.StatusBadRequest
		if taskErr.Code == tracker.ErrTaskNotFound {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{
			"error": map[string]any{
				"code":    taskErr.Code,
				"message": taskErr.Message,
			},
		})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]any{
		"error": map[string]any{
			"code":    "task_platform_error",
			"message": err.Error(),
		},
	})
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func hasExtension(path string) bool {
	base := filepath.Base(path)
	return filepath.Ext(base) != ""
}

func normalizeStateKey(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}

func writeMetrics(runtime Runtime, tasks TaskPlatform, w http.ResponseWriter) {
	snapshot := runtime.Snapshot()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = io.WriteString(w, "# HELP symphony_running_sessions Current number of running sessions.\n")
	_, _ = io.WriteString(w, "# TYPE symphony_running_sessions gauge\n")
	_, _ = io.WriteString(w, fmt.Sprintf("symphony_running_sessions %d\n", snapshot.Counts.Running))
	_, _ = io.WriteString(w, "# HELP symphony_retrying_sessions Current number of retrying sessions.\n")
	_, _ = io.WriteString(w, "# TYPE symphony_retrying_sessions gauge\n")
	_, _ = io.WriteString(w, fmt.Sprintf("symphony_retrying_sessions %d\n", snapshot.Counts.Retrying))
	_, _ = io.WriteString(w, "# HELP symphony_total_tokens Aggregate Codex tokens.\n")
	_, _ = io.WriteString(w, "# TYPE symphony_total_tokens gauge\n")
	_, _ = io.WriteString(w, fmt.Sprintf("symphony_total_tokens %d\n", snapshot.CodexTotals.TotalTokens))
	_, _ = io.WriteString(w, "# HELP symphony_runtime_seconds Aggregate running seconds.\n")
	_, _ = io.WriteString(w, "# TYPE symphony_runtime_seconds gauge\n")
	_, _ = io.WriteString(w, fmt.Sprintf("symphony_runtime_seconds %.0f\n", snapshot.CodexTotals.SecondsRunning))
	if tasks == nil {
		return
	}
	issues, err := tasks.ListTasks()
	if err != nil {
		return
	}
	_, _ = io.WriteString(w, "# HELP symphony_tasks_total Current number of tasks.\n")
	_, _ = io.WriteString(w, "# TYPE symphony_tasks_total gauge\n")
	_, _ = io.WriteString(w, fmt.Sprintf("symphony_tasks_total %d\n", len(issues)))
	counts := map[string]int{}
	for _, issue := range issues {
		counts[normalizeStateKey(issue.State)]++
	}
	_, _ = io.WriteString(w, "# HELP symphony_tasks_by_state Current number of tasks grouped by state.\n")
	_, _ = io.WriteString(w, "# TYPE symphony_tasks_by_state gauge\n")
	for state, count := range counts {
		_, _ = io.WriteString(w, fmt.Sprintf("symphony_tasks_by_state{state=%q} %d\n", state, count))
	}
}
