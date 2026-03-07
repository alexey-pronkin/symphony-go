package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/workflow"
)

func TestSessionStartHandshakeOrder(t *testing.T) {
	workspace := t.TempDir()
	logPath := filepath.Join(workspace, "messages.log")
	script := writeScript(t, workspace, `
set -eu
log_file="$APP_LOG"
while IFS= read -r line; do
  printf '%s\n' "$line" >> "$log_file"
  method=$(printf '%s' "$line" | sed -n 's/.*"method":"\([^"]*\)".*/\1/p')
  id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  case "$method" in
    initialize)
      printf '{"id":%s,"result":{"protocolVersion":"2026-01-01"}}\n' "$id"
      ;;
    initialized)
      ;;
    thread/start)
      printf '{"id":%s,"result":{"thread":{"id":"thread-1"}}}\n' "$id"
      ;;
    turn/start)
      printf '{"id":%s,"result":{"turn":{"id":"turn-1"}}}\n' "$id"
      printf '{"method":"turn/completed","params":{}}\n'
      exit 0
      ;;
  esac
done
`)

	cfg := config.New(map[string]any{
		"codex": map[string]any{"command": "sh " + script},
	})
	cfgEnv := append(os.Environ(), "APP_LOG="+logPath)

	client, err := NewClient(workspace, cfg.CodexCommand(), WithEnv(cfgEnv), WithReadTimeout(200*time.Millisecond))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	sess := NewSession(client, cfg)
	started, err := sess.Start(context.Background(), StartParams{
		WorkspacePath: workspace,
		Prompt:        "Solve it",
		Title:         "MT-1: Test",
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if started.ThreadID != "thread-1" || started.TurnID != "turn-1" {
		t.Fatalf("unexpected session ids: %#v", started)
	}

	logged := strings.Split(strings.TrimSpace(readFile(t, logPath)), "\n")
	if len(logged) != 4 {
		t.Fatalf("expected 4 messages, got %d: %v", len(logged), logged)
	}

	methods := []string{
		extractMethod(t, logged[0]),
		extractMethod(t, logged[1]),
		extractMethod(t, logged[2]),
		extractMethod(t, logged[3]),
	}
	expected := []string{"initialize", "initialized", "thread/start", "turn/start"}
	for i := range expected {
		if methods[i] != expected[i] {
			t.Fatalf("message %d method = %q, want %q", i, methods[i], expected[i])
		}
	}
}

func TestSessionBuffersPartialLinesUntilNewline(t *testing.T) {
	workspace := t.TempDir()
	script := writeScript(t, workspace, `
set -eu
turn_started=0
while IFS= read -r line; do
  method=$(printf '%s' "$line" | sed -n 's/.*"method":"\([^"]*\)".*/\1/p')
  id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  case "$method" in
    initialize)
      printf '{"id":%s,"result":{}}\n' "$id"
      ;;
    initialized)
      ;;
    thread/start)
      printf '{"id":%s,"result":{"thread":{"id":"thread-1"}}}\n' "$id"
      ;;
    turn/start)
      printf '{"id":%s,"result":{"turn":{"id":"turn-1"}}}\n' "$id"
      printf '{"method":"turn/started","params":{"turn":{"id":"turn-1"}},"usage":'
      sleep 0.1
      printf '{"input_tokens":3,"output_tokens":4,"total_tokens":7}}\n'
      printf '{"method":"turn/completed","params":{}}\n'
      exit 0
      ;;
  esac
done
`)

	cfg := config.New(map[string]any{"codex": map[string]any{"command": "sh " + script}})
	client, err := NewClient(workspace, cfg.CodexCommand(), WithReadTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	sess := NewSession(client, cfg)
	_, err = sess.Start(context.Background(), StartParams{WorkspacePath: workspace, Prompt: "Prompt", Title: "MT-1: Test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var events []Event
	result, err := sess.Run(context.Background(), func(event Event) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.Completed {
		t.Fatalf("expected completed result")
	}
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
	if events[0].Method != "turn/started" {
		t.Fatalf("first event method = %q", events[0].Method)
	}
	if events[0].Usage == nil || events[0].Usage.TotalTokens != 7 {
		t.Fatalf("missing usage on turn/started event: %#v", events[0].Usage)
	}
}

func TestSessionAutoApprovesAndRejectsUnsupportedToolCalls(t *testing.T) {
	workspace := t.TempDir()
	logPath := filepath.Join(workspace, "responses.log")
	script := writeScript(t, workspace, `
set -eu
log_file="$APP_LOG"
while IFS= read -r line; do
  method=$(printf '%s' "$line" | sed -n 's/.*"method":"\([^"]*\)".*/\1/p')
  id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  printf '%s\n' "$line" >> "$log_file"
  case "$method" in
    initialize)
      printf '{"id":%s,"result":{}}\n' "$id"
      ;;
    initialized)
      ;;
    thread/start)
      printf '{"id":%s,"result":{"thread":{"id":"thread-1"}}}\n' "$id"
      ;;
    turn/start)
      printf '{"id":%s,"result":{"turn":{"id":"turn-1"}}}\n' "$id"
      printf '{"id":91,"method":"item/commandExecution/requestApproval","params":{"command":"git status"}}\n'
      IFS= read -r approval
      printf '%s\n' "$approval" >> "$log_file"
      printf '{"id":92,"method":"item/tool/call","params":{"tool":"unknown_tool","arguments":{"x":1}}}\n'
      IFS= read -r tool_result
      printf '%s\n' "$tool_result" >> "$log_file"
      printf '{"method":"turn/completed","params":{}}\n'
      exit 0
      ;;
  esac
done
`)

	cfg := config.New(map[string]any{"codex": map[string]any{"command": "sh " + script, "approval_policy": "never"}})
	client, err := NewClient(
		workspace,
		cfg.CodexCommand(),
		WithEnv(append(os.Environ(), "APP_LOG="+logPath)),
		WithReadTimeout(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	sess := NewSession(client, cfg)
	_, err = sess.Start(context.Background(), StartParams{WorkspacePath: workspace, Prompt: "Prompt", Title: "MT-1: Test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	_, err = sess.Run(context.Background(), func(Event) {})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(readFile(t, logPath)), "\n")
	if len(lines) < 6 {
		t.Fatalf("expected response lines, got %d: %v", len(lines), lines)
	}
	approval := parseJSONLine(t, lines[4])
	if got := approval["id"]; got != float64(91) {
		t.Fatalf("approval response id = %#v", got)
	}
	approvalResult := approval["result"].(map[string]any)
	if approved, _ := approvalResult["approved"].(bool); !approved {
		t.Fatalf("approval response = %#v, want approved=true", approvalResult)
	}

	toolResp := parseJSONLine(t, lines[5])
	toolResult := toolResp["result"].(map[string]any)
	if success, _ := toolResult["success"].(bool); success {
		t.Fatalf("unsupported tool call should fail: %#v", toolResult)
	}
	if toolResult["error"] != "unsupported_tool_call" {
		t.Fatalf("unexpected tool error: %#v", toolResult)
	}
}

func TestSessionFailsOnInputRequired(t *testing.T) {
	workspace := t.TempDir()
	script := writeScript(t, workspace, `
set -eu
while IFS= read -r line; do
  method=$(printf '%s' "$line" | sed -n 's/.*"method":"\([^"]*\)".*/\1/p')
  id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  case "$method" in
    initialize)
      printf '{"id":%s,"result":{}}\n' "$id"
      ;;
    initialized)
      ;;
    thread/start)
      printf '{"id":%s,"result":{"thread":{"id":"thread-1"}}}\n' "$id"
      ;;
    turn/start)
      printf '{"id":%s,"result":{"turn":{"id":"turn-1"}}}\n' "$id"
      printf '{"method":"turn/input_required","params":{"requiresInput":true}}\n'
      exit 0
      ;;
  esac
done
`)

	cfg := config.New(map[string]any{"codex": map[string]any{"command": "sh " + script}})
	client, err := NewClient(workspace, cfg.CodexCommand(), WithReadTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	sess := NewSession(client, cfg)
	_, err = sess.Start(context.Background(), StartParams{WorkspacePath: workspace, Prompt: "Prompt", Title: "MT-1: Test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	_, err = sess.Run(context.Background(), func(Event) {})
	if err == nil {
		t.Fatal("Run error = nil, want turn_input_required")
	}
	var runErr *RunError
	if !AsRunError(err, &runErr) || runErr.Kind != ErrTurnInputRequired {
		t.Fatalf("Run error = %v, want %s", err, ErrTurnInputRequired)
	}
}

func TestTokenAccumulatorUsesDeltasFromThreadTotals(t *testing.T) {
	acc := TokenAccumulator{}
	acc.Add(Event{Usage: &Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}})
	acc.Add(Event{Usage: &Usage{InputTokens: 14, OutputTokens: 9, TotalTokens: 23}})
	acc.Add(Event{Usage: &Usage{InputTokens: 12, OutputTokens: 7, TotalTokens: 19}})

	if acc.Totals.InputTokens != 14 || acc.Totals.OutputTokens != 9 || acc.Totals.TotalTokens != 23 {
		t.Fatalf("unexpected totals: %#v", acc.Totals)
	}
}

func TestRunnerValidatesWorkspaceAndRendersPrompt(t *testing.T) {
	workspaceRoot := t.TempDir()
	workspacePath := filepath.Join(workspaceRoot, "MT-1")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	logPath := filepath.Join(workspaceRoot, "turn.json")
	script := writeScript(t, workspaceRoot, `
set -eu
while IFS= read -r line; do
  method=$(printf '%s' "$line" | sed -n 's/.*"method":"\([^"]*\)".*/\1/p')
  id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  case "$method" in
    initialize)
      printf '{"id":%s,"result":{}}\n' "$id"
      ;;
    initialized)
      ;;
    thread/start)
      printf '{"id":%s,"result":{"thread":{"id":"thread-1"}}}\n' "$id"
      ;;
    turn/start)
      printf '%s\n' "$line" > "$APP_LOG"
      printf '{"id":%s,"result":{"turn":{"id":"turn-1"}}}\n' "$id"
      printf '{"method":"turn/completed","params":{}}\n'
      exit 0
      ;;
  esac
done
`)

	cfg := config.New(map[string]any{
		"workspace": map[string]any{"root": workspaceRoot},
		"codex":     map[string]any{"command": "sh " + script},
	})

	runner := Runner{
		Config: cfg,
		ClientOptions: []ClientOption{
			WithEnv(append(os.Environ(), "APP_LOG="+logPath)),
			WithReadTimeout(500 * time.Millisecond),
		},
	}
	issue := tracker.Issue{ID: "1", Identifier: "MT-1", Title: "Runner Test"}
	def := &workflow.Definition{PromptTemplate: "Issue {{ .Issue.identifier }} attempt {{ .Attempt }}"}
	result, err := runner.Run(context.Background(), RunParams{
		Issue:         issue,
		Attempt:       2,
		WorkspacePath: workspacePath,
		Workflow:      def,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.Completed || result.Session.ThreadID != "thread-1" {
		t.Fatalf("unexpected runner result: %#v", result)
	}
	turnStart := parseJSONLine(t, readFile(t, logPath))
	params := turnStart["params"].(map[string]any)
	input := params["input"].([]any)
	payload := input[0].(map[string]any)
	if payload["text"] != "Issue MT-1 attempt 2" {
		t.Fatalf("unexpected rendered prompt: %#v", payload["text"])
	}
}

func writeScript(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-codex.sh")
	content := "#!/bin/sh\n" + strings.TrimSpace(body) + "\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return strings.TrimSpace(string(data))
}

func extractMethod(t *testing.T, line string) string {
	t.Helper()
	payload := parseJSONLine(t, line)
	method, _ := payload["method"].(string)
	return method
}

func parseJSONLine(t *testing.T, line string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("Unmarshal(%q): %v", line, err)
	}
	return payload
}
