package agent

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

const linearGraphQLToolName = "linear_graphql"

// linearGraphQLToolSchema returns the JSON Schema descriptor for the linear_graphql
// dynamic tool as required by the Codex thread/start dynamicTools field.
func linearGraphQLToolSchema() map[string]any {
	return map[string]any{
		"name":        linearGraphQLToolName,
		"description": "Execute a raw GraphQL query or mutation against the Linear API using Symphony's configured tracker authentication.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "A single GraphQL operation (query or mutation).",
				},
				"variables": map[string]any{
					"type":        "object",
					"description": "Optional variables for the GraphQL operation.",
				},
			},
			"required": []string{"query"},
		},
	}
}

// buildDynamicTools returns the dynamicTools slice for thread/start.
// The linear_graphql tool is included only when the tracker is configured as Linear
// with a non-empty API key; otherwise an empty slice is returned so Codex discovers
// no custom tools.
func buildDynamicTools(cfg interface {
	TrackerKind() string
	TrackerAPIKey() string
}) []any {
	if cfg.TrackerKind() == "linear" && strings.TrimSpace(cfg.TrackerAPIKey()) != "" {
		return []any{linearGraphQLToolSchema()}
	}
	return []any{}
}

// handleLinearGraphQL dispatches an item/tool/call message for the linear_graphql
// tool. It validates the arguments, executes the GraphQL call via the configured
// tracker client, and sends the structured result back on the wire.
func (s *Session) handleLinearGraphQL(ctx context.Context, msg Response) error {
	result := s.executeLinearGraphQL(ctx, msg)
	return s.client.Send(map[string]any{"id": msg.ID, "result": result})
}

// executeLinearGraphQL performs validation and the HTTP call. Separated from
// handleLinearGraphQL so unit tests can exercise the logic without a real client.
func (s *Session) executeLinearGraphQL(ctx context.Context, msg Response) map[string]any {
	args, _ := msg.Params["arguments"].(map[string]any)

	// query must be a non-empty string
	rawQuery, _ := args["query"].(string)
	if strings.TrimSpace(rawQuery) == "" {
		return toolCallError("query must be a non-empty string")
	}

	// Reject documents with multiple operations; Linear will also reject them, but
	// catching it early gives a cleaner error message.
	if hasMultipleOperations(rawQuery) {
		return toolCallError("query must contain exactly one GraphQL operation")
	}

	// variables, if present, must be an object
	var variables map[string]any
	if v, ok := args["variables"]; ok && v != nil {
		variables, ok = v.(map[string]any)
		if !ok {
			return toolCallError("variables must be an object")
		}
	}

	// Auth must be configured
	if s.cfg.TrackerKind() != "linear" || strings.TrimSpace(s.cfg.TrackerAPIKey()) == "" {
		return toolCallError("linear tracker is not configured")
	}

	client := tracker.Client{
		Endpoint:   s.cfg.TrackerEndpoint(),
		APIKey:     s.cfg.TrackerAPIKey(),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}

	payload, err := client.RawQuery(ctx, rawQuery, variables)
	if err != nil {
		return toolCallError(err.Error())
	}

	// Preserve full body on GraphQL errors so the agent can inspect them.
	if hasGraphQLErrors(payload) {
		return map[string]any{"success": false, "data": payload}
	}
	return map[string]any{"success": true, "data": payload}
}

// hasMultipleOperations reports whether q appears to contain more than one
// GraphQL operation definition. This is a conservative heuristic — it strips
// comments and string literals before scanning for operation keywords.
// Linear's API server is the authoritative validator; this check is defense-in-depth.
func hasMultipleOperations(q string) bool {
	count := 0
	i := 0
	for i < len(q) {
		// Line comment: skip to end of line.
		if q[i] == '#' {
			for i < len(q) && q[i] != '\n' {
				i++
			}
			continue
		}
		// Block string: """ ... """
		if i+2 < len(q) && q[i] == '"' && q[i+1] == '"' && q[i+2] == '"' {
			i += 3
			for i+2 < len(q) {
				if q[i] == '"' && q[i+1] == '"' && q[i+2] == '"' {
					i += 3
					break
				}
				i++
			}
			continue
		}
		// Regular string: skip escaped chars.
		if q[i] == '"' {
			i++
			for i < len(q) && q[i] != '"' {
				if q[i] == '\\' {
					i++
				}
				i++
			}
			i++ // closing "
			continue
		}
		// Scan for operation keywords.
		matched := false
		for _, kw := range []string{"query", "mutation", "subscription"} {
			if strings.HasPrefix(q[i:], kw) {
				before := prevNonSpaceIndex(q, i-1)
				after := nextNonSpaceIndex(q, i+len(kw))
				// Count only explicit operation definitions, not field names or aliases
				// inside a selection set. Valid operation keywords appear at the start
				// of the document or after a completed operation body.
				beforeOK := before < 0 || q[before] == '}'
				afterOK := after < len(q) && (q[after] == '{' || q[after] == '@' || isIdentRune(q[after]))
				if beforeOK && afterOK {
					count++
					if count > 1 {
						return true
					}
					i += len(kw)
					matched = true
					break
				}
			}
		}
		if !matched {
			i++
		}
	}
	return false
}

func prevNonSpaceIndex(q string, i int) int {
	for i >= 0 {
		if q[i] != ' ' && q[i] != '\t' && q[i] != '\n' && q[i] != '\r' && q[i] != ',' {
			return i
		}
		i--
	}
	return -1
}

func nextNonSpaceIndex(q string, i int) int {
	for i < len(q) {
		if q[i] != ' ' && q[i] != '\t' && q[i] != '\n' && q[i] != '\r' && q[i] != ',' {
			return i
		}
		i++
	}
	return len(q)
}

// isIdentRune reports whether b is a valid GraphQL identifier continuation byte.
func isIdentRune(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func hasGraphQLErrors(payload map[string]any) bool {
	errors, ok := payload["errors"]
	if !ok || errors == nil {
		return false
	}
	items, ok := errors.([]any)
	if !ok {
		return true
	}
	return len(items) > 0
}

func toolCallError(msg string) map[string]any {
	return map[string]any{"success": false, "error": msg}
}
