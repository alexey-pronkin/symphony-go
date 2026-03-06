package workflow

import "fmt"

// Error kinds as sentinel strings matching SPEC.md §5.5.
const (
	ErrMissingWorkflowFile = "missing_workflow_file"
	ErrWorkflowParseError  = "workflow_parse_error"
	ErrFrontMatterNotAMap  = "workflow_front_matter_not_a_map"
	ErrTemplateParseError  = "template_parse_error"
	ErrTemplateRenderError = "template_render_error"
)

// WorkflowError is a typed error for workflow loading and rendering failures.
type WorkflowError struct {
	Kind    string
	Message string
	Cause   error
}

func (e *WorkflowError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *WorkflowError) Unwrap() error { return e.Cause }

func wrapErr(kind, msg string, cause error) *WorkflowError {
	return &WorkflowError{Kind: kind, Message: msg, Cause: cause}
}
