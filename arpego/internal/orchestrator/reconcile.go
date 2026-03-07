package orchestrator

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/logging"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/workspace"
)

func (o *Orchestrator) runStartupCleanup(ctx context.Context) {
	issues, err := o.tracker.FetchByStates(ctx, o.cfg.TrackerTerminalStates())
	if err != nil {
		o.logger.Warn("startup_cleanup outcome=skipped", "reason", err)
		return
	}
	for _, issue := range issues {
		if issue.Identifier == "" {
			continue
		}
		o.cleanupWorkspace(issue.Identifier)
	}
}

func (o *Orchestrator) reconcileRunning(ctx context.Context) {
	o.mu.Lock()
	now := o.now()
	var refreshIDs []string
	for issueID, entry := range o.state.Running {
		if o.stalled(now, entry) {
			delete(o.state.Running, issueID)
			o.finishRuntime(entry, now)
			if entry.cancel != nil {
				entry.cancel()
			}
			o.scheduleRetry(entry.Issue, nextRetryAttempt(entry.RetryAttempt), false, "stall detected")
			continue
		}
		refreshIDs = append(refreshIDs, issueID)
	}
	o.mu.Unlock()

	if len(refreshIDs) == 0 {
		return
	}

	refreshed, err := o.tracker.FetchStatesByIDs(ctx, refreshIDs)
	if err != nil {
		o.logger.Warn("reconcile outcome=refresh_failed", "reason", err)
		return
	}

	byID := make(map[string]tracker.Issue, len(refreshed))
	for _, issue := range refreshed {
		byID[issue.ID] = issue
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	activeStates := stateSet(o.cfg.TrackerActiveStates())
	terminalStates := stateSet(o.cfg.TrackerTerminalStates())
	for issueID, issue := range byID {
		entry, ok := o.state.Running[issueID]
		if !ok {
			continue
		}
		stateName := normalizeState(issue.State)
		switch {
		case terminalStates[stateName]:
			delete(o.state.Running, issueID)
			o.finishRuntime(entry, o.now())
			if entry.cancel != nil {
				entry.cancel()
			}
			delete(o.state.Claimed, issueID)
			o.cleanupWorkspace(issue.Identifier)
			logging.WithIssue(o.logger, issue.ID, issue.Identifier).Info("reconcile outcome=terminal")
		case activeStates[stateName]:
			entry.Issue = issue
		default:
			delete(o.state.Running, issueID)
			o.finishRuntime(entry, o.now())
			if entry.cancel != nil {
				entry.cancel()
			}
			delete(o.state.Claimed, issueID)
			logging.WithIssue(o.logger, issue.ID, issue.Identifier).Info("reconcile outcome=stopped", "state", issue.State)
		}
	}
}

func (o *Orchestrator) stalled(now time.Time, entry *RunningEntry) bool {
	stallTimeout := time.Duration(o.cfg.CodexStallTimeoutMs()) * time.Millisecond
	if stallTimeout <= 0 {
		return false
	}
	last := entry.StartedAt
	if entry.LastEventAt != nil {
		last = *entry.LastEventAt
	}
	return now.Sub(last) > stallTimeout
}

func (o *Orchestrator) cleanupWorkspace(identifier string) {
	root := o.cfg.WorkspaceRoot()
	path := filepath.Join(root, workspace.SanitizeKey(identifier))
	if err := workspace.ValidatePath(root, path); err != nil {
		o.logger.Warn("workspace_cleanup outcome=skipped", "identifier", identifier, "reason", err)
		return
	}
	if hook := o.cfg.HookBeforeRemove(); hook != "" {
		if err := workspace.RunHook(hook, path, int(o.cfg.HookTimeoutMs())); err != nil {
			o.logger.Warn("workspace_cleanup outcome=hook_failed", "identifier", identifier, "reason", err)
		}
	}
	if err := os.RemoveAll(path); err != nil {
		o.logger.Warn("workspace_cleanup outcome=failed", "identifier", identifier, "reason", err)
	}
}

func (o *Orchestrator) finishRuntime(entry *RunningEntry, now time.Time) {
	o.state.CodexTotals.InputTokens += entry.CurrentUsage.InputTokens
	o.state.CodexTotals.OutputTokens += entry.CurrentUsage.OutputTokens
	o.state.CodexTotals.TotalTokens += entry.CurrentUsage.TotalTokens
	o.state.CodexTotals.SecondsRunning += int64(maxDuration(0, now.Sub(entry.StartedAt)).Seconds())
}

func nextRetryAttempt(current int) int {
	if current <= 0 {
		return 1
	}
	return current + 1
}

func maxDuration(a, b time.Duration) time.Duration {
	if b > a {
		return b
	}
	return a
}

func issueWorkspacePath(root, identifier string) string {
	return filepath.Join(root, workspace.SanitizeKey(identifier))
}

func bestEffortAfterRun(cfg config.Config, logger *slog.Logger, workspacePath string) {
	if hook := cfg.HookAfterRun(); hook != "" {
		if err := workspace.RunHook(hook, workspacePath, int(cfg.HookTimeoutMs())); err != nil {
			logger.Warn("hook outcome=ignored_failure", "hook", "after_run", "reason", err)
		}
	}
}
