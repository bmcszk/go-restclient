// Code in this file is migrated from test/client_execute_graphql.go (RunExecuteFile_GraphQL*).
package restclient_test

import (
	"encoding/json"
	"net/http"
	"testing"

	rc "github.com/bmcszk/go-restclient"
)

// graphqlResponse is the mocked server response body for GraphQL requests.
type graphqlResponse struct {
	Data   any            `json:"data,omitempty"`
	Errors []graphqlError `json:"errors,omitempty"`
}

// graphqlError mirrors the GraphQL error entry shape.
type graphqlError struct {
	Message   string `json:"message"`
	Locations []any  `json:"locations,omitempty"`
	Path      []any  `json:"path,omitempty"`
}

// TestExecuteFile_GraphQLBasicQuery tests basic GraphQL query execution.
func TestExecuteFile_GraphQLBasicQuery(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = writeGraphQLResponse(w, graphqlResponse{
				Data: map[string]any{
					"user": map[string]any{
						"id":    "123",
						"name":  "John Doe",
						"email": "john@example.com",
					},
				},
			})
		}).and().
		aTemplateFixture("graphql", "basic_query.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responseHeader("Content-Type", "application/json").and().
		capturedJSONFieldContains(0, "query GetUser", "query").and().
		capturedJSONFieldContains(0, `user(id: "123")`, "query").and().
		capturedJSONFieldContains(0, "{ id name email }", "query")
}

// TestExecuteFile_GraphQLQueryWithVariables tests GraphQL queries with variables.
func TestExecuteFile_GraphQLQueryWithVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = writeGraphQLResponse(w, graphqlResponse{
				Data: map[string]any{
					"user": map[string]any{
						"id":        "456",
						"name":      "Jane Smith",
						"email":     "jane@example.com",
						"createdAt": "2023-01-01T00:00:00Z",
					},
				},
			})
		}).and().
		aTemplateFixture("graphql", "query_with_variables.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient(rc.WithVars(map[string]any{"userId": "456"}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		capturedJSONFieldContains(0, "query GetUserById($userId: ID!)", "query").and().
		capturedJSONFieldIs(0, "456", "variables", "userId")
}

// TestExecuteFile_GraphQLMutation tests GraphQL mutation execution.
func TestExecuteFile_GraphQLMutation(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = writeGraphQLResponse(w, graphqlResponse{
				Data: map[string]any{
					"createUser": map[string]any{
						"id":    "789",
						"name":  "New User",
						"email": "newuser@example.com",
					},
				},
			})
		}).and().
		aTemplateFixture("graphql", "mutation.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient(rc.WithVars(map[string]any{
			"userName":  "New User",
			"userEmail": "newuser@example.com",
		}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		capturedJSONFieldContains(0, "mutation CreateUser", "query").and().
		capturedJSONFieldIs(0, "New User", "variables", "input", "name").and().
		capturedJSONFieldIs(0, "newuser@example.com", "variables", "input", "email")
}

// TestExecuteFile_GraphQLFragments tests GraphQL queries with fragments.
func TestExecuteFile_GraphQLFragments(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = writeGraphQLResponse(w, graphqlResponse{
				Data: map[string]any{
					"users": []map[string]any{
						{"id": "1", "name": "User 1", "email": "user1@example.com",
							"createdAt": "2023-01-01T00:00:00Z"},
						{"id": "2", "name": "User 2", "email": "user2@example.com",
							"createdAt": "2023-01-02T00:00:00Z"},
					},
					"activeUsers": []map[string]any{
						{"id": "1", "name": "User 1", "email": "user1@example.com",
							"createdAt": "2023-01-01T00:00:00Z"},
					},
				},
			})
		}).and().
		aTemplateFixture("graphql", "fragments.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		capturedJSONFieldContains(0, "fragment UserInfo on User", "query").and().
		capturedJSONFieldContains(0, "...UserInfo", "query").and().
		capturedJSONFieldContains(0, "query GetUsers", "query")
}

// TestExecuteFile_GraphQLIntrospection tests GraphQL introspection queries.
func TestExecuteFile_GraphQLIntrospection(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = writeGraphQLResponse(w, graphqlResponse{
				Data: map[string]any{
					"__schema": map[string]any{
						"queryType":        map[string]any{"name": "Query"},
						"mutationType":     map[string]any{"name": "Mutation"},
						"subscriptionType": nil,
						"types":            []any{},
					},
				},
			})
		}).and().
		aTemplateFixture("graphql", "introspection.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		capturedJSONFieldContains(0, "query IntrospectionQuery", "query").and().
		capturedJSONFieldContains(0, "__schema", "query").and().
		capturedJSONFieldContains(0, "queryType", "query")
}

// TestExecuteFile_GraphQLErrorHandling tests GraphQL error response handling.
func TestExecuteFile_GraphQLErrorHandling(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK) // GraphQL errors are still HTTP 200
			_ = writeGraphQLResponse(w, graphqlResponse{
				Errors: []graphqlError{
					{
						Message: "Cannot query field 'nonExistentField' on type 'Query'",
						Path:    []any{"nonExistentField"},
					},
				},
			})
		}).and().
		aTemplateFixture("graphql", "error_handling.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		capturedJSONFieldContains(0, "query InvalidQuery", "query").and().
		capturedJSONFieldContains(0, "nonExistentField", "query").and().
		responseContains("nonExistentField").and().
		responseContains("\"errors\"")
}

// TestExecuteFile_GraphQLBatchQueries tests GraphQL batch query execution.
func TestExecuteFile_GraphQLBatchQueries(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = writeGraphQLResponse(w, []graphqlResponse{
				{
					Data: map[string]any{
						"user": map[string]any{"id": "123", "name": "John Doe"},
					},
				},
				{
					Data: map[string]any{
						"posts": []map[string]any{
							{"id": "1", "title": "Post 1"},
							{"id": "2", "title": "Post 2"},
						},
					},
				},
			})
		}).and().
		aTemplateFixture("graphql", "batch_queries.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responseHeader("Content-Type", "application/json").and().
		capturedBodyContains(0, "query GetUser").and().
		capturedBodyContains(0, "query GetPosts").and().
		capturedBodyContains(0, `"id": "123"`).and().
		capturedBodyMatchesRegexp(0, `(?s)\[.*query GetPosts.*\]`)
}

// writeGraphQLResponse encodes v as JSON into the response writer.
func writeGraphQLResponse(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}
