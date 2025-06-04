package restclient

import (
	"bytes"
	"context"
	// "encoding/hex" // Not needed for these specific tests
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url" // Added for TestExecuteFile_WithHttpClientEnvJson
	"os"
	"path/filepath"
	"runtime" // Added for testGetProjectRoot
	// "strconv" // Not needed for these specific tests
	"strings" // Added for TestExecuteFile_WithHttpClientEnvJson
	// "sync/atomic" // Not needed for these specific tests
	"testing"
	"text/template" // Added for processing env file template
	// "time" // Not needed for these specific tests

	// "github.com/google/uuid" // Not needed for these specific tests
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_WithDotEnvSystemVariable(t *testing.T) {
	// Given
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Dotenv-Value")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := NewClient()
	tempDir := t.TempDir()

	// Scenario 1: .env file exists and variable is present
	// Given
	dotEnvContent1 := "DOTENV_VAR1=dotenv_value_one\nDOTENV_VAR2=another val from dotenv"
	dotEnvFile1Path := filepath.Join(tempDir, ".env")
	err := os.WriteFile(dotEnvFile1Path, []byte(dotEnvContent1), 0644)
	require.NoError(t, err)

	httpFile1Path := createTestFileFromTemplate(t,
		"testdata/client_execute_env_vars/dotenv_scenario1_request.http.template",
		struct{ ServerURL string }{ServerURL: server.URL},
	)

	// When
	responses1, err1 := client.ExecuteFile(context.Background(), httpFile1Path)

	// Then
	require.NoError(t, err1, "ExecuteFile (scenario 1) should not return an error for $dotenv processing")
	require.Len(t, responses1, 1, "Expected 1 response for scenario 1")
	resp1 := responses1[0]
	assert.NoError(t, resp1.Error)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	expectedURL1 := "/path-dotenv_value_one/data" // SCENARIO-LIB-020-001
	assert.Equal(t, expectedURL1, interceptedRequest.URL, "URL (scenario 1) should contain substituted dotenv variable")
	assert.Equal(t, "another val from dotenv", interceptedRequest.Header, "X-Dotenv-Value header (scenario 1) should contain substituted dotenv variable") // SCENARIO-LIB-020-001

	var bodyJSON1 map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON1)
	require.NoError(t, err, "Failed to unmarshal request body JSON (scenario 1)")
	dotenvPayload1, ok1 := bodyJSON1["payload"]
	require.True(t, ok1, "payload not found in body (scenario 1)")
	assert.Equal(t, "dotenv_value_one", dotenvPayload1, "Body payload (scenario 1) should contain substituted dotenv variable") // SCENARIO-LIB-020-001
	missingPayload1, ok2 := bodyJSON1["missing_payload"]
	require.True(t, ok2, "missing_payload not found in body (scenario 1)")
	assert.Empty(t, missingPayload1, "Body missing_payload (scenario 1) should be empty for a missing dotenv variable") // SCENARIO-LIB-020-002

	// Scenario 2: .env file does not exist
	// Given
	err = os.Remove(dotEnvFile1Path)
	require.NoError(t, err, "Failed to remove .env file for scenario 2 prep")

	httpFile2Path := createTestFileFromTemplate(t,
		"testdata/client_execute_env_vars/dotenv_scenario2_request.http.template",
		struct{ ServerURL string }{ServerURL: server.URL},
	)

	// When
	responses2, err2 := client.ExecuteFile(context.Background(), httpFile2Path)

	// Then
	require.NoError(t, err2, "ExecuteFile (scenario 2) should not return an error if .env not found")
	require.Len(t, responses2, 1, "Expected 1 response for scenario 2")
	resp2 := responses2[0]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	expectedURL2 := "/path-/data" // SCENARIO-LIB-020-003
	assert.Equal(t, expectedURL2, interceptedRequest.URL, "URL (scenario 2) should have empty substitution for dotenv variable")

	var bodyJSON2 map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON2)
	require.NoError(t, err, "Failed to unmarshal request body JSON (scenario 2)")
	dotenvPayload2, ok3 := bodyJSON2["payload"]
	require.True(t, ok3, "payload not found in body (scenario 2)")
	assert.Empty(t, dotenvPayload2, "Body payload (scenario 2) should be empty if .env not found") // SCENARIO-LIB-020-003
}

// SCENARIO-LIB-018-001: Env selected, http-client.env.json exists, env exists in file
func TestExecuteFile_HttpClientEnv_Selected_Exists_InFile(t *testing.T) {
	// Given
	var interceptedRequest struct {
		Path   string
		Host   string
		Header string
		Body   string
		Method string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.Path = r.URL.Path
		interceptedRequest.Host = r.Host
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Env-Var")
		interceptedRequest.Method = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create http-client.env.json
	envFilePath := filepath.Join(tempDir, "http-client.env.json")
	// Read and process the http-client.env.json template
	envTemplatePath := filepath.Join(testGetProjectRoot(t), "testdata", "client_execute_env_vars", "http-client.env.json.template")
	envTemplateBytes, err := os.ReadFile(envTemplatePath)
	require.NoError(t, err, "Failed to read http-client.env.json.template")

	tmpl, err := template.New("http-client.env.json").Parse(string(envTemplateBytes))
	require.NoError(t, err, "Failed to parse http-client.env.json.template")

	var processedEnvContent bytes.Buffer
	err = tmpl.Execute(&processedEnvContent, struct{ ServerURL string }{ServerURL: server.URL})
	require.NoError(t, err, "Failed to execute http-client.env.json.template")

	err = os.WriteFile(envFilePath, processedEnvContent.Bytes(), 0600)
	require.NoError(t, err, "Failed to write processed http-client.env.json")

	// Create request file
	httpFilePath := filepath.Join(tempDir, "test_env_vars.http")
	// Path to the source template file in testdata
	projectRoot := testGetProjectRoot(t) // Helper to get project root reliably
	sourceHTTPRequestFilePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "test_env_vars_request.http")
	httpRequestFileContentBytes, readErr := os.ReadFile(sourceHTTPRequestFilePath)
	require.NoError(t, readErr, "Failed to read source HTTP request file: "+sourceHTTPRequestFilePath)
	writeErr := os.WriteFile(httpFilePath, httpRequestFileContentBytes, 0600)
	require.NoError(t, writeErr, "Failed to write HTTP request content to temp file")

	client, err := NewClient(WithEnvironment("dev"))
	require.NoError(t, err)

	// When
	responses, err := client.ExecuteFile(context.Background(), httpFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, http.MethodPost, interceptedRequest.Method)
	parsedServerURL, pErr := url.Parse(server.URL)
	require.NoError(t, pErr)
	assert.Equal(t, parsedServerURL.Host, interceptedRequest.Host)
	assert.Equal(t, "/resource/dev-user", interceptedRequest.Path)
	assert.Equal(t, "dev-token", interceptedRequest.Header)
	// projectRoot is already defined in this scope
	expectedBodyFilePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "scenario1_expected_response_body.json")
	expectedBodyBytes, err := os.ReadFile(expectedBodyFilePath)
	require.NoError(t, err, "Failed to read expected_response_body.json")
	assert.JSONEq(t, string(expectedBodyBytes), interceptedRequest.Body)
	assert.Equal(t, "dev", client.selectedEnvironmentName) // Verify client has the env name
	// EnvironmentVariables are used internally; their effect is checked by the substituted values above.
}

// SCENARIO-LIB-018-002: Env selected, http-client.env.json exists, but env NOT in file
func TestExecuteFile_HttpClientEnv_Selected_Exists_NotInFile(t *testing.T) {
	// Given
	serverURL := "http://localhost:12345" // A dummy URL, server won't actually be hit with {{host}}
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		// This handler might not be reached if {{host}} isn't resolved by any mechanism
		// and the HTTP client fails before sending.
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	tempDir := t.TempDir()
	// Read and process the http-client.env.json template for scenario 2
	envFilePath := filepath.Join(tempDir, "http-client.env.json")
	projectRoot := testGetProjectRoot(t) // Ensure projectRoot is available
	envTemplatePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "http-client.env.scenario2.json.template")
	envTemplateBytes, err := os.ReadFile(envTemplatePath)
	require.NoError(t, err, "Failed to read http-client.env.scenario2.json.template")

	tmpl, err := template.New("http-client.env.json").Parse(string(envTemplateBytes))
	require.NoError(t, err, "Failed to parse http-client.env.scenario2.json.template")

	var processedEnvContent bytes.Buffer
	err = tmpl.Execute(&processedEnvContent, struct{ ServerURL string }{ServerURL: serverURL})
	require.NoError(t, err, "Failed to execute http-client.env.scenario2.json.template")

	err = os.WriteFile(envFilePath, processedEnvContent.Bytes(), 0600)
	require.NoError(t, err, "Failed to write processed http-client.env.json for scenario 2")

	// Copy the HTTP request file for 'env not in file' scenario
	httpFilePath := filepath.Join(tempDir, "test_env_vars_missing_env.http")
	sourceHTTPRequestFilePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "test_env_vars_missing_env_request.http")
	httpRequestFileContentBytes, err := os.ReadFile(sourceHTTPRequestFilePath)
	require.NoError(t, err, "Failed to read source HTTP request file for 'env not in file' scenario")
	err = os.WriteFile(httpFilePath, httpRequestFileContentBytes, 0600)
	require.NoError(t, err, "Failed to write HTTP request content to temp file for 'env not in file' scenario")

	client, err := NewClient(WithEnvironment("staging"))
	require.NoError(t, err)

	// When
	responses, err := client.ExecuteFile(context.Background(), httpFilePath)

	// Then
	require.Error(t, err)                                               // ExecuteFile itself should return an error if a request fails this way
	assert.Contains(t, err.Error(), "unsupported protocol scheme \"\"") // Check the error from ExecuteFile
	require.Len(t, responses, 1)
	resp := responses[0]
	require.NotNil(t, resp)
	assert.Error(t, resp.Error) // Expect an error because {{host}} is not resolved, leading to bad URL
	assert.Contains(t, resp.Error.Error(), "unsupported protocol scheme \"\"")

	// Check that {{host}} was not replaced because 'staging' env was not found
	// The RawURLString should still contain the placeholder as it was in the file.
	assert.True(t, strings.Contains(resp.Request.RawURLString, "{{host}}"), "RawURLString should still contain {{host}}")
	assert.Equal(t, "staging", client.selectedEnvironmentName)
	// EnvironmentVariables map on ParsedFile would be nil internally, effect is placeholder {{host}} remains.
}

// SCENARIO-LIB-018-003: Env selected, but http-client.env.json does NOT exist
func TestExecuteFile_HttpClientEnv_Selected_FileDoesNotExist(t *testing.T) {
	// Given
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	defer server.Close()

	tempDir := t.TempDir()
	// http-client.env.json is NOT created in tempDir

	// Copy the HTTP request file for 'file does NOT exist' scenario
	projectRoot := testGetProjectRoot(t) // Ensure projectRoot is available
	httpFilePath := filepath.Join(tempDir, "test_env_vars_no_env_file.http")
	sourceHTTPRequestFilePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "test_env_vars_missing_env_request.http") // Reusing existing file
	httpRequestFileContentBytes, err := os.ReadFile(sourceHTTPRequestFilePath)
	require.NoError(t, err, "Failed to read source HTTP request file for 'file does NOT exist' scenario")
	err = os.WriteFile(httpFilePath, httpRequestFileContentBytes, 0600)
	require.NoError(t, err, "Failed to write HTTP request content to temp file for 'file does NOT exist' scenario")

	client, err := NewClient(WithEnvironment("dev"))
	require.NoError(t, err)

	// When
	responses, err := client.ExecuteFile(context.Background(), httpFilePath)

	// Then
	require.Error(t, err)                                               // ExecuteFile itself should return an error if a request fails this way
	assert.Contains(t, err.Error(), "unsupported protocol scheme \"\"") // Check the error from ExecuteFile
	require.Len(t, responses, 1)
	resp := responses[0]
	require.NotNil(t, resp)
	assert.Error(t, resp.Error) // Expect an error because {{host}} is not resolved, leading to bad URL
	assert.Contains(t, resp.Error.Error(), "unsupported protocol scheme \"\"")
	assert.True(t, strings.Contains(resp.Request.RawURLString, "{{host}}"), "RawURLString should still contain {{host}}")
	assert.Equal(t, "dev", client.selectedEnvironmentName)
	// EnvironmentVariables map on ParsedFile would be nil internally, effect is placeholder {{host}} remains.
}

// SCENARIO-LIB-018-004: No env selected, http-client.env.json exists
func TestExecuteFile_HttpClientEnv_NotSelected_FileExists(t *testing.T) {
	// Given
	serverURL := "http://localhost:54321"
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	defer server.Close()

	tempDir := t.TempDir()
	// Read and process the http-client.env.json template for 'no env selected, file exists' scenario
	projectRoot := testGetProjectRoot(t) // Ensure projectRoot is available
	envFilePath := filepath.Join(tempDir, "http-client.env.json")
	envTemplatePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "http-client.env.scenario2.json.template") // Reusing template
	envTemplateBytes, err := os.ReadFile(envTemplatePath)
	require.NoError(t, err, "Failed to read http-client.env.scenario2.json.template")

	tmpl, err := template.New("http-client.env.json").Parse(string(envTemplateBytes))
	require.NoError(t, err, "Failed to parse http-client.env.scenario2.json.template")

	var processedEnvContent bytes.Buffer
	err = tmpl.Execute(&processedEnvContent, struct{ ServerURL string }{ServerURL: serverURL})
	require.NoError(t, err, "Failed to execute http-client.env.scenario2.json.template")

	err = os.WriteFile(envFilePath, processedEnvContent.Bytes(), 0600)
	require.NoError(t, err, "Failed to write processed http-client.env.json for 'no env selected, file exists' scenario")

	// Copy the HTTP request file for 'no env selected, file exists' scenario
	httpFilePath := filepath.Join(tempDir, "test_no_env_selected.http")
	sourceHTTPRequestFilePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "test_env_vars_missing_env_request.http") // Reusing existing file
	httpRequestFileContentBytes, err := os.ReadFile(sourceHTTPRequestFilePath)
	require.NoError(t, err, "Failed to read source HTTP request file for 'no env selected, file exists' scenario")
	err = os.WriteFile(httpFilePath, httpRequestFileContentBytes, 0600)
	require.NoError(t, err, "Failed to write HTTP request content to temp file for 'no env selected, file exists' scenario")

	client, err := NewClient() // No WithEnvironment option
	require.NoError(t, err)

	// When
	responses, errExecute := client.ExecuteFile(context.Background(), httpFilePath)

	// Then
	require.Error(t, errExecute) // ExecuteFile itself should return an error if a request fails this way
	assert.Contains(t, errExecute.Error(), "unsupported protocol scheme \"\"")
	require.Len(t, responses, 1)
	resp := responses[0]
	require.NotNil(t, resp)
	assert.Error(t, resp.Error) // Expect an error because {{host}} is not resolved, leading to bad URL
	assert.Contains(t, resp.Error.Error(), "unsupported protocol scheme \"\"")
	assert.True(t, strings.Contains(resp.Request.RawURLString, "{{host}}"), "RawURLString should still contain {{host}}")
	assert.Empty(t, client.selectedEnvironmentName) // No env was selected
	// EnvironmentVariables map on ParsedFile would be nil internally, effect is placeholder {{host}} remains.
}

// SCENARIO-LIB-018-005: Env selected, http-client.env.json is malformed
func TestExecuteFile_HttpClientEnv_Selected_FileMalformed(t *testing.T) {
	// Given
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	defer server.Close()

	tempDir := t.TempDir()
	// Copy the malformed environment JSON file
	projectRoot := testGetProjectRoot(t) // Ensure projectRoot is available
	envFilePath := filepath.Join(tempDir, "http-client.env.json")
	sourceEnvFilePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "http-client.env.malformed.json")
	envFileBytes, err := os.ReadFile(sourceEnvFilePath)
	require.NoError(t, err, "Failed to read source malformed env JSON file")
	err = os.WriteFile(envFilePath, envFileBytes, 0600)
	require.NoError(t, err, "Failed to write malformed env JSON to temp file")

	// Copy the HTTP request file
	httpFilePath := filepath.Join(tempDir, "test_malformed_env.http")
	sourceHTTPRequestFilePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "test_env_vars_missing_env_request.http") // Reusing existing file
	httpRequestFileContentBytes, err := os.ReadFile(sourceHTTPRequestFilePath)
	require.NoError(t, err, "Failed to read source HTTP request file for malformed env scenario")
	err = os.WriteFile(httpFilePath, httpRequestFileContentBytes, 0600)
	require.NoError(t, err, "Failed to write HTTP request content to temp file for malformed env scenario")

	client, clientErr := NewClient(WithEnvironment("dev"))
	require.NoError(t, clientErr) // NewClient itself doesn't fail on malformed env file, load is lazy

	// When
	responses, execErr := client.ExecuteFile(context.Background(), httpFilePath)

	// Then
	// The client.loadEnvironmentVariables method logs an error for malformed JSON
	// but proceeds as if the environment was not found or the file didn't exist.
	// So, the behavior should be similar to "env NOT in file" or "file does NOT exist".
	require.Error(t, execErr)                                               // ExecuteFile itself should return an error if a request fails this way
	assert.Contains(t, execErr.Error(), "unsupported protocol scheme \"\"") // Check the error from ExecuteFile
	require.Len(t, responses, 1)
	resp := responses[0]
	require.NotNil(t, resp)
	assert.Error(t, resp.Error) // Expect an error because {{host}} is not resolved, leading to bad URL
	assert.Contains(t, resp.Error.Error(), "unsupported protocol scheme \"\"")
	assert.True(t, strings.Contains(resp.Request.RawURLString, "{{host}}"), "RawURLString should still contain {{host}}")
}

// SCENARIO-LIB-018-006: Env selected, http-client.env.json exists, but specific env key is not a map/object
func TestExecuteFile_HttpClientEnv_Selected_KeyNotObject(t *testing.T) {
	// Given
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	defer server.Close()

	tempDir := t.TempDir()
	envFilePath := filepath.Join(tempDir, "http-client.env.json")
	// Copy the 'key not an object' environment JSON file
	projectRoot := testGetProjectRoot(t) // Ensure projectRoot is available
	sourceEnvFilePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "http-client.env.keynotobject.json")
	envFileBytes, err := os.ReadFile(sourceEnvFilePath)
	require.NoError(t, err, "Failed to read source 'key not an object' env JSON file")
	err = os.WriteFile(envFilePath, envFileBytes, 0600)
	require.NoError(t, err, "Failed to write 'key not an object' env JSON to temp file")

	// Copy the HTTP request file
	httpFilePath := filepath.Join(tempDir, "test_env_not_object.http")
	sourceHTTPRequestFilePath := filepath.Join(projectRoot, "testdata", "client_execute_env_vars", "test_env_vars_missing_env_request.http") // Reusing existing file
	httpRequestFileContentBytes, err := os.ReadFile(sourceHTTPRequestFilePath)
	require.NoError(t, err, "Failed to read source HTTP request file for 'key not an object' scenario")
	err = os.WriteFile(httpFilePath, httpRequestFileContentBytes, 0600)
	require.NoError(t, err, "Failed to write HTTP request content to temp file for 'key not an object' scenario")

	client, err := NewClient(WithEnvironment("dev"))
	require.NoError(t, err)

	// When
	responses, err := client.ExecuteFile(context.Background(), httpFilePath)

	// Then
	// Similar to malformed JSON, if the specific environment is not a map,
	// it's treated as if the environment variables for 'dev' were not found.
	require.Error(t, err)                                               // ExecuteFile itself should return an error if a request fails this way
	assert.Contains(t, err.Error(), "unsupported protocol scheme \"\"") // Check the error from ExecuteFile
	require.Len(t, responses, 1)
	resp := responses[0]
	require.NotNil(t, resp)
	assert.Error(t, resp.Error) // Expect an error because {{host}} is not resolved, leading to bad URL
	assert.Contains(t, resp.Error.Error(), "unsupported protocol scheme \"\"")
	assert.True(t, strings.Contains(resp.Request.RawURLString, "{{host}}"), "RawURLString should still contain {{host}}")
}

// testGetProjectRoot is a helper to reliably find the project root directory
// based on the location of the go.mod file.
func testGetProjectRoot(t *testing.T) string {
	t.Helper()
	// Get the directory of the current test file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to get current file path using runtime.Caller")
	}
	currentDir := filepath.Dir(currentFile)

	// Traverse up to find go.mod
	dir := currentDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir // Found go.mod, this is the project root
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// Reached filesystem root without finding go.mod
			t.Fatalf("Failed to find project root (go.mod) from test file: %s", currentFile)
		}
		dir = parentDir
	}
}
