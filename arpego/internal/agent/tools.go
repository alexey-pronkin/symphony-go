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

	// Reject documents that do not contain exactly one operation; Linear will also
	// reject them, but catching it early gives a cleaner error message.
	if countGraphQLOperations(rawQuery) != 1 {
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

// countGraphQLOperations returns the number of top-level GraphQL operations in q.
// It strips comments and string literals first, then counts only top-level

// anonymous operations (`{ ... }`) and explicit operation definitions.
// This is defense-in-depth for the extension contract; Linear remains the

// authoritative parser.
func countGraphQLOperations(q string) int {
	sanitized := sanitizeGraphQLForOperationScan(q)
	count := 0
	depth := 0
	pendingExplicit := false
	pendingFragment := false
	for i := 0; i < len(sanitized); i++ {
		ch := sanitized[i]
		before := prevNonSpaceIndex(sanitized, i-1)
		if depth > 0 {
			switch ch {
			case '{':
				depth++
			case '}':
				depth--
			}
			continue
		}
		if pendingExplicit || pendingFragment {
			if ch == '{' {
				depth = 1
				pendingExplicit = false
				pendingFragment = false
			}
			continue
		}
		if ch == '{' && (before < 0 || sanitized[before] == '}') {
			count++
			depth = 1
			continue
		}
		if strings.HasPrefix(sanitized[i:], "fragment") {
			after := nextNonSpaceIndex(sanitized, i+len("fragment"))
			beforeOK := before < 0 || sanitized[before] == '}'
			afterOK := after < len(sanitized) && isIdentRune(sanitized[after])
			if beforeOK && afterOK {
				pendingFragment = true
				i += len("fragment") - 1
				continue
			}
		}
		for _, kw := range []string{"query", "mutation", "subscription"} {
			if strings.HasPrefix(sanitized[i:], kw) {
				after := nextNonSpaceIndex(sanitized, i+len(kw))
				beforeOK := before < 0 || sanitized[before] == '}'
				afterOK := after < len(sanitized) && (sanitized[after] == '{' || sanitized[after] == '@' || isIdentRune(sanitized[after]))
				if beforeOK && afterOK {
					count++
					pendingExplicit = true
					i += len(kw) - 1
					break
				}
			}
		}
	}
	return count
}

func hasMultipleOperations(q string) bool {
	return countGraphQLOperations(q) > 1
}

func sanitizeGraphQLForOperationScan(q string) string {
	buf := []byte(q)
	for i := 0; i < len(buf); i++ {
		if buf[i] == '#' {
			for i < len(buf) && buf[i] != '\n' {
				buf[i] = ' '
				i++
			}
			if i < len(buf) {
				buf[i] = ' '
			}
			continue
		}
		if i+2 < len(buf) && buf[i] == '"' && buf[i+1] == '"' && buf[i+2] == '"' {
			buf[i], buf[i+1], buf[i+2] = ' ', ' ', ' '
			i += 3
			for i+2 < len(buf) {
				if buf[i] == '"' && buf[i+1] == '"' && buf[i+2] == '"' {
					buf[i], buf[i+1], buf[i+2] = ' ', ' ', ' '
					i += 2
					break
				}
				buf[i] = ' '
				i++
			}
			continue
		}
		if buf[i] == '"' {
			buf[i] = ' '
			i++
			for i < len(buf) && buf[i] != '"' {
				if buf[i] == '\\' {
					buf[i] = ' '
					i++
					if i < len(buf) {
						buf[i] = ' '
					}
					continue
				}
				buf[i] = ' '
				i++
			}
			if i < len(buf) {
				buf[i] = ' '
			}
		}
	}
	return string(buf)
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
