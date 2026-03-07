package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestIssueAndSessionContextFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, LevelInfo)
	WithSession(WithIssue(logger, "issue-1", "MT-1"), "thread-1-turn-1").Info("dispatch outcome=started")

	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if payload["issue_id"] != "issue-1" {
		t.Fatalf("issue_id = %#v", payload["issue_id"])
	}
	if payload["issue_identifier"] != "MT-1" {
		t.Fatalf("issue_identifier = %#v", payload["issue_identifier"])
	}
	if payload["session_id"] != "thread-1-turn-1" {
		t.Fatalf("session_id = %#v", payload["session_id"])
	}
	if payload["level"] != slog.LevelInfo.String() {
		t.Fatalf("level = %#v", payload["level"])
	}
}
