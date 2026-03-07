package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
)

type StartParams struct {
	WorkspacePath string
	Prompt        string
	Title         string
}

type Session struct {
	client *Client
	cfg    config.Config
	state  SessionStarted
	acc    TokenAccumulator
}

func NewSession(client *Client, cfg config.Config) *Session {
	return &Session{client: client, cfg: cfg}
}

func (s *Session) Start(ctx context.Context, params StartParams) (SessionStarted, error) {
	if err := s.client.Send(Request{ID: initializeID, Method: "initialize", Params: map[string]any{
		"capabilities": map[string]any{"experimentalApi": true},
		"clientInfo":   map[string]any{"name": "symphony-orchestrator", "title": "Symphony Orchestrator", "version": "0.1.0"},
	}}); err != nil {
		return SessionStarted{}, err
	}
	if _, err := s.client.AwaitResponse(
		ctx,
		initializeID,
		time.Duration(s.cfg.CodexReadTimeoutMs())*time.Millisecond,
	); err != nil {
		return SessionStarted{}, normalizeTimeout(err)
	}
	if err := s.client.Send(Request{Method: "initialized", Params: map[string]any{}}); err != nil {
		return SessionStarted{}, err
	}
	if err := s.client.Send(Request{ID: threadStartID, Method: "thread/start", Params: map[string]any{
		"cwd":            params.WorkspacePath,
		"approvalPolicy": approvalPolicy(s.cfg),
		"sandbox":        emptyToNil(s.cfg.CodexThreadSandbox()),
		"dynamicTools":   []any{},
	}}); err != nil {
		return SessionStarted{}, err
	}
	threadResp, err := s.client.AwaitResponse(
		ctx,
		threadStartID,
		time.Duration(s.cfg.CodexReadTimeoutMs())*time.Millisecond,
	)
	if err != nil {
		return SessionStarted{}, normalizeTimeout(err)
	}
	threadID, err := extractNestedID(threadResp.Result, "thread")
	if err != nil {
		return SessionStarted{}, err
	}
	s.state = SessionStarted{ThreadID: threadID}
	return s.StartTurn(ctx, params)
}

func (s *Session) StartTurn(ctx context.Context, params StartParams) (SessionStarted, error) {
	if s.state.ThreadID == "" {
		return SessionStarted{}, &RunError{Kind: ErrProtocolPayload, Message: "thread not started"}
	}
	if err := s.client.Send(Request{ID: turnStartID, Method: "turn/start", Params: map[string]any{
		"threadId":       s.state.ThreadID,
		"cwd":            params.WorkspacePath,
		"title":          params.Title,
		"approvalPolicy": approvalPolicy(s.cfg),
		"sandboxPolicy":  emptyToNil(s.cfg.CodexTurnSandboxPolicy()),
		"input": []map[string]any{{
			"type": "text",
			"text": params.Prompt,
		}},
	}}); err != nil {
		return SessionStarted{}, err
	}
	turnResp, err := s.client.AwaitResponse(ctx, turnStartID, time.Duration(s.cfg.CodexReadTimeoutMs())*time.Millisecond)
	if err != nil {
		return SessionStarted{}, normalizeTimeout(err)
	}
	turnID, err := extractNestedID(turnResp.Result, "turn")
	if err != nil {
		return SessionStarted{}, err
	}
	s.state.TurnID = turnID
	return s.state, nil
}

func (s *Session) Run(ctx context.Context, onEvent func(Event)) (RunResult, error) {
	turnTimeout := time.Duration(s.cfg.CodexTurnTimeoutMs()) * time.Millisecond
	if turnTimeout <= 0 {
		turnTimeout = time.Hour
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, turnTimeout)
	defer cancel()

	for {
		line, err := s.client.ReadLine(deadlineCtx, turnTimeout)
		if err != nil {
			if strings.Contains(err.Error(), ErrResponseTimeout) || deadlineCtx.Err() == context.DeadlineExceeded {
				return RunResult{}, &RunError{Kind: ErrTurnTimeout, Message: "turn exceeded timeout", Cause: err}
			}
			return RunResult{}, err
		}
		var msg Response
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.Method == "" {
			continue
		}
		event := Event{Method: msg.Method, Payload: mapWithUsage(msg), Usage: msg.Usage}
		if msg.Usage == nil {
			event.Usage = usageFrom(msg.Params)
		}
		s.acc.Add(event)
		if onEvent != nil {
			onEvent(event)
		}
		switch msg.Method {
		case "turn/completed":
			return RunResult{Completed: true, Usage: s.acc.Totals}, nil
		case "turn/failed":
			return RunResult{}, &RunError{Kind: ErrProtocolPayload, Message: "turn failed"}
		case "turn/input_required", "turn/needs_input", "turn/request_input", "turn/approval_required":
			return RunResult{}, &RunError{Kind: ErrTurnInputRequired, Message: "turn requires user input"}
		case "item/commandExecution/requestApproval",
			"item/fileChange/requestApproval",
			"execCommandApproval",
			"applyPatchApproval":
			if approvalPolicy(s.cfg) == "never" {
				if err := s.client.Send(map[string]any{"id": msg.ID, "result": map[string]any{"approved": true}}); err != nil {
					return RunResult{}, err
				}
				continue
			}
			return RunResult{}, &RunError{Kind: ErrApprovalRequired, Message: "approval required"}
		case "item/tool/call":
			if err := s.handleToolCall(msg); err != nil {
				return RunResult{}, err
			}
		}
	}
}

func (s *Session) handleToolCall(msg Response) error {
	result := map[string]any{"success": false, "error": ErrUnsupportedToolCall}
	if err := s.client.Send(map[string]any{"id": msg.ID, "result": result}); err != nil {
		return err
	}
	return nil
}

func approvalPolicy(cfg config.Config) any {
	if v := strings.TrimSpace(cfg.CodexApprovalPolicy()); v != "" {
		return v
	}
	return nil
}

func usageFrom(params map[string]any) *Usage {
	if params == nil {
		return nil
	}
	raw, _ := params["usage"].(map[string]any)
	if raw == nil {
		return nil
	}
	return &Usage{
		InputTokens:  intValue(raw["input_tokens"]),
		OutputTokens: intValue(raw["output_tokens"]),
		TotalTokens:  intValue(raw["total_tokens"]),
	}
}

func intValue(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

func mapWithUsage(msg Response) map[string]any {
	payload := map[string]any{}
	for k, v := range msg.Params {
		payload[k] = v
	}
	return payload
}

func extractNestedID(result map[string]any, key string) (string, error) {
	entry, _ := result[key].(map[string]any)
	id, _ := entry["id"].(string)
	if id == "" {
		return "", &RunError{Kind: ErrProtocolPayload, Message: fmt.Sprintf("missing %s.id in response", key)}
	}
	return id, nil
}

func normalizeTimeout(err error) error {
	var runErr *RunError
	if AsRunError(err, &runErr) && runErr.Kind == ErrResponseTimeout {
		return runErr
	}
	return err
}

func emptyToNil(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
