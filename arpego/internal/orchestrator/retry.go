package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/logging"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

func retryDelay(attempt int, continuation bool, maxBackoff time.Duration) time.Duration {
	if continuation && attempt <= 1 {
		return time.Second
	}
	if attempt <= 0 {
		attempt = 1
	}
	delay := 10 * time.Second
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= maxBackoff {
			return maxBackoff
		}
	}
	if maxBackoff > 0 && delay > maxBackoff {
		return maxBackoff
	}
	return delay
}

func (o *Orchestrator) scheduleRetry(issue tracker.Issue, attempt int, continuation bool, reason string) {
	if attempt <= 0 {
		attempt = 1
	}
	if o.state.RetryAttempts == nil {
		o.state.RetryAttempts = map[string]RetryEntry{}
	}
	delay := retryDelay(attempt, continuation, time.Duration(o.cfg.MaxRetryBackoffMs())*time.Millisecond)
	retry := RetryEntry{
		IssueID:    issue.ID,
		Identifier: issue.Identifier,
		Attempt:    attempt,
		DueAt:      o.now().Add(delay),
		Error:      reason,
	}
	if current, ok := o.state.RetryAttempts[issue.ID]; ok && current.Timer != nil {
		current.Timer.Stop()
	}
	retry.Timer = o.afterFunc(delay, func() {
		select {
		case o.retryCh <- issue.ID:
		default:
			go func() { o.retryCh <- issue.ID }()
		}
	})
	o.state.RetryAttempts[issue.ID] = retry
	logging.WithIssue(o.logger, issue.ID, issue.Identifier).Warn(
		"retry outcome=queued",
		"attempt", attempt,
		"delay_ms", delay.Milliseconds(),
		"reason", reason,
	)
}

func (o *Orchestrator) handleRetry(ctx context.Context, issueID string) {
	o.mu.Lock()
	retry, ok := o.state.RetryAttempts[issueID]
	if ok {
		delete(o.state.RetryAttempts, issueID)
	}
	o.mu.Unlock()
	if !ok {
		return
	}

	issues, err := o.tracker.FetchCandidates(ctx, o.cfg.TrackerActiveStates())
	if err != nil {
		o.mu.Lock()
		o.scheduleRetry(
			tracker.Issue{ID: issueID, Identifier: retry.Identifier},
			retry.Attempt+1,
			false,
			fmt.Sprintf("retry poll failed: %v", err),
		)
		o.mu.Unlock()
		return
	}

	var issue *tracker.Issue
	for i := range issues {
		if issues[i].ID == issueID {
			issue = &issues[i]
			break
		}
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if issue == nil {
		delete(o.state.Claimed, issueID)
		return
	}
	if !candidateIssue(*issue, o.cfg) {
		delete(o.state.Claimed, issueID)
		return
	}
	if !stateSlotsAvailable(issue.State, o.state.Running, o.cfg) || len(o.state.Running) >= o.cfg.MaxConcurrentAgents() {
		o.scheduleRetry(*issue, retry.Attempt+1, false, "no available orchestrator slots")
		return
	}
	o.dispatchIssue(ctx, *issue, retry.Attempt)
}
