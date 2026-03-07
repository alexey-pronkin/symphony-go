package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/agent"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
	ilog "github.com/alexey-pronkin/symphony-go/arpego/internal/logging"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/workflow"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/workspace"
)

type Tracker interface {
	FetchCandidates(context.Context, []string) ([]tracker.Issue, error)
	FetchByStates(context.Context, []string) ([]tracker.Issue, error)
	FetchStatesByIDs(context.Context, []string) ([]tracker.Issue, error)
}

type Runner interface {
	Run(context.Context, agent.RunParams) (agent.RunnerResult, error)
}

type Options struct {
	Config    config.Config
	Workflow  *workflow.Definition
	Logger    *slog.Logger
	Tracker   Tracker
	Runner    Runner
	Now       func() time.Time
	AfterFunc func(time.Duration, func()) timerHandle
}

type Orchestrator struct {
	mu        sync.Mutex
	cfg       config.Config
	workflow  *workflow.Definition
	logger    *slog.Logger
	tracker   Tracker
	runner    Runner
	now       func() time.Time
	afterFunc func(time.Duration, func()) timerHandle
	state     State

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	refreshCh chan struct{}
	retryCh   chan string
	resultsCh chan workerResult
	ticker    *time.Ticker
}

type workerResult struct {
	IssueID string
	Result  agent.RunnerResult
	Err     error
}

func New(opts Options) *Orchestrator {
	logger := opts.Logger
	if logger == nil {
		logger = ilog.Default("")
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	afterFunc := opts.AfterFunc
	if afterFunc == nil {
		afterFunc = func(delay time.Duration, fn func()) timerHandle {
			return time.AfterFunc(delay, fn)
		}
	}

	orc := &Orchestrator{
		cfg:       opts.Config,
		workflow:  opts.Workflow,
		logger:    logger,
		tracker:   opts.Tracker,
		runner:    opts.Runner,
		now:       now,
		afterFunc: afterFunc,
		state:     newState(),
		refreshCh: make(chan struct{}, 1),
		retryCh:   make(chan string, 32),
		resultsCh: make(chan workerResult, 32),
	}
	if orc.tracker == nil {
		orc.tracker = trackerAdapter{cfg: func() config.Config { return orc.cfg }}
	}
	if orc.runner == nil {
		orc.runner = runnerAdapter{cfg: func() config.Config { return orc.cfg }}
	}
	orc.applyConfigLocked(opts.Config)
	return orc
}

func (o *Orchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	if o.cancel != nil {
		o.mu.Unlock()
		return nil
	}
	o.ctx, o.cancel = context.WithCancel(ctx)
	o.ticker = time.NewTicker(time.Duration(o.state.PollIntervalMs) * time.Millisecond)
	o.mu.Unlock()

	o.runStartupCleanup(o.ctx)
	o.wg.Add(1)
	go o.loop()
	o.Refresh()
	return nil
}

func (o *Orchestrator) Stop() {
	o.mu.Lock()
	cancel := o.cancel
	ticker := o.ticker
	o.cancel = nil
	o.ticker = nil
	o.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if ticker != nil {
		ticker.Stop()
	}
	o.wg.Wait()
}

func (o *Orchestrator) Refresh() {
	select {
	case o.refreshCh <- struct{}{}:
	default:
	}
}

func (o *Orchestrator) ApplyWorkflow(def *workflow.Definition) {
	if def == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.workflow = def
	o.cfg = config.New(def.Config)
	o.applyConfigLocked(o.cfg)
	if o.ticker != nil {
		o.ticker.Reset(time.Duration(o.state.PollIntervalMs) * time.Millisecond)
	}
}

func (o *Orchestrator) State() State {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.state
}

func (o *Orchestrator) loop() {
	defer o.wg.Done()
	for {
		o.mu.Lock()
		ctx := o.ctx
		var tickerC <-chan time.Time
		if o.ticker != nil {
			tickerC = o.ticker.C
		}
		o.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		case <-o.refreshCh:
			o.tick(ctx)
		case <-tickerC:
			o.tick(ctx)
		case issueID := <-o.retryCh:
			o.handleRetry(ctx, issueID)
		case result := <-o.resultsCh:
			o.handleWorkerResult(result)
		}
	}
}

func (o *Orchestrator) tick(ctx context.Context) {
	o.reconcileRunning(ctx)
	if err := config.ValidateDispatch(o.cfg); err != nil {
		o.logger.Error("dispatch outcome=validation_failed", "reason", err)
		return
	}
	issues, err := o.tracker.FetchCandidates(ctx, o.cfg.TrackerActiveStates())
	if err != nil {
		o.logger.Warn("dispatch outcome=candidate_fetch_failed", "reason", err)
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, issue := range sortIssuesForDispatch(issues) {
		if !dispatchEligible(issue, o.state, o.cfg) {
			continue
		}
		o.dispatchIssue(ctx, issue, 0)
	}
}

func (o *Orchestrator) dispatchIssue(ctx context.Context, issue tracker.Issue, attempt int) {
	ws, err := workspace.EnsureWorkspace(o.cfg.WorkspaceRoot(), issue.Identifier)
	if err != nil {
		o.scheduleRetry(issue, nextRetryAttempt(attempt), false, fmt.Sprintf("workspace error: %v", err))
		return
	}
	if ws.CreatedNow && o.cfg.HookAfterCreate() != "" {
		if err := workspace.RunHook(o.cfg.HookAfterCreate(), ws.Path, int(o.cfg.HookTimeoutMs())); err != nil {
			o.scheduleRetry(issue, nextRetryAttempt(attempt), false, fmt.Sprintf("after_create hook error: %v", err))
			return
		}
	}
	if o.cfg.HookBeforeRun() != "" {
		if err := workspace.RunHook(o.cfg.HookBeforeRun(), ws.Path, int(o.cfg.HookTimeoutMs())); err != nil {
			o.scheduleRetry(issue, nextRetryAttempt(attempt), false, fmt.Sprintf("before_run hook error: %v", err))
			return
		}
	}

	runCtx, cancel := context.WithCancel(o.ctx)
	entry := &RunningEntry{
		Issue:         issue,
		WorkspacePath: ws.Path,
		StartedAt:     o.now(),
		RetryAttempt:  attempt,
		cancel:        cancel,
	}
	o.state.Running[issue.ID] = entry
	o.state.Claimed[issue.ID] = struct{}{}
	delete(o.state.RetryAttempts, issue.ID)
	ilog.WithIssue(o.logger, issue.ID, issue.Identifier).Info("dispatch outcome=started", "attempt", attempt)

	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		defer bestEffortAfterRun(o.cfg, o.logger, ws.Path)
		result, err := o.runner.Run(runCtx, agent.RunParams{
			Issue:         issue,
			Attempt:       normalizeAttempt(attempt),
			WorkspacePath: ws.Path,
			Workflow:      o.workflow,
			OnSession: func(started agent.SessionStarted) {
				o.recordSession(issue, started)
			},
			OnEvent: func(event agent.Event) {
				o.recordEvent(issue.ID, event)
			},
			RefreshIssue: func(ctx context.Context, issueID string) (tracker.Issue, bool, error) {
				return o.refreshIssueForContinuation(ctx, issueID)
			},
		})
		o.resultsCh <- workerResult{IssueID: issue.ID, Result: result, Err: err}
	}()
}

func (o *Orchestrator) handleWorkerResult(result workerResult) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state.Completed == nil {
		o.state.Completed = map[string]struct{}{}
	}
	entry, ok := o.state.Running[result.IssueID]
	if !ok {
		return
	}

	delete(o.state.Running, result.IssueID)
	if result.Err == nil && result.Result.Usage.TotalTokens > entry.CurrentUsage.TotalTokens {
		entry.CurrentUsage = result.Result.Usage
	}
	o.finishRuntime(entry, o.now())

	if result.Err == nil && result.Result.Completed {
		o.state.Completed[result.IssueID] = struct{}{}
		o.scheduleRetry(entry.Issue, 1, true, "continuation")
		return
	}
	o.scheduleRetry(entry.Issue, nextRetryAttempt(entry.RetryAttempt), false, fmt.Sprintf("worker failed: %v", result.Err))
}

func (o *Orchestrator) recordSession(issue tracker.Issue, started agent.SessionStarted) {
	o.mu.Lock()
	defer o.mu.Unlock()
	entry, ok := o.state.Running[issue.ID]
	if !ok {
		return
	}
	entry.ThreadID = started.ThreadID
	entry.TurnID = started.TurnID
	entry.SessionID = started.ThreadID + "-" + started.TurnID
	entry.SessionLog = started.LogPath
	entry.TurnCount++
	ilog.WithSession(ilog.WithIssue(o.logger, issue.ID, issue.Identifier), entry.SessionID).Info("session outcome=started")
}

func (o *Orchestrator) recordEvent(issueID string, event agent.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	entry, ok := o.state.Running[issueID]
	if !ok {
		return
	}
	now := o.now()
	entry.LastEventAt = &now
	entry.LastEvent = event.Method
	if message, ok := event.Payload["text"].(string); ok {
		entry.LastMessage = message
	}
	entry.RecentEvents = appendRecentEvent(entry.RecentEvents, IssueEvent{
		At:      now,
		Event:   event.Method,
		Message: summarizeEvent(event),
	})
	if event.Usage != nil {
		if event.Usage.InputTokens >= entry.lastUsage.InputTokens {
			entry.CurrentUsage.InputTokens += event.Usage.InputTokens - entry.lastUsage.InputTokens
		}
		if event.Usage.OutputTokens >= entry.lastUsage.OutputTokens {
			entry.CurrentUsage.OutputTokens += event.Usage.OutputTokens - entry.lastUsage.OutputTokens
		}
		if event.Usage.TotalTokens >= entry.lastUsage.TotalTokens {
			entry.CurrentUsage.TotalTokens += event.Usage.TotalTokens - entry.lastUsage.TotalTokens
		}
		entry.lastUsage = *event.Usage
	}
	if limits := extractRateLimits(event.Payload); limits != nil {
		o.state.CodexRateLimits = limits
	}
}

func appendRecentEvent(events []IssueEvent, event IssueEvent) []IssueEvent {
	events = append(events, event)
	if len(events) > 20 {
		return append([]IssueEvent(nil), events[len(events)-20:]...)
	}
	return events
}

func summarizeEvent(event agent.Event) string {
	if text, ok := event.Payload["text"].(string); ok && strings.TrimSpace(text) != "" {
		return text
	}
	if turn, ok := event.Payload["turn"].(map[string]any); ok {
		if id, ok := turn["id"].(string); ok && id != "" {
			return id
		}
	}
	return event.Method
}

func (o *Orchestrator) applyConfigLocked(cfg config.Config) {
	o.state.PollIntervalMs = cfg.PollIntervalMs()
	o.state.MaxConcurrentAgents = cfg.MaxConcurrentAgents()
}

func normalizeAttempt(attempt int) any {
	if attempt <= 0 {
		return nil
	}
	return attempt
}

func extractRateLimits(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	if raw, ok := payload["rate_limits"].(map[string]any); ok {
		return raw
	}
	if raw, ok := payload["rateLimits"].(map[string]any); ok {
		return raw
	}
	return nil
}

type trackerAdapter struct {
	cfg func() config.Config
}

func (a trackerAdapter) client() tracker.Client {
	cfg := a.cfg()
	return tracker.Client{
		Endpoint:    cfg.TrackerEndpoint(),
		APIKey:      cfg.TrackerAPIKey(),
		ProjectSlug: cfg.TrackerProjectSlug(),
	}
}

func (a trackerAdapter) FetchCandidates(ctx context.Context, states []string) ([]tracker.Issue, error) {
	return a.client().FetchCandidates(ctx, states)
}

func (a trackerAdapter) FetchByStates(ctx context.Context, states []string) ([]tracker.Issue, error) {
	return a.client().FetchByStates(ctx, states)
}

func (a trackerAdapter) FetchStatesByIDs(ctx context.Context, ids []string) ([]tracker.Issue, error) {
	return a.client().FetchStatesByIDs(ctx, ids)
}

type runnerAdapter struct {
	cfg func() config.Config
}

func (a runnerAdapter) Run(ctx context.Context, params agent.RunParams) (agent.RunnerResult, error) {
	runner := agent.Runner{Config: a.cfg()}
	return runner.Run(ctx, params)
}
