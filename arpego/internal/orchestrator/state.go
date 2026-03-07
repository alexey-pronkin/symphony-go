package orchestrator

import (
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/agent"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

type timerHandle interface {
	Stop() bool
}

type RetryEntry struct {
	IssueID    string
	Identifier string
	Attempt    int
	DueAt      time.Time
	Timer      timerHandle
	Error      string
}

type RunningEntry struct {
	Issue         tracker.Issue
	WorkspacePath string
	StartedAt     time.Time
	LastEventAt   *time.Time
	LastEvent     string
	LastMessage   string
	SessionID     string
	ThreadID      string
	TurnID        string
	TurnCount     int
	RetryAttempt  int
	CurrentUsage  agent.Usage
	lastUsage     agent.Usage
	cancel        func()
}

type CodexTotals struct {
	InputTokens    int
	OutputTokens   int
	TotalTokens    int
	SecondsRunning int64
}

type State struct {
	PollIntervalMs      int64
	MaxConcurrentAgents int
	Running             map[string]*RunningEntry
	Claimed             map[string]struct{}
	RetryAttempts       map[string]RetryEntry
	Completed           map[string]struct{}
	CodexTotals         CodexTotals
	CodexRateLimits     map[string]any
}

func newState() State {
	return State{
		Running:       map[string]*RunningEntry{},
		Claimed:       map[string]struct{}{},
		RetryAttempts: map[string]RetryEntry{},
		Completed:     map[string]struct{}{},
	}
}
