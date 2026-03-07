package app

import "testing"

func TestParseArgsSupportsWorkflowPathAndPort(t *testing.T) {
	path, port, portSet, err := parseArgs([]string{"--port", "8080", "/tmp/WORKFLOW.md"})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if path != "/tmp/WORKFLOW.md" {
		t.Fatalf("path = %q", path)
	}
	if port != 8080 || !portSet {
		t.Fatalf("port = %d portSet = %v", port, portSet)
	}
}

func TestResolvePortPrefersCLIOverWorkflowConfig(t *testing.T) {
	port, ok := resolvePort(8080, true, map[string]any{
		"server": map[string]any{"port": 9090},
	})
	if !ok || port != 8080 {
		t.Fatalf("resolvePort = (%d, %v) want (8080, true)", port, ok)
	}

	port, ok = resolvePort(-1, false, map[string]any{
		"server": map[string]any{"port": 9090},
	})
	if !ok || port != 9090 {
		t.Fatalf("resolvePort = (%d, %v) want (9090, true)", port, ok)
	}

	port, ok = resolvePort(-1, false, map[string]any{})
	if ok {
		t.Fatalf("resolvePort = (%d, %v) want disabled", port, ok)
	}
}
