package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_InPlace_SimpleVariableInURL(t *testing.T) {
	// Given
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json") // Ensure content type for expected.hresp
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`)) // Match expected.hresp
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/simple_variable_in_url/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/simple_variable_in_url/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable for {{test_server_url}} in request.http
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	require.NoError(t, resp.Error, "Response error should be nil")

	// Parse expected response from .hresp file
	expectedHeaders, expectedBodyStr, pErr := parseHrespBody(expectedHrespPath)
	require.NoError(t, pErr, "Failed to parse .hresp file: %s", expectedHrespPath)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	for key, expectedValue := range expectedHeaders {
		assert.Equal(t, expectedValue[0], resp.Headers.Get(key), fmt.Sprintf("Header %s mismatch", key))
	}

	if expectedBodyStr != "" {
		assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
	} else {
		assert.Empty(t, string(resp.Body), "Response body should be empty")
	}

	// Verify the server received the request with the substituted URL
	// request.http defines:
	// @hostname = {{test_server_url}}
	// @path_segment = /api/v1/items
	// GET {{hostname}}{{path_segment}}/123
	// So, expected path is /api/v1/items/123
	expectedServerPath := "/api/v1/items/123"
	assert.Equal(t, expectedServerPath, capturedPath, "Captured path by server mismatch")

	// Verify that the FileVariables in the parsed file reflect the raw definitions
	// This part checks how the .http file itself was parsed, not the final runtime values.
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["hostname"], "Parsed file variable 'hostname' mismatch")
	assert.Equal(t, "/api/v1/items", parsedFile.FileVariables["path_segment"], "Parsed file variable 'path_segment' mismatch")
}

func TestExecuteFile_InPlace_VariableInHeader(t *testing.T) {
	// Given
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json") // Match expected.hresp
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`)) // Match expected.hresp
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/variable_in_header/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/variable_in_header/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable for {{test_server_url}} in request.http
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	require.NoError(t, resp.Error, "Response error should be nil")

	// Parse expected response from .hresp file
	expectedHeaders, expectedBodyStr, pErr := parseHrespBody(expectedHrespPath)
	require.NoError(t, pErr, "Failed to parse .hresp file: %s", expectedHrespPath)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	for key, expectedValue := range expectedHeaders {
		assert.Equal(t, expectedValue[0], resp.Headers.Get(key), fmt.Sprintf("Header %s mismatch", key))
	}

	if expectedBodyStr != "" {
		assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
	} else {
		assert.Empty(t, string(resp.Body), "Response body should be empty")
	}

	// Verify the server received the request with the substituted header
	// request.http defines:
	// @auth_token = Bearer_secret_token_123
	// GET {{test_server_url}}/checkheaders
	// Authorization: {{auth_token}}
	// User-Agent: test-client
	assert.Equal(t, "Bearer_secret_token_123", capturedHeaders.Get("Authorization"))
	assert.Equal(t, "test-client", capturedHeaders.Get("User-Agent")) // Ensure other headers are preserved

	// Verify ParsedFile.FileVariables
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "Bearer_secret_token_123", parsedFile.FileVariables["auth_token"])
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["test_server_url"]) // Check placeholder
}

func TestExecuteFile_InPlace_VariableInBody(t *testing.T) {
	// Given
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errRead error
		capturedBody, errRead = io.ReadAll(r.Body) // io.ReadAll is used here
		require.NoError(t, errRead)
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json") // Match expected.hresp
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"created"}`)) // Match expected.hresp
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/variable_in_body/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/variable_in_body/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable for {{test_server_url}} in request.http
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	require.NoError(t, resp.Error, "Response error should be nil")

	// Parse expected response from .hresp file
	expectedHeaders, expectedBodyStr, pErr := parseHrespBody(expectedHrespPath)
	require.NoError(t, pErr, "Failed to parse .hresp file: %s", expectedHrespPath)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	for key, expectedValue := range expectedHeaders {
		assert.Equal(t, expectedValue[0], resp.Headers.Get(key), fmt.Sprintf("Header %s mismatch", key))
	}

	if expectedBodyStr != "" {
		assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
	} else {
		assert.Empty(t, string(resp.Body), "Response body should be empty")
	}

	// Verify the server received the request with the substituted body
	expectedSentBodyJSON := `{
  "id": "SW1000",
  "name": "SuperWidget",
  "price": 49.99
}`
	assert.JSONEq(t, expectedSentBodyJSON, string(capturedBody), "Captured body by server mismatch")

	// Verify ParsedFile.FileVariables
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "SuperWidget", parsedFile.FileVariables["product_name"])
	assert.Equal(t, "SW1000", parsedFile.FileVariables["product_id"])
	assert.Equal(t, "49.99", parsedFile.FileVariables["product_price"])                 // Variables are stored as strings
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["test_server_url"]) // Check placeholder
}

func TestExecuteFile_InPlace_VariableDefinedByAnotherVariable(t *testing.T) {
	// Given
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String() // Captures path and query
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/variable_defined_by_another_variable/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/variable_defined_by_another_variable/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	client.SetProgrammaticVar("test_server_url", server.URL)

	// When
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	require.NoError(t, resp.Error, "Response error should be nil")

	expectedHeaders, expectedBodyStr, pErr := parseHrespBody(expectedHrespPath)
	require.NoError(t, pErr, "Failed to parse .hresp file: %s", expectedHrespPath)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	assert.Equal(t, expectedHeaders.Get("Content-Type"), resp.Headers.Get("Content-Type"), "Content-Type mismatch")

	if expectedBodyStr != "" {
		assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
	} else {
		assert.Empty(t, string(resp.Body), "Response body should be empty")
	}

	expectedPathAndQuery := fmt.Sprintf("/api/v2/items?check_base=%s&check_path=/api/v2/items", server.URL)
	assert.Equal(t, expectedPathAndQuery, capturedURL, "Captured URL by server mismatch")

	// Verify ParsedFile.FileVariables (should store raw definitions from the .http file)
	// request.http is:
	// @base_url = {{test_server_url}}
	// @path = /api/v2/items
	// @full_url = {{base_url}}{{path}}
	// GET {{full_url}}?check_base={{base_url}}&check_path={{path}}
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["base_url"])
	assert.Equal(t, "/api/v2/items", parsedFile.FileVariables["path"])
	assert.Equal(t, "{{base_url}}{{path}}", parsedFile.FileVariables["full_url"])
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["test_server_url"]) // Check placeholder
}

func TestExecuteFile_InPlace_VariablePrecedenceOverEnvironment(t *testing.T) {
	// Given: an .http file with an in-place variable and an environment variable with the same name
	var capturedURLPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path // We only care about the path part
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/variable_precedence_over_environment/request.http"
	// expectedHrespPath is not strictly needed for validation in this test as original only checks path
	// but good to have for consistency if we wanted to validate response status code etc.
	// expectedHrespPath := "testdata/execute_inplace_vars/variable_precedence_over_environment/expected.hresp"

	// The environment name matches the one in the .json filename
	envName := "testPrecedenceEnv"
	// The client will look for http-client.env.testPrecedenceEnv.json in the same dir as request.http
	client, err := NewClient(WithEnvironment(envName))
	require.NoError(t, err)

	// Set the mock server URL for the {{test_server_url}} variable in request.http
	// This is distinct from the 'host' variable defined in the .env.json file
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When: the .http file is executed
	_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the in-place variable from request.http should take precedence
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	// The request.http is:
	// @host = {{test_server_url}}
	// GET {{host}}/expected_path
	// So, {{host}} becomes server.URL, and path becomes /expected_path
	assert.Equal(t, "/expected_path", capturedURLPath, "The request path should match, indicating in-place var was used")

	// Additionally, verify that file variables were parsed correctly
	// client.loadEnvironment is unexported, so we can't directly check the loaded environment here
	// as it was attempted previously.
	// However, we can still check how parseRequestFile interprets the file variables.
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["host"], "File variable 'host' mismatch")
}

func TestExecuteFile_InPlace_VariableInCustomHeader(t *testing.T) {
	// Given: an .http file with an in-place variable used in a custom header
	var capturedHeaderValue string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaderValue = r.Header.Get("X-Custom-Header")
		// No specific content type or body needed for this test's expected.hresp
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/variable_in_custom_header/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/variable_in_custom_header/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When: the .http file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur, response should be validated, and the header should be substituted correctly
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	require.NoError(t, resp.Error, "Response error should be nil")

	// Parse expected response from .hresp file
	expectedHeaders, expectedBodyStr, pErr := parseHrespBody(expectedHrespPath)
	require.NoError(t, pErr, "Failed to parse .hresp file: %s", expectedHrespPath)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	for key, expectedValue := range expectedHeaders { // Should be empty for this .hresp
		assert.Equal(t, expectedValue[0], resp.Headers.Get(key), fmt.Sprintf("Header %s mismatch", key))
	}

	if expectedBodyStr != "" { // Should be empty for this .hresp
		assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
	} else {
		assert.Empty(t, string(resp.Body), "Response body should be empty")
	}

	assert.Equal(t, "secret-token", capturedHeaderValue, "The X-Custom-Header should be correctly substituted")

	// Verify ParsedFile.FileVariables
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "secret-token", parsedFile.FileVariables["my_header_value"])
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["test_server_url"]) // Check placeholder
}

func TestExecuteFile_InPlace_VariableSubstitutionInBody(t *testing.T) {
	// Given: an .http file with an in-place variable used in a JSON request body
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		// The expected.hresp is minimal (200 OK), so the server doesn't need to send a specific body or Content-Type.
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/variable_substitution_in_body/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/variable_substitution_in_body/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable for {{test_server_url}} in request.http
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When: the .http file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur, response should be validated, and the body should be substituted correctly
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	require.NoError(t, resp.Error, "Response error should be nil")

	// Parse expected response from .hresp file
	expectedHeaders, expectedBodyStr, pErr := parseHrespBody(expectedHrespPath)
	require.NoError(t, pErr, "Failed to parse .hresp file: %s", expectedHrespPath)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	// Assert headers from .hresp (should be none for the minimal .hresp)
	for key, expectedValue := range expectedHeaders {
		assert.Equal(t, expectedValue[0], resp.Headers.Get(key), fmt.Sprintf("Header %s mismatch", key))
	}
	// Assert body from .hresp (should be empty for the minimal .hresp)
	if expectedBodyStr != "" {
		assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
	} else {
		assert.Empty(t, string(resp.Body), "Response body should be empty")
	}

	// Assert the captured body by the server (this is the main assertion for this test)
	assert.JSONEq(t, `{"id": "user123", "status": "active"}`, string(capturedBody), "The request body captured by the server should be correctly substituted")

	// Verify ParsedFile.FileVariables
	// This checks how the .http file itself was parsed.
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "user123", parsedFile.FileVariables["user_id"], "Parsed file variable 'user_id' mismatch")
}
