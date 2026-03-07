package agent

type TokenAccumulator struct {
	Totals   Usage
	lastSeen *Usage
}

func (a *TokenAccumulator) Add(event Event) {
	if event.Usage == nil {
		return
	}
	if a.lastSeen == nil {
		a.Totals = *event.Usage
		copyUsage := *event.Usage
		a.lastSeen = &copyUsage
		return
	}
	if event.Usage.InputTokens >= a.lastSeen.InputTokens {
		a.Totals.InputTokens += event.Usage.InputTokens - a.lastSeen.InputTokens
	}
	if event.Usage.OutputTokens >= a.lastSeen.OutputTokens {
		a.Totals.OutputTokens += event.Usage.OutputTokens - a.lastSeen.OutputTokens
	}
	if event.Usage.TotalTokens >= a.lastSeen.TotalTokens {
		a.Totals.TotalTokens += event.Usage.TotalTokens - a.lastSeen.TotalTokens
	}
	copyUsage := *event.Usage
	a.lastSeen = &copyUsage
}
