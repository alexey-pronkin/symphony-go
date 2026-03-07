package agent

import "fmt"

const (
	initializeID  = 1
	threadStartID = 2
	turnStartID   = 3
)

type Request struct {
	ID     int            `json:"id,omitempty"`
	Method string         `json:"method,omitempty"`
	Params map[string]any `json:"params,omitempty"`
	Result map[string]any `json:"result,omitempty"`
}

type Response struct {
	ID     int            `json:"id,omitempty"`
	Method string         `json:"method,omitempty"`
	Params map[string]any `json:"params,omitempty"`
	Result map[string]any `json:"result,omitempty"`
	Error  any            `json:"error,omitempty"`
	Usage  *Usage         `json:"usage,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

type SessionStarted struct {
	ThreadID string
	TurnID   string
}

type Event struct {
	Method  string
	Payload map[string]any
	Usage   *Usage
}

type RunResult struct {
	Completed bool
	Usage     Usage
}

type RunError struct {
	Kind    string
	Message string
	Cause   error
}

func (e *RunError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *RunError) Unwrap() error { return e.Cause }

func AsRunError(err error, target **RunError) bool {
	if err == nil {
		return false
	}
	runErr, ok := err.(*RunError)
	if !ok {
		return false
	}
	*target = runErr
	return true
}

const (
	ErrInvalidWorkspaceCWD = "invalid_workspace_cwd"
	ErrResponseTimeout     = "response_timeout"
	ErrTurnTimeout         = "turn_timeout"
	ErrTurnInputRequired   = "turn_input_required"
	ErrApprovalRequired    = "approval_required"
	ErrUnsupportedToolCall = "unsupported_tool_call"
	ErrProtocolPayload     = "protocol_payload_error"
)
