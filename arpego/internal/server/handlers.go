package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/orchestrator"
)

type Runtime interface {
	Snapshot() orchestrator.Snapshot
	Issue(string) (orchestrator.IssueDetail, bool)
	Refresh()
}

func NewHandler(runtime Runtime) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			handleAPI(runtime, w, r)
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!doctype html><html><body><h1>Arpego</h1><p>Observability endpoint.</p></body></html>"))
	})
	return mux
}

func handleAPI(runtime Runtime, w http.ResponseWriter, r *http.Request) {
	switch {
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
