// Package workflow loads and parses WORKFLOW.md files per SPEC.md §5.
package workflow

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultWorkflowFile is used when no explicit path is provided.
const DefaultWorkflowFile = "WORKFLOW.md"

// Definition holds the parsed result of a WORKFLOW.md file.
type Definition struct {
	// Config is the YAML front matter root object (empty map if no front matter).
	Config map[string]any
	// PromptTemplate is the trimmed Markdown body after front matter.
	PromptTemplate string
}

// Load reads and parses a WORKFLOW.md file.
// If path is empty, DefaultWorkflowFile in the working directory is used.
func Load(path string) (*Definition, error) {
	if path == "" {
		path = DefaultWorkflowFile
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, wrapErr(ErrMissingWorkflowFile, fmt.Sprintf("file not found: %s", path), err)
		}
		return nil, wrapErr(ErrMissingWorkflowFile, fmt.Sprintf("cannot read file: %s", path), err)
	}
	return parse(data)
}

// parse splits YAML front matter from the prompt body and validates both.
// Per SPEC.md §5.2: if the file starts with "---", front matter runs until
// the next "---" line. Absent front matter → empty config, full content as body.
func parse(data []byte) (*Definition, error) {
	lines := splitLines(string(data))

	cfg := map[string]any{}
	body := strings.TrimSpace(strings.Join(lines, "\n"))

	if len(lines) > 0 && lines[0] == "---" {
		// Find closing ---
		closing := -1
		for i := 1; i < len(lines); i++ {
			if lines[i] == "---" {
				closing = i
				break
			}
		}
		if closing > 0 {
			frontMatterLines := lines[1:closing]
			bodyLines := lines[closing+1:]

			var raw any
			if err := yaml.Unmarshal([]byte(strings.Join(frontMatterLines, "\n")), &raw); err != nil {
				return nil, wrapErr(ErrWorkflowParseError, "invalid YAML front matter", err)
			}
			if raw != nil {
				m, ok := raw.(map[string]any)
				if !ok {
					return nil, wrapErr(ErrFrontMatterNotAMap,
						fmt.Sprintf("front matter must be a YAML map, got %T", raw), nil)
				}
				cfg = m
			}
			body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
		}
	}

	return &Definition{
		Config:         cfg,
		PromptTemplate: body,
	}, nil
}

// splitLines splits s into lines, stripping trailing \r from each.
func splitLines(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, len(raw))
	for i, l := range raw {
		out[i] = strings.TrimRight(l, "\r")
	}
	return out
}
