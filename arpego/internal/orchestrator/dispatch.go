package orchestrator

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

func sortIssuesForDispatch(issues []tracker.Issue) []tracker.Issue {
	sorted := append([]tracker.Issue(nil), issues...)
	slices.SortStableFunc(sorted, func(a, b tracker.Issue) int {
		if cmp := cmp.Compare(priorityRank(a.Priority), priorityRank(b.Priority)); cmp != 0 {
			return cmp
		}
		if cmp := compareTimes(a.CreatedAt, b.CreatedAt); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Identifier, b.Identifier)
	})
	return sorted
}

func dispatchEligible(issue tracker.Issue, state State, cfg config.Config) bool {
	if !candidateIssue(issue, cfg) {
		return false
	}
	if _, ok := state.Claimed[issue.ID]; ok {
		return false
	}
	if _, ok := state.Running[issue.ID]; ok {
		return false
	}
	return slotsAvailable(issue, state, cfg)
}

func candidateIssue(issue tracker.Issue, cfg config.Config) bool {
	if strings.TrimSpace(issue.ID) == "" ||
		strings.TrimSpace(issue.Identifier) == "" ||
		strings.TrimSpace(issue.Title) == "" ||
		strings.TrimSpace(issue.State) == "" {
		return false
	}
	activeStates := stateSet(cfg.TrackerActiveStates())
	terminalStates := stateSet(cfg.TrackerTerminalStates())
	stateName := normalizeState(issue.State)
	if !activeStates[stateName] || terminalStates[stateName] {
		return false
	}
	if stateName == "todo" {
		for _, blocker := range issue.BlockedBy {
			if blocker.State == nil || !terminalStates[normalizeState(*blocker.State)] {
				return false
			}
		}
	}
	return true
}

func slotsAvailable(issue tracker.Issue, state State, cfg config.Config) bool {
	if len(state.Running) >= cfg.MaxConcurrentAgents() {
		return false
	}
	return stateSlotsAvailable(issue.State, state.Running, cfg)
}

func stateSlotsAvailable(issueState string, running map[string]*RunningEntry, cfg config.Config) bool {
	perState := cfg.MaxConcurrentAgentsByState()
	limit, ok := perState[normalizeState(issueState)]
	if !ok {
		limit = cfg.MaxConcurrentAgents()
	}
	used := 0
	for _, entry := range running {
		if entry != nil && normalizeState(entry.Issue.State) == normalizeState(issueState) {
			used++
		}
	}
	return used < limit
}

func compareTimes(a, b *time.Time) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	case a.Before(*b):
		return -1
	case a.After(*b):
		return 1
	default:
		return 0
	}
}

func priorityRank(priority *int) int {
	if priority == nil {
		return 5
	}
	if *priority >= 1 && *priority <= 4 {
		return *priority
	}
	return 5
}

func stateSet(states []string) map[string]bool {
	out := make(map[string]bool, len(states))
	for _, state := range states {
		if normalized := normalizeState(state); normalized != "" {
			out[normalized] = true
		}
	}
	return out
}

func normalizeState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}
