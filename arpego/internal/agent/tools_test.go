package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
)

// sessionWithConfig builds a minimal Session for unit tests that only call
// executeLinearGraphQL. The client field is nil — only valid when not calling
// handleLinearGraphQL (which needs s.client.Send).
func sessionWithConfig(cfg config.Config) *Session {
	return &Session{cfg: cfg}
}

func linearCfg(endpoint, apiKey string) config.Config {
	return config.New(map[string]any{
		"tracker": map[string]any{
			"kind":     "linear",
			"endpoint": endpoint,
			"api_key":  apiKey,
		},
	})
}

func toolMsg(args map[string]any) Response {
	return Response{Params: map[string]any{"arguments": args}}
}

// --- input validation ---

func TestExecuteLinearGraphQLRejectsEmptyQuery(t *testing.T) {
	s := sessionWithConfig(linearCfg("https://linear.invalid", "key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{"query": ""}))
	assertToolCallError(t, result, "query must be a non-empty string")
}

func TestExecuteLinearGraphQLRejectsWhitespaceQuery(t *testing.T) {
	s := sessionWithConfig(linearCfg("https://linear.invalid", "key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{"query": "   "}))
	assertToolCallError(t, result, "query must be a non-empty string")
}

func TestExecuteLinearGraphQLRejectsMultipleOperations(t *testing.T) {
	s := sessionWithConfig(linearCfg("https://linear.invalid", "key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query": "query Foo { id } query Bar { id }",
	}))
	assertToolCallError(t, result, "exactly one GraphQL operation")
}

func TestExecuteLinearGraphQLRejectsFragmentOnlyDocument(t *testing.T) {
	s := sessionWithConfig(linearCfg("https://linear.invalid", "key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query": "fragment ViewerFields on Viewer { id }",
	}))
	assertToolCallError(t, result, "exactly one GraphQL operation")
}

func TestExecuteLinearGraphQLRejectsCommentOnlyDocument(t *testing.T) {
	s := sessionWithConfig(linearCfg("https://linear.invalid", "key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query": "# just a comment\n# still no operation",
	}))
	assertToolCallError(t, result, "exactly one GraphQL operation")
}

func TestExecuteLinearGraphQLRejectsNonObjectVariables(t *testing.T) {
	s := sessionWithConfig(linearCfg("https://linear.invalid", "key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query":     "query Foo { id }",
		"variables": "not-an-object",
	}))
	assertToolCallError(t, result, "variables must be an object")
}

func TestExecuteLinearGraphQLRejectsMissingTrackerKind(t *testing.T) {
	s := sessionWithConfig(config.New(map[string]any{}))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query": "query { viewer { id } }",
	}))
	assertToolCallError(t, result, "linear tracker is not configured")
}

func TestExecuteLinearGraphQLRejectsEmptyAPIKey(t *testing.T) {
	s := sessionWithConfig(config.New(map[string]any{
		"tracker": map[string]any{"kind": "linear", "api_key": ""},
	}))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query": "query { viewer { id } }",
	}))
	assertToolCallError(t, result, "linear tracker is not configured")
}

// --- HTTP success and error paths ---

func TestExecuteLinearGraphQLSuccessPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "lin_test_key" {
			t.Errorf("wrong auth header: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"id":"user-1"}}}`))
	}))
	defer srv.Close()

	s := sessionWithConfig(linearCfg(srv.URL, "lin_test_key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query": "query { viewer { id } }",
	}))

	if success, _ := result["success"].(bool); !success {
		t.Fatalf("expected success=true, got: %#v", result)
	}
	data, _ := result["data"].(map[string]any)
	if data == nil {
		t.Fatalf("expected data field, got: %#v", result)
	}
}

func TestExecuteLinearGraphQLPreservesGraphQLErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"not found"}],"data":null}`))
	}))
	defer srv.Close()

	s := sessionWithConfig(linearCfg(srv.URL, "key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query": "query { viewer { id } }",
	}))

	if success, _ := result["success"].(bool); success {
		t.Fatalf("expected success=false for GraphQL errors, got: %#v", result)
	}
	if _, hasData := result["data"]; !hasData {
		t.Fatalf("expected data field preserved on GraphQL error, got: %#v", result)
	}
	// Must use 'data', not 'error', for GraphQL-level errors.
	if _, hasError := result["error"]; hasError {
		t.Fatalf("GraphQL error response should not have 'error' field, got: %#v", result)
	}
}

func TestExecuteLinearGraphQLEmptyErrorsArrayIsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[],"data":{"viewer":{"id":"user-1"}}}`))
	}))
	defer srv.Close()

	s := sessionWithConfig(linearCfg(srv.URL, "key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query": "query { viewer { id } }",
	}))

	if success, _ := result["success"].(bool); !success {
		t.Fatalf("expected success=true for empty errors array, got: %#v", result)
	}
	if _, hasError := result["error"]; hasError {
		t.Fatalf("expected no transport error for empty errors array, got: %#v", result)
	}
}

func TestExecuteLinearGraphQLTransportError(t *testing.T) {
	// Port 0 never accepts connections.
	s := sessionWithConfig(linearCfg("http://127.0.0.1:0", "key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query": "query { viewer { id } }",
	}))

	if success, _ := result["success"].(bool); success {
		t.Fatalf("expected success=false on transport error, got: %#v", result)
	}
	if msg, _ := result["error"].(string); msg == "" {
		t.Fatalf("expected non-empty error message on transport failure, got: %#v", result)
	}
}

func TestExecuteLinearGraphQLForwardsVariables(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"issue":{"id":"123"}}}`))
	}))
	defer srv.Close()

	s := sessionWithConfig(linearCfg(srv.URL, "key"))
	result := s.executeLinearGraphQL(context.Background(), toolMsg(map[string]any{
		"query":     "query GetIssue($id: ID!) { issue(id: $id) { id } }",
		"variables": map[string]any{"id": "123"},
	}))

	if success, _ := result["success"].(bool); !success {
		t.Fatalf("expected success=true, got: %#v", result)
	}
	vars, _ := gotBody["variables"].(map[string]any)
	if vars["id"] != "123" {
		t.Fatalf("variables not forwarded correctly: %#v", gotBody)
	}
}

// --- hasMultipleOperations ---

func TestHasMultipleOperations(t *testing.T) {
	cases := []struct {
		name string
		q    string
		want bool
	}{
		{"single query", "query Foo { id }", false},
		{"single mutation", "mutation CreateIssue { issueCreate { issue { id } } }", false},
		{"shorthand query", "{ viewer { id } }", false},
		{"anonymous then explicit operation", "{ viewer { id } } mutation Update { viewer { id } }", true},
		{"two anonymous operations", "{ viewer { id } } { teams { nodes { id } } }", true},
		{"two queries", "query Foo { id } query Bar { id }", true},
		{"query and mutation", "query Foo { id } mutation Bar { id }", true},
		{"keyword in string literal", `query Foo { description(text: "mutation inside") }`, false},
		{"keyword in line comment", "# mutation comment\nquery Foo { id }", false},
		{"two queries with leading comment", "# note\nquery Foo { id } query Bar { id }", true},
		{"two queries with inline comment", "query Foo { id } # note\nquery Bar { id }", true},
		{"queryCount field is not keyword", "query Foo { queryCount }", false},
		{"mutationResult field is not keyword", "mutation Foo { mutationResult { id } }", false},
		{"query alias is not extra operation", "query Viewer { query: viewer { id } }", false},
		{"mutation field is not extra operation", "query Viewer { mutation }", false},
		{"introspection sibling queryType field", "query { __schema { mutationType { name } queryType { name } } }", false},
		{"fragment body queryCount field", "query Q { viewer { ...F } } fragment F on Viewer { teams { nodes { id } } queryCount: id }", false},
		{"fragment only", "fragment ViewerFields on Viewer { id }", false},
		{"comment only", "# just a comment\n# still no operation", false},
		{"empty string", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasMultipleOperations(tc.q)
			if got != tc.want {
				t.Fatalf("hasMultipleOperations(%q) = %v, want %v", tc.q, got, tc.want)
			}
		})
	}
}

// --- helpers ---

func assertToolCallError(t *testing.T, result map[string]any, wantSubstr string) {
	t.Helper()
	if success, _ := result["success"].(bool); success {
		t.Fatalf("expected success=false, got: %#v", result)
	}
	msg, _ := result["error"].(string)
	if !strings.Contains(msg, wantSubstr) {
		t.Fatalf("expected error containing %q, got: %q", wantSubstr, msg)
	}
}
