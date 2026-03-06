package workflow

import (
	"bytes"
	"strings"
	"text/template"
)

// DefaultPrompt is used when the workflow prompt body is empty (SPEC.md §5.4).
const DefaultPrompt = "You are working on an issue from Linear."

// RenderData is the template input struct exposed to WORKFLOW.md templates.
type RenderData struct {
	Issue   map[string]any
	Attempt any // nil on first run, integer on retry/continuation
}

// Render renders the prompt template with the given issue and attempt.
// Unknown variables cause a render failure (strict mode via missingkey=error).
// If tmpl is empty, DefaultPrompt is returned.
func Render(tmpl string, data RenderData) (string, error) {
	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		return DefaultPrompt, nil
	}

	t, err := template.New("workflow").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", wrapErr(ErrTemplateParseError, "failed to parse prompt template", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", wrapErr(ErrTemplateRenderError, "failed to render prompt template", err)
	}
	return buf.String(), nil
}
