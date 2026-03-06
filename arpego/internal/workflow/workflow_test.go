package workflow_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/workflow"
)

// ---- Load / parse tests ----

func TestLoad_MissingFile(t *testing.T) {
	_, err := workflow.Load("/nonexistent/WORKFLOW.md")
	assertErrorKind(t, err, workflow.ErrMissingWorkflowFile)
}

func TestLoad_EmptyPath_UsesDefault(t *testing.T) {
	// Write a minimal WORKFLOW.md in a temp dir and chdir there.
	dir := t.TempDir()
	write(t, filepath.Join(dir, "WORKFLOW.md"), "hello")
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	_ = os.Chdir(dir)

	def, err := workflow.Load("")
	mustOK(t, err)
	if def.PromptTemplate != "hello" {
		t.Fatalf("expected prompt 'hello', got %q", def.PromptTemplate)
	}
}

func TestLoad_NoFrontMatter(t *testing.T) {
	path := writeTmp(t, "just a prompt body\n")
	def, err := workflow.Load(path)
	mustOK(t, err)
	if len(def.Config) != 0 {
		t.Fatalf("expected empty config, got %v", def.Config)
	}
	if def.PromptTemplate != "just a prompt body" {
		t.Fatalf("unexpected prompt: %q", def.PromptTemplate)
	}
}

func TestLoad_WithFrontMatter(t *testing.T) {
	content := `---
tracker:
  kind: linear
  project_slug: my-proj
---

Work on this issue.
`
	path := writeTmp(t, content)
	def, err := workflow.Load(path)
	mustOK(t, err)

	tracker, ok := def.Config["tracker"].(map[string]any)
	if !ok {
		t.Fatalf("config.tracker should be a map, got %T", def.Config["tracker"])
	}
	if tracker["kind"] != "linear" {
		t.Fatalf("expected tracker.kind=linear, got %v", tracker["kind"])
	}
	if def.PromptTemplate != "Work on this issue." {
		t.Fatalf("unexpected prompt: %q", def.PromptTemplate)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTmp(t, "---\n: invalid: yaml: [\n---\nbody\n")
	_, err := workflow.Load(path)
	assertErrorKind(t, err, workflow.ErrWorkflowParseError)
}

func TestLoad_FrontMatterNotAMap(t *testing.T) {
	path := writeTmp(t, "---\n- item1\n- item2\n---\nbody\n")
	_, err := workflow.Load(path)
	assertErrorKind(t, err, workflow.ErrFrontMatterNotAMap)
}

func TestLoad_PromptBodyTrimmed(t *testing.T) {
	path := writeTmp(t, "---\nkey: val\n---\n\n  trimmed  \n\n")
	def, err := workflow.Load(path)
	mustOK(t, err)
	if def.PromptTemplate != "trimmed" {
		t.Fatalf("expected trimmed body, got %q", def.PromptTemplate)
	}
}

// ---- Template rendering tests ----

func TestRender_BasicIssueField(t *testing.T) {
	tmpl := "Working on: {{ .Issue.title }}"
	data := workflow.RenderData{
		Issue: map[string]any{"title": "Fix bug"},
	}
	out, err := workflow.Render(tmpl, data)
	mustOK(t, err)
	if out != "Working on: Fix bug" {
		t.Fatalf("unexpected render: %q", out)
	}
}

func TestRender_UnknownVariable_Fails(t *testing.T) {
	tmpl := "{{ .Issue.nonexistent }}"
	data := workflow.RenderData{Issue: map[string]any{}}
	_, err := workflow.Render(tmpl, data)
	assertErrorKind(t, err, workflow.ErrTemplateRenderError)
}

func TestRender_EmptyTemplate_UsesDefault(t *testing.T) {
	out, err := workflow.Render("", workflow.RenderData{Issue: map[string]any{}})
	mustOK(t, err)
	if out != workflow.DefaultPrompt {
		t.Fatalf("expected default prompt, got %q", out)
	}
}

func TestRender_AttemptVariable(t *testing.T) {
	attempt := 2
	tmpl := "Attempt: {{ .Attempt }}"
	data := workflow.RenderData{Issue: map[string]any{}, Attempt: attempt}
	out, err := workflow.Render(tmpl, data)
	mustOK(t, err)
	if out != "Attempt: 2" {
		t.Fatalf("unexpected render: %q", out)
	}
}

func TestRender_InvalidTemplate_Fails(t *testing.T) {
	_, err := workflow.Render("{{ .Unclosed", workflow.RenderData{Issue: map[string]any{}})
	assertErrorKind(t, err, workflow.ErrTemplateParseError)
}

// ---- Watcher test ----

func TestWatch_ReloadsOnChange(t *testing.T) {
	path := writeTmp(t, "---\nkey: v1\n---\nbody\n")

	def, err := workflow.Load(path)
	mustOK(t, err)

	reloaded := make(chan *workflow.Definition, 1)
	closer, err := workflow.Watch(path, def, func(d *workflow.Definition) {
		reloaded <- d
	})
	mustOK(t, err)
	t.Cleanup(func() { _ = closer.Close() })

	// Overwrite the file.
	write(t, path, "---\nkey: v2\n---\nupdated body\n")

	select {
	case d := <-reloaded:
		if d.Config["key"] != "v2" {
			t.Fatalf("expected key=v2, got %v", d.Config["key"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for reload")
	}
}

func TestWatch_InvalidReload_KeepsLast(t *testing.T) {
	path := writeTmp(t, "---\nkey: v1\n---\nbody\n")
	def, _ := workflow.Load(path)

	reloaded := make(chan struct{}, 1)
	closer, _ := workflow.Watch(path, def, func(_ *workflow.Definition) {
		reloaded <- struct{}{}
	})
	t.Cleanup(func() { _ = closer.Close() })

	// Write invalid YAML — should NOT trigger onChange.
	write(t, path, "---\n: bad: yaml: [\n---\nbody\n")

	select {
	case <-reloaded:
		t.Fatal("onChange should not be called on invalid reload")
	case <-time.After(600 * time.Millisecond):
		// Expected: no reload fired.
	}
}

// ---- helpers ----

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeTmp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "workflow-*.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return f.Name()
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertErrorKind(t *testing.T, err error, kind string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with kind %q, got nil", kind)
	}
	we, ok := err.(*workflow.WorkflowError)
	if !ok {
		t.Fatalf("expected *WorkflowError, got %T: %v", err, err)
	}
	if we.Kind != kind {
		t.Fatalf("expected error kind %q, got %q", kind, we.Kind)
	}
}
