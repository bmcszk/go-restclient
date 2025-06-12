package test

import (
	rc "github.com/bmcszk/go-restclient"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GraphQL test data structure for mock responses
type graphqlResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []struct {
		Message   string `json:"message"`
		Locations []struct {
			Line   int `json:"line"`
			Column int `json:"column"`
		} `json:"locations,omitempty"`
		Path []interface{} `json:"path,omitempty"`
	} `json:"errors,omitempty"`
}

// GraphQL request structure for parsing incoming requests
type graphqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// RunExecuteFile_GraphQLBasicQuery tests basic GraphQL query execution
func RunExecuteFile_GraphQLBasicQuery(t *testing.T) {
	t.Helper()
	// Given
	var interceptedRequest graphqlRequest
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		json.Unmarshal(bodyBytes, &interceptedRequest)
		
		// Mock GraphQL response
		response := graphqlResponse{
			Data: map[string]interface{}{
				"user": map[string]interface{}{
					"id":    "123",
					"name":  "John Doe",
					"email": "john@example.com",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/graphql/basic_query.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"))
	
	// Validate GraphQL request structure
	assert.Contains(t, interceptedRequest.Query, "query GetUser")
	assert.Contains(t, interceptedRequest.Query, "user(id: \"123\")")
	assert.Contains(t, interceptedRequest.Query, "{ id name email }")
}

// RunExecuteFile_GraphQLQueryWithVariables tests GraphQL queries with variables
func RunExecuteFile_GraphQLQueryWithVariables(t *testing.T) {
	t.Helper()
	// Given
	var interceptedRequest graphqlRequest
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		json.Unmarshal(bodyBytes, &interceptedRequest)
		
		// Mock GraphQL response
		response := graphqlResponse{
			Data: map[string]interface{}{
				"user": map[string]interface{}{
					"id":        "456",
					"name":      "Jane Smith",
					"email":     "jane@example.com",
					"createdAt": "2023-01-01T00:00:00Z",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client, _ := rc.NewClient(rc.WithVars(map[string]any{
		"userId": "456",
	}))
	requestFilePath := createTestFileFromTemplate(t, "test/data/graphql/query_with_variables.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Validate GraphQL request structure
	assert.Contains(t, interceptedRequest.Query, "query GetUserById($userId: ID!)")
	assert.NotNil(t, interceptedRequest.Variables)
	assert.Equal(t, "456", interceptedRequest.Variables["userId"])
}

// RunExecuteFile_GraphQLMutation tests GraphQL mutation execution
func RunExecuteFile_GraphQLMutation(t *testing.T) {
	t.Helper()
	// Given
	var interceptedRequest graphqlRequest
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		json.Unmarshal(bodyBytes, &interceptedRequest)
		
		// Mock GraphQL mutation response
		response := graphqlResponse{
			Data: map[string]interface{}{
				"createUser": map[string]interface{}{
					"id":    "789",
					"name":  "New User",
					"email": "newuser@example.com",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client, _ := rc.NewClient(rc.WithVars(map[string]any{
		"userName":  "New User",
		"userEmail": "newuser@example.com",
	}))
	requestFilePath := createTestFileFromTemplate(t, "test/data/graphql/mutation.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Validate GraphQL mutation structure
	assert.Contains(t, interceptedRequest.Query, "mutation CreateUser")
	assert.NotNil(t, interceptedRequest.Variables)
	
	// Check that variables were properly substituted
	input, ok := interceptedRequest.Variables["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "New User", input["name"])
	assert.Equal(t, "newuser@example.com", input["email"])
}

// RunExecuteFile_GraphQLFragments tests GraphQL queries with fragments
func RunExecuteFile_GraphQLFragments(t *testing.T) {
	t.Helper()
	// Given
	var interceptedRequest graphqlRequest
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		json.Unmarshal(bodyBytes, &interceptedRequest)
		
		// Mock GraphQL response with fragments
		response := graphqlResponse{
			Data: map[string]interface{}{
				"users": []map[string]interface{}{
					{"id": "1", "name": "User 1", "email": "user1@example.com", "createdAt": "2023-01-01T00:00:00Z"},
					{"id": "2", "name": "User 2", "email": "user2@example.com", "createdAt": "2023-01-02T00:00:00Z"},
				},
				"activeUsers": []map[string]interface{}{
					{"id": "1", "name": "User 1", "email": "user1@example.com", "createdAt": "2023-01-01T00:00:00Z"},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/graphql/fragments.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Validate GraphQL fragment structure
	assert.Contains(t, interceptedRequest.Query, "fragment UserInfo on User")
	assert.Contains(t, interceptedRequest.Query, "...UserInfo")
	assert.Contains(t, interceptedRequest.Query, "query GetUsers")
}

// RunExecuteFile_GraphQLIntrospection tests GraphQL introspection queries
func RunExecuteFile_GraphQLIntrospection(t *testing.T) {
	t.Helper()
	// Given
	var interceptedRequest graphqlRequest
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		json.Unmarshal(bodyBytes, &interceptedRequest)
		
		// Mock minimal introspection response
		response := graphqlResponse{
			Data: map[string]interface{}{
				"__schema": map[string]interface{}{
					"queryType":        map[string]interface{}{"name": "Query"},
					"mutationType":     map[string]interface{}{"name": "Mutation"},
					"subscriptionType": nil,
					"types":            []interface{}{},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/graphql/introspection.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Validate introspection query structure
	assert.Contains(t, interceptedRequest.Query, "query IntrospectionQuery")
	assert.Contains(t, interceptedRequest.Query, "__schema")
	assert.Contains(t, interceptedRequest.Query, "queryType")
}

// RunExecuteFile_GraphQLErrorHandling tests GraphQL error response handling
func RunExecuteFile_GraphQLErrorHandling(t *testing.T) {
	t.Helper()
	// Given
	var interceptedRequest graphqlRequest
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		json.Unmarshal(bodyBytes, &interceptedRequest)
		
		// Mock GraphQL error response
		response := graphqlResponse{
			Errors: []struct {
				Message   string `json:"message"`
				Locations []struct {
					Line   int `json:"line"`
					Column int `json:"column"`
				} `json:"locations,omitempty"`
				Path []interface{} `json:"path,omitempty"`
			}{
				{
					Message: "Cannot query field 'nonExistentField' on type 'Query'",
					Locations: []struct {
						Line   int `json:"line"`
						Column int `json:"column"`
					}{
						{Line: 1, Column: 15},
					},
					Path: []interface{}{"nonExistentField"},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // GraphQL errors are still HTTP 200
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/graphql/error_handling.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode) // GraphQL errors are HTTP 200
	
	// Validate GraphQL error query structure
	assert.Contains(t, interceptedRequest.Query, "query InvalidQuery")
	assert.Contains(t, interceptedRequest.Query, "nonExistentField")
	
	// Parse and validate error response structure
	var responseBody graphqlResponse
	err = json.Unmarshal(resp.Body, &responseBody)
	require.NoError(t, err)
	assert.NotEmpty(t, responseBody.Errors)
	assert.Contains(t, responseBody.Errors[0].Message, "nonExistentField")
}

// RunExecuteFile_GraphQLBatchQueries tests GraphQL batch query execution
func RunExecuteFile_GraphQLBatchQueries(t *testing.T) {
	t.Helper()
	// Given
	var interceptedRequests []graphqlRequest
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		
		// Try to parse as batch (array) first, then as single request
		var batchRequest []graphqlRequest
		if err := json.Unmarshal(bodyBytes, &batchRequest); err == nil {
			interceptedRequests = batchRequest
		} else {
			var singleRequest graphqlRequest
			json.Unmarshal(bodyBytes, &singleRequest)
			interceptedRequests = []graphqlRequest{singleRequest}
		}
		
		// Mock batch response
		responses := []graphqlResponse{
			{
				Data: map[string]interface{}{
					"user": map[string]interface{}{
						"id":   "123",
						"name": "John Doe",
					},
				},
			},
			{
				Data: map[string]interface{}{
					"posts": []map[string]interface{}{
						{"id": "1", "title": "Post 1"},
						{"id": "2", "title": "Post 2"},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(responses)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/graphql/batch_queries.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Validate batch request structure
	require.Len(t, interceptedRequests, 2)
	assert.Contains(t, interceptedRequests[0].Query, "query GetUser")
	assert.Equal(t, "123", interceptedRequests[0].Variables["id"])
	assert.Contains(t, interceptedRequests[1].Query, "query GetPosts")
}