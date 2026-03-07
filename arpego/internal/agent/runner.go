package agent

import (
	"context"
	"fmt"

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
		Issue: map[string]any{
			"id":         params.Issue.ID,
			"identifier": params.Issue.Identifier,
			"title":      params.Issue.Title,
			"state":      params.Issue.State,
			"labels":     params.Issue.Labels,
		},
		Attempt: params.Attempt,
	})
	if err != nil {
		return RunnerResult{}, err
	}
	client, err := NewClient(params.WorkspacePath, r.Config.CodexCommand(), r.ClientOptions...)
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
	if params.OnSession != nil {
		params.OnSession(started)
	}
	result, err := sess.Run(ctx, params.OnEvent)
	if err != nil {
		return RunnerResult{}, err
	}
	return RunnerResult{Session: started, Completed: result.Completed, Usage: result.Usage}, nil
}
