package orchestrator

import (
	"slices"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/agent"
)

type Snapshot struct {
	GeneratedAt time.Time       `json:"generated_at"`
	Counts      SnapshotCounts  `json:"counts"`
	Running     []RunningStatus `json:"running"`
	Retrying    []RetryStatus   `json:"retrying"`
	CodexTotals SnapshotTotals  `json:"codex_totals"`
	RateLimits  map[string]any  `json:"rate_limits"`
}

type SnapshotCounts struct {
	Running  int `json:"running"`
	Retrying int `json:"retrying"`
}

type SnapshotTotals struct {
	InputTokens    int     `json:"input_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	TotalTokens    int     `json:"total_tokens"`
	SecondsRunning float64 `json:"seconds_running"`
}

type RunningStatus struct {
	IssueID         string      `json:"issue_id"`
	IssueIdentifier string      `json:"issue_identifier"`
	State           string      `json:"state"`
	SessionID       string      `json:"session_id"`
	TurnCount       int         `json:"turn_count"`
	LastEvent       string      `json:"last_event"`
	LastMessage     string      `json:"last_message"`
	StartedAt       time.Time   `json:"started_at"`
	LastEventAt     *time.Time  `json:"last_event_at"`
	Tokens          agent.Usage `json:"tokens"`
}

type RetryStatus struct {
	IssueID         string    `json:"issue_id"`
	IssueIdentifier string    `json:"issue_identifier"`
	Attempt         int       `json:"attempt"`
	DueAt           time.Time `json:"due_at"`
	Error           string    `json:"error"`
}

type IssueDetail struct {
	IssueIdentifier string         `json:"issue_identifier"`
	IssueID         string         `json:"issue_id"`
	Status          string         `json:"status"`
	Workspace       WorkspaceInfo  `json:"workspace"`
	Attempts        AttemptInfo    `json:"attempts"`
	Running         *RunningStatus `json:"running"`
	Retry           *RetryStatus   `json:"retry"`
	LastError       *string        `json:"last_error"`
	Tracked         map[string]any `json:"tracked"`
}

type WorkspaceInfo struct {
	Path string `json:"path"`
}

type AttemptInfo struct {
	RestartCount        int `json:"restart_count"`
	CurrentRetryAttempt int `json:"current_retry_attempt"`
}

func (o *Orchestrator) Snapshot() Snapshot {
	o.mu.Lock()
	defer o.mu.Unlock()

	now := o.now()
	snapshot := Snapshot{
		GeneratedAt: now,
		Running:     make([]RunningStatus, 0, len(o.state.Running)),
		Retrying:    make([]RetryStatus, 0, len(o.state.RetryAttempts)),
		RateLimits:  cloneMap(o.state.CodexRateLimits),
		CodexTotals: SnapshotTotals{
			InputTokens:    o.state.CodexTotals.InputTokens,
			OutputTokens:   o.state.CodexTotals.OutputTokens,
			TotalTokens:    o.state.CodexTotals.TotalTokens,
			SecondsRunning: float64(o.state.CodexTotals.SecondsRunning),
		},
	}

	for _, entry := range o.state.Running {
		if entry == nil {
			continue
		}
		snapshot.Running = append(snapshot.Running, runningStatusFromEntry(entry))
		snapshot.CodexTotals.InputTokens += entry.CurrentUsage.InputTokens
		snapshot.CodexTotals.OutputTokens += entry.CurrentUsage.OutputTokens
		snapshot.CodexTotals.TotalTokens += entry.CurrentUsage.TotalTokens
		snapshot.CodexTotals.SecondsRunning += now.Sub(entry.StartedAt).Seconds()
	}
	for _, retry := range o.state.RetryAttempts {
		snapshot.Retrying = append(snapshot.Retrying, RetryStatus{
			IssueID:         retry.IssueID,
			IssueIdentifier: retry.Identifier,
			Attempt:         retry.Attempt,
			DueAt:           retry.DueAt,
			Error:           retry.Error,
		})
	}

	slices.SortFunc(snapshot.Running, func(a, b RunningStatus) int {
		if a.IssueIdentifier != b.IssueIdentifier {
			if a.IssueIdentifier < b.IssueIdentifier {
				return -1
			}
			return 1
		}
		if a.IssueID < b.IssueID {
			return -1
		}
		if a.IssueID > b.IssueID {
			return 1
		}
		return 0
	})
	slices.SortFunc(snapshot.Retrying, func(a, b RetryStatus) int {
		if a.IssueIdentifier != b.IssueIdentifier {
			if a.IssueIdentifier < b.IssueIdentifier {
				return -1
			}
			return 1
		}
		if a.IssueID < b.IssueID {
			return -1
		}
		if a.IssueID > b.IssueID {
			return 1
		}
		return 0
	})

	snapshot.Counts = SnapshotCounts{Running: len(snapshot.Running), Retrying: len(snapshot.Retrying)}
	return snapshot
}

func (o *Orchestrator) Issue(identifier string) (IssueDetail, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, entry := range o.state.Running {
		if entry == nil || entry.Issue.Identifier != identifier {
			continue
		}
		return IssueDetail{
			IssueIdentifier: entry.Issue.Identifier,
			IssueID:         entry.Issue.ID,
			Status:          "running",
			Workspace:       WorkspaceInfo{Path: entry.WorkspacePath},
			Attempts: AttemptInfo{
				RestartCount:        max(entry.RetryAttempt, 0),
				CurrentRetryAttempt: max(entry.RetryAttempt, 0),
			},
			Running: runningStatusPtr(entry),
			Tracked: map[string]any{},
		}, true
	}
	for _, retry := range o.state.RetryAttempts {
		if retry.Identifier != identifier {
			continue
		}
		status := RetryStatus{
			IssueID:         retry.IssueID,
			IssueIdentifier: retry.Identifier,
			Attempt:         retry.Attempt,
			DueAt:           retry.DueAt,
			Error:           retry.Error,
		}
		return IssueDetail{
			IssueIdentifier: retry.Identifier,
			IssueID:         retry.IssueID,
			Status:          "retrying",
			Workspace:       WorkspaceInfo{Path: issueWorkspacePath(o.cfg.WorkspaceRoot(), retry.Identifier)},
			Attempts: AttemptInfo{
				RestartCount:        retry.Attempt,
				CurrentRetryAttempt: retry.Attempt,
			},
			Retry:     &status,
			LastError: stringPtrOrNil(retry.Error),
			Tracked:   map[string]any{},
		}, true
	}
	return IssueDetail{}, false
}

func runningStatusPtr(entry *RunningEntry) *RunningStatus {
	status := runningStatusFromEntry(entry)
	return &status
}

func runningStatusFromEntry(entry *RunningEntry) RunningStatus {
	return RunningStatus{
		IssueID:         entry.Issue.ID,
		IssueIdentifier: entry.Issue.Identifier,
		State:           entry.Issue.State,
		SessionID:       entry.SessionID,
		TurnCount:       entry.TurnCount,
		LastEvent:       entry.LastEvent,
		LastMessage:     entry.LastMessage,
		StartedAt:       entry.StartedAt,
		LastEventAt:     entry.LastEventAt,
		Tokens:          entry.CurrentUsage,
	}
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
