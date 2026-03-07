package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/workflow"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/workspace"
)

type Runner struct {
	Config        config.Config
	ClientOptions []ClientOption
}

type RunParams struct {
	Issue         tracker.Issue
	Attempt       any
	WorkspacePath string
	Workflow      *workflow.Definition
	OnEvent       func(Event)
	OnSession     func(SessionStarted)
	RefreshIssue  func(context.Context, string) (tracker.Issue, bool, error)
}

type RunnerResult struct {
	Session   SessionStarted
	Completed bool
	Usage     Usage
}

func (r Runner) Run(ctx context.Context, params RunParams) (RunnerResult, error) {
	if err := workspace.ValidatePath(r.Config.WorkspaceRoot(), params.WorkspacePath); err != nil {
		return RunnerResult{}, &RunError{Kind: ErrInvalidWorkspaceCWD, Message: err.Error(), Cause: err}
	}
	if params.Workflow == nil {
		return RunnerResult{}, fmt.Errorf("workflow definition is required")
	}
	prompt, err := workflow.Render(params.Workflow.PromptTemplate, workflow.RenderData{
		Issue:   issueTemplateData(params.Issue),
		Attempt: params.Attempt,
	})
	if err != nil {
		return RunnerResult{}, err
	}
	logPath := filepath.Join(params.WorkspacePath, ".symphony", "session.jsonl")
	clientOptions := append([]ClientOption{}, r.ClientOptions...)
	clientOptions = append(clientOptions, WithProtocolLog(logPath))
	client, err := NewClient(params.WorkspacePath, r.Config.CodexCommand(), clientOptions...)
	if err != nil {
		return RunnerResult{}, err
	}
	defer func() {
		_ = client.Close()
	}()

	sess := NewSession(client, r.Config)
	started, err := sess.Start(ctx, StartParams{
		WorkspacePath: params.WorkspacePath,
		Prompt:        prompt,
		Title:         params.Issue.Identifier + ": " + params.Issue.Title,
	})
	if err != nil {
		return RunnerResult{}, err
	}
	started.LogPath = logPath
	if params.OnSession != nil {
		params.OnSession(started)
	}
	var result RunResult
	for turn := 1; ; turn++ {
		result, err = sess.Run(ctx, params.OnEvent)
		if err != nil {
			return RunnerResult{}, err
		}
		if turn >= r.Config.MaxTurns() || params.RefreshIssue == nil {
			break
		}
		_, shouldContinue, err := params.RefreshIssue(ctx, params.Issue.ID)
		if err != nil {
			return RunnerResult{}, err
		}
		if !shouldContinue {
			break
		}
		started, err = sess.StartTurn(ctx, StartParams{
			WorkspacePath: params.WorkspacePath,
			Prompt:        continuationPrompt(),
			Title:         params.Issue.Identifier + ": " + params.Issue.Title,
		})
		if err != nil {
			return RunnerResult{}, err
		}
		started.LogPath = logPath
		if params.OnSession != nil {
			params.OnSession(started)
		}
	}
	return RunnerResult{Session: started, Completed: result.Completed, Usage: result.Usage}, nil
}

func issueTemplateData(issue tracker.Issue) map[string]any {
	blockedBy := make([]map[string]any, 0, len(issue.BlockedBy))
	for _, blocker := range issue.BlockedBy {
		blockedBy = append(blockedBy, map[string]any{
			"id":         stringValueOrNil(blocker.ID),
			"identifier": stringValueOrNil(blocker.Identifier),
			"state":      stringValueOrNil(blocker.State),
		})
	}
	return map[string]any{
		"id":          issue.ID,
		"identifier":  issue.Identifier,
		"title":       issue.Title,
		"description": stringValueOrNil(issue.Description),
		"priority":    intValueOrNil(issue.Priority),
		"state":       issue.State,
		"branch_name": stringValueOrNil(issue.BranchName),
		"url":         stringValueOrNil(issue.URL),
		"labels":      append([]string(nil), issue.Labels...),
		"blocked_by":  blockedBy,
		"created_at":  timeValueOrNil(issue.CreatedAt),
		"updated_at":  timeValueOrNil(issue.UpdatedAt),
	}
}

func continuationPrompt() string {
	return strings.TrimSpace(
		"Continue from the existing thread context. " +
			"Do not repeat the original task prompt. " +
			"Focus on the next concrete actions and current blockers.",
	)
}

func stringValueOrNil(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func intValueOrNil(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func timeValueOrNil(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}
