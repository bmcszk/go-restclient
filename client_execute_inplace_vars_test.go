package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
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
	// Programmatic variables are resolved during parsing and don't remain as placeholders
	assert.Equal(t, server.URL, parsedFile.FileVariables["test_server_url"])
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
	assert.Equal(t, "49.99", parsedFile.FileVariables["product_price"]) // Variables are stored as strings
	// Programmatic variables are resolved during parsing and stored with their actual values
	assert.Equal(t, server.URL, parsedFile.FileVariables["test_server_url"])
}

func TestExecuteFile_InPlace_VariableDefinedByAnotherVariable(t *testing.T) {
	// Given
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String() // Captures path and query
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return response matching expected.hresp
		_, _ = w.Write([]byte(`{"id":"123", "status":"ok"}`))
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

	// The actual request URL in the file is simply GET {{full_url}}
	// where full_url resolves to {{base_url}}{{path}}/123 which is {{test_server_url}}/users/123
	expectedPathAndQuery := "/users/123"
	assert.Equal(t, expectedPathAndQuery, capturedURL, "Captured URL by server mismatch")

	// Verify ParsedFile.FileVariables - should store defined variables from the .http file
	// with programmatic variables fully resolved
	// From request.http:
	// @base_url = {{test_server_url}}
	// @path = /users
	// @full_url = {{base_url}}{{path}}/123
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	// base_url references the programmatic variable test_server_url, so it should be resolved
	assert.Equal(t, server.URL, parsedFile.FileVariables["base_url"])
	assert.Equal(t, "/users", parsedFile.FileVariables["path"])
	assert.Equal(t, fmt.Sprintf("%s/users/123", server.URL), parsedFile.FileVariables["full_url"])
	// Programmatic variables are resolved during parsing
	assert.Equal(t, server.URL, parsedFile.FileVariables["test_server_url"]) // Resolved programmatic var
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

func TestExecuteFile_InPlace_VariableDefinedBySystemVariable(t *testing.T) {
	// Given: an .http file with an in-place variable defined by a system variable {{$uuid}}
	var capturedURLPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		// The expected.hresp is minimal (200 OK)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/inplace_variable_defined_by_system_variable/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/inplace_variable_defined_by_system_variable/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When: the .http file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur, response should be validated, and the path should contain a resolved UUID
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

	// Assertions for the captured URL path
	require.NotEmpty(t, capturedURLPath, "Captured URL path should not be empty")
	pathSegments := strings.Split(strings.Trim(capturedURLPath, "/"), "/")
	require.Len(t, pathSegments, 2, "URL path should have two segments")
	assert.Len(t, pathSegments[0], 36, "The first path segment (resolved UUID) should be 36 characters long") // UUIDs are 36 chars
	assert.Equal(t, "resource", pathSegments[1], "The second path segment should be 'resource'")
	assert.NotEqual(t, "{{$uuid}}", pathSegments[0], "The UUID part should not be the literal system variable")
	assert.NotEqual(t, "{{my_request_id}}", pathSegments[0], "The UUID part should not be the literal in-place variable")

	// Verify ParsedFile.FileVariables
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "{{$uuid}}", parsedFile.FileVariables["my_request_id"], "Parsed file variable 'my_request_id' mismatch")
}

func TestExecuteFile_InPlace_VariableDefinedByOsEnvVariable(t *testing.T) {
	// Given: an OS environment variable and an .http file with an in-place variable defined by it
	const testEnvVarName = "TEST_USER_HOME_INPLACE"
	const testEnvVarValue = "/testhome/userdir" // This value starts with a slash
	t.Setenv(testEnvVarName, testEnvVarValue)

	// Debug: Check if t.Setenv is working as expected in the test goroutine
	val, ok := os.LookupEnv(testEnvVarName)
	require.True(t, ok, "os.LookupEnv should find the var set by t.Setenv")
	require.Equal(t, testEnvVarValue, val, "os.LookupEnv should return the correct value set by t.Setenv")

	var capturedURLPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/inplace_variable_defined_by_os_env_variable/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/inplace_variable_defined_by_os_env_variable/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable for {{test_server_url}} in request.http
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When: the .http file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the path should contain the resolved OS env variable
	require.NoError(t, execErr, "ExecuteFile should not return an error for in-place OS env var")
	require.Len(t, responses, 1, "Should have one result for in-place OS env var")
	resp := responses[0]
	require.Nil(t, resp.Error, "Request execution error should be nil for in-place OS env var")

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

	// capturedURLPath should be "/testhome/userdir/files"
	assert.Equal(t, testEnvVarValue+"/files", capturedURLPath, "The URL path should be correctly substituted with the OS environment variable via in-place var")

	// Verify ParsedFile.FileVariables
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "{{$processEnv TEST_USER_HOME_INPLACE}}", parsedFile.FileVariables["my_home_dir"], "Parsed file variable 'my_home_dir' mismatch")
}

func TestExecuteFile_InPlace_VariableInAuthHeader(t *testing.T) {
	// Given: an .http file with an in-place variable used in an X-Auth-Token header
	const headerKey = "X-Auth-Token"
	const headerValue = "secret-token-12345"

	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK) // Minimal response, as per expected.hresp
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/inplace_variable_in_auth_header/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/inplace_variable_in_auth_header/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable for {{test_server_url}} in request.http
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When: the .http file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur, response should be validated, and the header should be correctly substituted
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	require.NoError(t, resp.Error, "Response error should be nil")

	// Parse expected response from .hresp file
	expectedRespHeaders, expectedBodyStr, pErr := parseHrespBody(expectedHrespPath)
	require.NoError(t, pErr, "Failed to parse .hresp file: %s", expectedHrespPath)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	// Assert headers from .hresp (should be none for the minimal .hresp)
	for key, expectedVal := range expectedRespHeaders {
		assert.Equal(t, expectedVal[0], resp.Headers.Get(key), fmt.Sprintf("Header %s mismatch", key))
	}
	// Assert body from .hresp (should be empty for the minimal .hresp)
	if expectedBodyStr != "" {
		assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
	} else {
		assert.Empty(t, string(resp.Body), "Response body should be empty")
	}

	// Main assertion: check the captured header
	assert.Equal(t, headerValue, capturedHeaders.Get(headerKey), "The X-Auth-Token header should be correctly substituted")

	// Verify ParsedFile.FileVariables
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, headerValue, parsedFile.FileVariables["my_token"], "Parsed file variable 'my_token' mismatch")
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["test_server_url"], "Parsed file variable 'test_server_url' (placeholder) mismatch")
}

func TestExecuteFile_InPlace_VariableInJsonRequestBody(t *testing.T) {
	// Given: an .http file with an in-place variable used in the JSON request body
	const userIdValue = "user-from-var-456" // This value is defined in request.http
	const expectedSentBody = `{"id": "user-from-var-456", "status": "pending"}`

	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errReadBody error
		capturedBody, errReadBody = io.ReadAll(r.Body)
		require.NoError(t, errReadBody)
		w.WriteHeader(http.StatusOK) // Minimal response for the server
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/inplace_variable_in_json_request_body/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/inplace_variable_in_json_request_body/expected.hresp" // Minimal hresp

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable for {{test_server_url}} in request.http
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When: the .http file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur, response should be validated, and the sent body should be correctly substituted
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	require.NoError(t, resp.Error, "Response error should be nil")

	// Parse expected response from .hresp file (minimal for this test)
	expectedRespHeaders, expectedBodyStr, pErr := parseHrespBody(expectedHrespPath)
	require.NoError(t, pErr, "Failed to parse .hresp file: %s", expectedHrespPath)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	// Assert headers from .hresp (should be none for the minimal .hresp)
	for key, expectedVal := range expectedRespHeaders {
		assert.Equal(t, expectedVal[0], resp.Headers.Get(key), fmt.Sprintf("Header %s mismatch", key))
	}
	// Assert body from .hresp (should be empty for the minimal .hresp)
	if expectedBodyStr != "" {
		assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
	} else {
		assert.Empty(t, string(resp.Body), "Response body should be empty")
	}

	// Main assertion: check the captured request body sent to the server
	assert.JSONEq(t, expectedSentBody, string(capturedBody), "The request body sent to the server should be correctly substituted")

	// Verify ParsedFile.FileVariables
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, userIdValue, parsedFile.FileVariables["my_user_id"], "Parsed file variable 'my_user_id' mismatch")
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["test_server_url"], "Parsed file variable 'test_server_url' (placeholder) mismatch")
}

func TestExecuteFile_InPlace_VariableDefinedByAnotherInPlaceVariable(t *testing.T) {
	// Given: an .http file with an in-place variable defined by another in-place variable
	const basePathValue = "/api/v1"   // Defined in request.http
	const resourcePathValue = "items" // Defined in request.http
	const expectedURLPath = "/api/v1/items/123"
	var capturedURLPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		w.WriteHeader(http.StatusOK) // Minimal response
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/inplace_variable_defined_by_another_inplace_variable/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/inplace_variable_defined_by_another_inplace_variable/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable for {{test_server_url}} in request.http
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When: the .http file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur, response should be validated, and the URL path should be correctly substituted
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	require.NoError(t, resp.Error, "Response error should be nil")

	// Parse expected response from .hresp file (minimal for this test)
	expectedRespHeaders, expectedBodyStr, pErr := parseHrespBody(expectedHrespPath)
	require.NoError(t, pErr, "Failed to parse .hresp file: %s", expectedHrespPath)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	// Assert headers from .hresp (should be none for the minimal .hresp)
	for key, expectedVal := range expectedRespHeaders {
		assert.Equal(t, expectedVal[0], resp.Headers.Get(key), fmt.Sprintf("Header %s mismatch", key))
	}
	// Assert body from .hresp (should be empty for the minimal .hresp)
	if expectedBodyStr != "" {
		assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
	} else {
		assert.Empty(t, string(resp.Body), "Response body should be empty")
	}

	// Main assertion: check the captured URL path
	assert.Equal(t, expectedURLPath, capturedURLPath, "The URL path should be correctly substituted with nested in-place variables")

	// Verify ParsedFile.FileVariables
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, basePathValue, parsedFile.FileVariables["base_path"], "Parsed file variable 'base_path' mismatch")
	assert.Equal(t, resourcePathValue, parsedFile.FileVariables["resource"], "Parsed file variable 'resource' mismatch")
	assert.Equal(t, "{{base_path}}/{{resource}}/123", parsedFile.FileVariables["full_url_segment"], "Parsed file variable 'full_url_segment' mismatch")
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["test_server_url"], "Parsed file variable 'test_server_url' (placeholder) mismatch")
}

func TestExecuteFile_InPlace_VariableDefinedByDotEnvOsVariable(t *testing.T) {
	// Given: an .http file with an in-place variable defined by an OS environment variable using {{$env.VAR_NAME}}
	const testEnvVarName = "MY_CONFIG_PATH_TEST_DOT_ENV"
	const testEnvVarValue = "/usr/local/appconfig_dotenv"
	var capturedURLPath string

	// Set the OS environment variable that the .http file will reference via {{$env.MY_CONFIG_PATH_TEST_DOT_ENV}}
	t.Setenv(testEnvVarName, testEnvVarValue)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		w.WriteHeader(http.StatusOK) // Minimal response
	}))
	defer server.Close()

	requestFilePath := "testdata/execute_inplace_vars/inplace_variable_defined_by_dot_env_os_variable/request.http"
	expectedHrespPath := "testdata/execute_inplace_vars/inplace_variable_defined_by_dot_env_os_variable/expected.hresp"

	client, err := NewClient()
	require.NoError(t, err)

	// Set the mock server URL as a programmatic variable for {{test_server_url}} in request.http
	client.SetProgrammaticVar("test_server_url", server.URL)

	// When: the .http file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur, response should be validated, and the URL path should be correctly substituted
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	require.NoError(t, resp.Error, "Response error should be nil")

	// Parse expected response from .hresp file (minimal for this test)
	expectedRespHeaders, expectedBodyStr, pErr := parseHrespBody(expectedHrespPath)
	require.NoError(t, pErr, "Failed to parse .hresp file: %s", expectedHrespPath)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	// Assert headers from .hresp (should be none for the minimal .hresp)
	for key, expectedVal := range expectedRespHeaders {
		assert.Equal(t, expectedVal[0], resp.Headers.Get(key), fmt.Sprintf("Header %s mismatch", key))
	}
	// Assert body from .hresp (should be empty for the minimal .hresp)
	if expectedBodyStr != "" {
		assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
	} else {
		assert.Empty(t, string(resp.Body), "Response body should be empty")
	}

	// Main assertion: check the captured URL path
	expectedPath := testEnvVarValue + "/data" // As per original test logic
	assert.Equal(t, expectedPath, capturedURLPath, "The URL path should be correctly substituted with the OS environment variable via {{$env.VAR_NAME}} in-place var")

	// Verify ParsedFile.FileVariables
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "{{$env.MY_CONFIG_PATH_TEST_DOT_ENV}}", parsedFile.FileVariables["my_path_from_env"], "Parsed file variable 'my_path_from_env' mismatch")
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["test_server_url"], "Parsed file variable 'test_server_url' (placeholder) mismatch")
}

func TestExecuteFile_InPlace_Malformed_NameOnlyNoEqualsNoValue(t *testing.T) {
	// Given: an .http file with a malformed in-place variable (name only, no equals, no value)
	requestFilePath := "testdata/execute_inplace_vars/malformed_name_only_no_equals_no_value/request.http"
	expectedErrorSubstring := "malformed in-place variable definition, missing '=' or name"

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: an error should occur indicating a parsing failure due to the malformed variable
	require.Error(t, execErr, "ExecuteFile should return an error for malformed variable definition")
	assert.Contains(t, execErr.Error(), "failed to parse request file", "Error message should indicate parsing failure")
	assert.Contains(t, execErr.Error(), expectedErrorSubstring, "Error message should contain specific malformed reason")
}

func TestExecuteFile_InPlace_Malformed_NoNameEqualsValue(t *testing.T) {
	// Given: an .http file with a malformed in-place variable (no name, equals, value)
	requestFilePath := "testdata/execute_inplace_vars/malformed_no_name_equals_value/request.http"
	expectedErrorSubstring := "variable name cannot be empty in definition"

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: an error should occur indicating a parsing failure due to the malformed variable
	require.Error(t, execErr, "ExecuteFile should return an error for malformed variable definition")
	assert.Contains(t, execErr.Error(), "failed to parse request file", "Error message should indicate parsing failure")
	assert.Contains(t, execErr.Error(), expectedErrorSubstring, "Error message should contain specific malformed reason")
}

func TestExecuteFile_InPlace_VariableDefinedByDotEnvSystemVariable(t *testing.T) {
	// Given: an .http file using {{$dotenv VAR_NAME}} for an in-place variable,
	// and a .env file defining VAR_NAME in the same directory as the .http file.
	const requestFilePath = "testdata/execute_inplace_vars/inplace_variable_defined_by_dotenv_system_variable/request.http"
	// The .env file is: testdata/execute_inplace_vars/inplace_variable_defined_by_dotenv_system_variable/.env
	// with DOTENV_VAR_FOR_SYSTEM_TEST=actual_dotenv_value

	const expectedSubstitutedValue = "actual_dotenv_value"
	var capturedURLPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		w.WriteHeader(http.StatusOK) // Minimal response for the server
	}))
	defer server.Close()

	client, err := NewClient(WithVars(map[string]interface{}{
		"test_server_url": server.URL,
	}))
	require.NoError(t, err)

	// When: the HTTP file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: the request should be successful and the variable substituted correctly
	require.NoError(t, execErr, "ExecuteFile returned an unexpected error")
	require.Len(t, responses, 1, "Expected one response")
	require.Nil(t, responses[0].Error, "Response error should be nil")
	assert.Equal(t, http.StatusOK, responses[0].StatusCode, "Expected HTTP 200 OK")
	assert.Equal(t, "/"+expectedSubstitutedValue, capturedURLPath, "Expected path to be substituted with .env value via {{$dotenv}}")

	// Verify ParsedFile.FileVariables to confirm parser behavior with {{$dotenv}}
	parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "{{$dotenv DOTENV_VAR_FOR_SYSTEM_TEST}}", parsedFile.FileVariables["my_api_key"], "Parsed file variable 'my_api_key' (placeholder) mismatch")
	assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["test_server_url"], "Parsed file variable 'test_server_url' (placeholder) mismatch")
}

func TestExecuteFile_InPlace_VariableDefinedByRandomInt(t *testing.T) {
	// Given: an .http file using {{$randomInt MIN MAX}} for an in-place variable
	const requestFilePath = "testdata/execute_inplace_vars/inplace_variable_defined_by_random_int/request.http"
	const minPort = 8000
	const maxPort = 8080

	var capturedURLPath string
	var capturedHeaderValue string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		capturedHeaderValue = r.Header.Get("X-Random-Port")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(WithVars(map[string]interface{}{
		"test_server_url": server.URL,
	}))
	require.NoError(t, err)

	// When: the HTTP file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: the request should be successful and the variable substituted with a random int in range
	require.NoError(t, execErr, "ExecuteFile returned an unexpected error")
	require.Len(t, responses, 1, "Expected one response")
	require.Nil(t, responses[0].Error, "Response error should be nil")
	assert.Equal(t, http.StatusOK, responses[0].StatusCode, "Expected HTTP 200 OK")

	// Extract port from path: /port/{{my_random_port}}
	pathParts := strings.Split(strings.Trim(capturedURLPath, "/"), "/")
	require.Len(t, pathParts, 2, "URL path should be in format /port/NUMBER")
	require.Equal(t, "port", pathParts[0], "First part of path should be 'port'")

	portFromPathStr := pathParts[1]
	portFromPath, err := strconv.Atoi(portFromPathStr)
	require.NoError(t, err, "Port from path should be a valid integer. Got: %s", portFromPathStr)

	portFromHeader, err := strconv.Atoi(capturedHeaderValue)
	require.NoError(t, err, "Port from X-Random-Port header should be a valid integer. Got: %s", capturedHeaderValue)

	assert.Equal(t, portFromPath, portFromHeader, "Port from path and header should match")

	assert.GreaterOrEqual(t, portFromPath, minPort, "Port should be >= %d. Got: %d", minPort, portFromPath)
	assert.LessOrEqual(t, portFromPath, maxPort, "Port should be <= %d. Got: %d", maxPort, portFromPath)
}
