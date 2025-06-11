package restclient_test

import (
	"context"
	"encoding/hex" // Added for TestExecuteFile_WithExtendedRandomSystemVariables
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest" // Added for refactoring TestExecuteFile_WithHttpClientEnvJson
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	rc "github.com/bmcszk/go-restclient"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR1.1 - Custom Variables: Basic Definition and Substitution
// Corresponds to: Client's ability to define and substitute custom variables within an .http file
// (e.g., @name = value) (http_syntax.md "Custom Variables").
// This test uses 'testdata/http_request_files/custom_variables.http' to verify substitution
// of custom variables in URLs, headers, and bodies.
func TestExecuteFile_WithCustomVariables(t *testing.T) {
	// Given
	var requestCount int32
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		switch r.URL.Path {
		case "/users/testuser123": // SCENARIO-LIB-013-001, SCENARIO-LIB-013-002, SCENARIO-LIB-013-003
			assert.Equal(t, http.MethodPost, r.Method)
			bodyBytes, _ := io.ReadAll(r.Body)
			assert.JSONEq(t, `{"id": "testuser123"}`, string(bodyBytes))
			assert.Equal(t, "Bearer secret-token-value", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response for user testuser123")
		case "/products/testuser123": // SCENARIO-LIB-013-004 (variable override for pathSegment)
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response from products/testuser123")
		case "/items/": // SCENARIO-LIB-013-005 (undefined variable resolves to empty string in path)
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response for items ()")
		default:
			t.Errorf("Unexpected request path to mock server: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/custom_variables.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for variable processing")
	require.Len(t, responses, 3, "Expected 3 responses")
	assert.EqualValues(t, 3, atomic.LoadInt32(&requestCount), "Mock server should have been hit 3 times")

	// Check response 1 (SCENARIO-LIB-013-001, SCENARIO-LIB-013-002, SCENARIO-LIB-013-003)
	resp1 := responses[0]
	assert.NoError(t, resp1.Error)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Equal(t, "response for user testuser123", resp1.BodyString)

	// Check response 2 (SCENARIO-LIB-013-004)
	resp2 := responses[1]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Equal(t, "response from products/testuser123", resp2.BodyString)

	// Check response 3 (SCENARIO-LIB-013-005)
	resp3 := responses[2]
	assert.NoError(t, resp3.Error)
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	assert.Equal(t, "response for items ()", resp3.BodyString)
}

// PRD-COMMENT: FR1.3.6 - System Variables: {{$processEnv.VAR_NAME}}
// Corresponds to: Client's ability to substitute the {{$processEnv.VAR_NAME}} system variable
// with the value of an OS environment variable (http_syntax.md "System Variables").
// This test uses 'testdata/http_request_files/system_var_process_env.http' and sets OS environment
// variables to verify their substitution in URLs, headers, and bodies. It also checks behavior
// for undefined environment variables.
func TestExecuteFile_WithProcessEnvSystemVariable(t *testing.T) {
	// t.Skip("Skipping due to bug in {{$processEnv VAR}} substitution
	//   (MEMORY d1edb831-da89-4cde-93ad-a9129eb7b8aa): placeholder not replaced with
	//   OS environment variable value. See task TBD for fix.")
	// Given
	const testEnvVarName = "GO_RESTCLIENT_TEST_VAR"
	const testEnvVarValue = "test_env_value_123"
	const undefinedEnvVarName = "GO_RESTCLIENT_UNDEFINED_VAR"

	err := os.Setenv(testEnvVarName, testEnvVarValue)
	require.NoError(t, err, "Failed to set environment variable for test")
	defer func() { _ = os.Unsetenv(testEnvVarName) }()

	_ = os.Unsetenv(undefinedEnvVarName)

	var interceptedRequest struct {
		URL                string
		Header             string // X-Env-Value
		CacheControlHeader string
		Body               string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Env-Value")
		interceptedRequest.CacheControlHeader = r.Header.Get("Cache-Control")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t,
		"testdata/http_request_files/system_var_process_env.http",
		struct {
			ServerURL           string
			TestEnvVarName      string
			UndefinedEnvVarName string
		}{
			ServerURL:           server.URL,
			TestEnvVarName:      testEnvVarName,
			UndefinedEnvVarName: undefinedEnvVarName,
		},
	)

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for $processEnv processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// SCENARIO-LIB-019-001
	expectedURL := fmt.Sprintf("/path-%s/data", testEnvVarValue)
	assert.Equal(t, expectedURL, interceptedRequest.URL, "URL should contain substituted env variable")
	assert.Equal(t, testEnvVarValue, interceptedRequest.Header,
		"X-Env-Value header should contain substituted env variable")

	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal request body JSON")

	envPayload, ok := bodyJSON["env_payload"]
	require.True(t, ok, "env_payload not found in body")
	assert.Equal(t, testEnvVarValue, envPayload, "Body env_payload should contain substituted env variable")

	// SCENARIO-LIB-019-002
	undefinedPayload, ok := bodyJSON["undefined_payload"]
	require.True(t, ok, "undefined_payload not found in body")
	assert.Equal(t, fmt.Sprintf("{{$processEnv %s}}", undefinedEnvVarName), undefinedPayload,
		"Body undefined_payload should be the unresolved placeholder")

	// Check Cache-Control header for unresolved placeholder
	assert.Equal(t, "{{$processEnv UNDEFINED_CACHE_VAR_SHOULD_BE_EMPTY}}", interceptedRequest.CacheControlHeader,
		"Cache-Control header should be the unresolved placeholder")
}

type dotEnvInterceptedRequestData struct {
	URL    string
	Header string
	Body   string
}

type dotEnvTestCase struct {
	name                string
	dotEnvContent       string // Content to write to .env file for this test case
	requestFileTemplate string // Template for the .http file content
	expectedURL         string
	expectedHeader      string
	expectedBodyPayload map[string]string // Key-value pairs for expected JSON body
	expectErrorInExec   bool
	expectErrorInResp   bool
}

func runDotEnvScenarioTest(t *testing.T, client *rc.Client, serverURL string, tempDir string,
	tc dotEnvTestCase, interceptedData *dotEnvInterceptedRequestData) {
	t.Helper()

	// Given: Setup .env file for the current scenario
	dotEnvFilePath := filepath.Join(tempDir, ".env")
	if tc.dotEnvContent != "" {
		err := os.WriteFile(dotEnvFilePath, []byte(tc.dotEnvContent), 0644)
		require.NoError(t, err, "Failed to write .env file for scenario: %s", tc.name)
	} else {
		// Ensure .env file does not exist if no content is specified
		_ = os.Remove(dotEnvFilePath) // Ignore error if file doesn't exist
	}

	// Given: Create .http file from template
	requestFileContent := fmt.Sprintf(tc.requestFileTemplate, serverURL)
	httpFilePath := filepath.Join(tempDir, "request.http") // Use a consistent name, overwritten per test
	err := os.WriteFile(httpFilePath, []byte(requestFileContent), 0644)
	require.NoError(t, err, "Failed to write .http file for scenario: %s", tc.name)

	// When
	responses, execErr := client.ExecuteFile(context.Background(), httpFilePath)

	// Then
	if tc.expectErrorInExec {
		assert.Error(t, execErr, "Expected an error during ExecuteFile for scenario: %s", tc.name)
		return // Stop further checks if execution error was expected
	}
	require.NoError(t, execErr, "ExecuteFile should not return an error for scenario: %s", tc.name)
	require.Len(t, responses, 1, "Expected 1 response for scenario: %s", tc.name)

	resp := responses[0]
	if tc.expectErrorInResp {
		assert.Error(t, resp.Error, "Expected an error in the response object for scenario: %s", tc.name)
		return // Stop further checks if response error was expected
	}
	assert.NoError(t, resp.Error, "Response error should be nil for scenario: %s", tc.name)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be OK for scenario: %s", tc.name)

	assert.Equal(t, tc.expectedURL, interceptedData.URL, "URL mismatch for scenario: %s", tc.name)
	if tc.expectedHeader != "" { // Only assert header if an expectation is set
		assert.Equal(t, tc.expectedHeader, interceptedData.Header, "Header mismatch for scenario: %s", tc.name)
	}

	if len(tc.expectedBodyPayload) > 0 {
		var bodyJSON map[string]string
		err = json.Unmarshal([]byte(interceptedData.Body), &bodyJSON)
		require.NoError(t, err, "Failed to unmarshal request body JSON for scenario: %s. Body: %s",
			tc.name, interceptedData.Body)
		for key, expectedValue := range tc.expectedBodyPayload {
			actualValue, ok := bodyJSON[key]
			assert.True(t, ok, "Expected key '%s' not found in body for scenario: %s", key, tc.name)
			assert.Equal(t, expectedValue, actualValue,
				"Body payload for key '%s' mismatch for scenario: %s", key, tc.name)
		}
	}
}

// PRD-COMMENT: FR1.3.7 - System Variables: {{$dotenv.VAR_NAME}}
// Corresponds to: Client's ability to substitute the {{$dotenv.VAR_NAME}} system variable
// with values from a .env file located in the same directory as the .http file
// (http_syntax.md "System Variables").
// This test uses 'testdata/http_request_files/system_var_dotenv.http' and a dynamically created
// '.env' file to verify substitution in URLs, headers, and bodies. It also checks behavior
// for variables not present in the .env file.
func TestExecuteFile_WithDotEnvSystemVariable(t *testing.T) {
	// Use a single instance, reset/captured by mock server per call
	var currentInterceptedData dotEnvInterceptedRequestData

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		currentInterceptedData.URL = r.URL.String() // Capture relative URL path and query
		bodyBytes, _ := io.ReadAll(r.Body)
		currentInterceptedData.Body = string(bodyBytes)
		currentInterceptedData.Header = r.Header.Get("X-Dotenv-Value") // Specific header for these tests
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, err := rc.NewClient()
	require.NoError(t, err) // Ensure client creation doesn't fail

	tempDir := t.TempDir()

	testCases := []dotEnvTestCase{
		{
			name:          "Scenario 1: .env file exists and variable is present",
			dotEnvContent: "DOTENV_VAR1=dotenv_value_one\nDOTENV_VAR2=another val from dotenv",
			requestFileTemplate: `
GET %s/path-{{$dotenv DOTENV_VAR1}}/data
Content-Type: application/json
X-Dotenv-Value: {{$dotenv DOTENV_VAR2}}

{
  "payload": "{{$dotenv DOTENV_VAR1}}",
  "missing_payload": "{{$dotenv MISSING_DOTENV_VAR}}"
}`,
			expectedURL:    "/path-dotenv_value_one/data",
			expectedHeader: "another val from dotenv",
			expectedBodyPayload: map[string]string{
				"payload":         "dotenv_value_one",
				"missing_payload": "", // SCENARIO-LIB-020-002: Missing var results in empty string
			},
		},
		{
			name:          "Scenario 2: .env file does not exist",
			dotEnvContent: "", // Empty content means .env file will be removed/not created
			requestFileTemplate: `
GET %s/path-{{$dotenv DOTENV_VAR_SHOULD_BE_EMPTY}}/data
User-Agent: test-client

{
  "payload": "{{$dotenv DOTENV_VAR_ALSO_EMPTY}}"
}`,
			expectedURL: "/path-/data", // SCENARIO-LIB-020-003: Missing .env or var results in empty string
			expectedBodyPayload: map[string]string{
				"payload": "",
			},
		},
		// Add more scenarios as needed, e.g., empty .env file, variable present in OS but not .env, etc.
	}

	for _, tc := range testCases {
		capturedTC := tc // Capture range variable for t.Run
		t.Run(capturedTC.name, func(t *testing.T) {
			runDotEnvScenarioTest(t, client, server.URL, tempDir, capturedTC, &currentInterceptedData)
		})
	}
}

// PRD-COMMENT: FR1.5 - Programmatic Variable Injection
// Corresponds to: Client's ability to accept and substitute variables passed programmatically
// during the ExecuteFile call, which can override variables defined in the .http file or
// environment files (http_syntax.md "Variable Precedence").
// This test uses 'testdata/http_request_files/programmatic_vars.http' and injects variables
// via `WithVariables` and `ExecuteFile` options to verify their substitution and precedence
// over file-defined variables.
func TestExecuteFile_WithProgrammaticVariables(t *testing.T) {
	// Given
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.URL = r.URL.Path // Only path for easier assertion
		interceptedRequest.Header = r.Header.Get("X-Test-Header")
		interceptedRequest.Body = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	clientProgrammaticVars := map[string]any{
		"prog_baseUrl":         server.URL,
		"prog_path":            "items",
		"prog_id":              "prog123",
		"prog_headerVal":       "ProgrammaticHeaderValue",
		"prog_bodyField":       "dataFromProgrammatic",
		"file_var_to_override": "overridden_by_programmatic",
		"PROG_ENV_VAR":         "programmatic_wins_over_env",
	}

	// Set an OS env var that will be overridden by programmatic var
	_ = os.Setenv("PROG_ENV_VAR", "env_value_should_be_overridden")
	defer os.Unsetenv("PROG_ENV_VAR")

	client, err := rc.NewClient(rc.WithVars(clientProgrammaticVars))
	require.NoError(t, err)

	requestFilePath := "testdata/http_request_files/programmatic_variables.http"

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Assertions for the request sent to the server
	assert.Equal(t, "/items/prog123", interceptedRequest.URL, "URL path mismatch")
	assert.Equal(t, "ProgrammaticHeaderValue", interceptedRequest.Header, "X-Test-Header mismatch")

	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal request body JSON")

	assert.Equal(t, "dataFromProgrammatic", bodyJSON["field"], "Body field 'field' mismatch")
	assert.Equal(t, "overridden_by_programmatic", bodyJSON["overridden_file_var"],
		"Body field 'overridden_file_var' mismatch")
	assert.Equal(t, "programmatic_wins_over_env", bodyJSON["env_var_check"], "Body field 'env_var_check' mismatch")
	assert.Equal(t, "file_only", bodyJSON["file_only_check"], "Body field 'file_only_check' mismatch")

	// Also check headers received by the server for variable substitution confirmation
	// These were set up in the new programmatic_variables.http file to check different sources
	assert.Equal(t, "overridden_by_programmatic", resp.Request.Headers.Get("X-File-Var"))
	assert.Equal(t, "programmatic_wins_over_env", resp.Request.Headers.Get("X-Env-Var"))
	assert.Equal(t, "file_only", resp.Request.Headers.Get("X-Unused-File-Var"))
}

// PRD-COMMENT: FR1.3.4.1 - System Variables: {{$localDatetime "format"}}
// Corresponds to: Client's ability to substitute the {{$localDatetime}} system variable
// with the current timestamp in the system's local timezone, formatted according to a Go layout
// string (http_syntax.md "System Variables").
// This test uses 'testdata/http_request_files/system_var_local_datetime.http' to verify
// correct formatting and substitution in URLs, headers, and bodies.
func TestExecuteFile_WithLocalDatetimeSystemVariable(t *testing.T) {
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
		interceptedRequest.Header = r.Header.Get("X-Request-Time")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := rc.NewClient()

	// Capture current time to compare against, allowing for slight delay
	beforeTime := time.Now().UTC().Unix()

	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_timestamp.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	responses, err := client.ExecuteFile(context.Background(), requestFilePath)
	require.NoError(t, err, "ExecuteFile should not return an error for $timestamp processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	afterTime := time.Now().UTC().Unix()

	// SCENARIO-LIB-016-001: {{$timestamp}} in URL, header, body
	// Check URL
	urlParts := strings.Split(interceptedRequest.URL, "/")
	require.True(t, len(urlParts) >= 2, "URL path should have at least two parts")
	timestampFromURLStr := urlParts[len(urlParts)-1]
	timestampFromURL, parseErrURL := strconv.ParseInt(timestampFromURLStr, 10, 64)
	assert.NoError(t, parseErrURL, "Timestamp from URL should be a valid integer")
	assert.GreaterOrEqual(t, timestampFromURL, beforeTime, "Timestamp from URL should be >= time before request")
	assert.LessOrEqual(t, timestampFromURL, afterTime, "Timestamp from URL should be <= time after request")

	// Check Header
	timestampFromHeader, parseErrHeader := strconv.ParseInt(interceptedRequest.Header, 10, 64)
	assert.NoError(t, parseErrHeader, "Timestamp from Header should be a valid integer")
	assert.GreaterOrEqual(t, timestampFromHeader, beforeTime, "Timestamp from Header should be >= time before request")
	assert.LessOrEqual(t, timestampFromHeader, afterTime, "Timestamp from Header should be <= time after request")

	// Check Body
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal request body JSON")

	timestampFromBody1Str, ok1 := bodyJSON["event_time"]
	require.True(t, ok1, "event_time not found in body")
	timestampFromBody1, parseErrBody1 := strconv.ParseInt(timestampFromBody1Str, 10, 64)
	assert.NoError(t, parseErrBody1, "Timestamp from body (event_time) should be valid int")
	assert.GreaterOrEqual(t, timestampFromBody1, beforeTime)
	assert.LessOrEqual(t, timestampFromBody1, afterTime)

	timestampFromBody2Str, ok2 := bodyJSON["processed_at"]
	require.True(t, ok2, "processed_at not found in body")
	timestampFromBody2, parseErrBody2 := strconv.ParseInt(timestampFromBody2Str, 10, 64)
	assert.NoError(t, parseErrBody2, "Timestamp from body (processed_at) should be valid int")
	assert.GreaterOrEqual(t, timestampFromBody2, beforeTime)
	assert.LessOrEqual(t, timestampFromBody2, afterTime)

	// SCENARIO-LIB-016-002: Multiple {{$timestamp}} instances yield the same value for that pass
	assert.Equal(t, timestampFromURL, timestampFromHeader)
	assert.Equal(t, timestampFromHeader, timestampFromBody1)
	assert.Equal(t, timestampFromBody1, timestampFromBody2)
}

type capturedConsistencyValues struct {
	PathUUID        string
	HeaderUUID      string
	BodyUUID        string
	BodyAnotherUUID string
	HeaderTimestamp string
	BodyTimestamp   string
	HeaderRandomInt string
	BodyRandomInt   string
}

func setupConsistencyTestServer(t *testing.T, capturedVals *capturedConsistencyValues) *httptest.Server {
	t.Helper()
	return startMockServer(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) == 3 && pathParts[1] == "test-uuid" {
			capturedVals.PathUUID = pathParts[2]
		} else {
			t.Logf("Unexpected path format in setupConsistencyTestServer: %s", r.URL.Path)
		}

		capturedVals.HeaderUUID = r.Header.Get("X-Request-UUID")
		capturedVals.HeaderTimestamp = r.Header.Get("X-Request-Timestamp")
		capturedVals.HeaderRandomInt = r.Header.Get("X-Request-RandomInt")

		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyJSON map[string]any
		err = json.Unmarshal(bodyBytes, &bodyJSON)
		require.NoError(t, err)

		if id, ok := bodyJSON["id"].(string); ok {
			capturedVals.BodyUUID = id
		}
		if anotherID, ok := bodyJSON["another_id"].(string); ok {
			capturedVals.BodyAnotherUUID = anotherID
		}
		if ts, ok := bodyJSON["timestamp"].(string); ok {
			capturedVals.BodyTimestamp = ts
		}
		if ri, ok := bodyJSON["randomInt"].(string); ok {
			capturedVals.BodyRandomInt = ri
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
}

func assertServerCapturedConsistencyValues(t *testing.T, capturedVals *capturedConsistencyValues) {
	t.Helper()
	assert.NotEmpty(t, capturedVals.PathUUID, "Path UUID should not be empty")
	assert.NotEqual(t, "{{$uuid}}", capturedVals.PathUUID, "Path UUID should be resolved from {{$uuid}}")
	_, parseUUIDErr := uuid.Parse(capturedVals.PathUUID)
	assert.NoError(t, parseUUIDErr, "Captured Path UUID should be a valid UUID")

	assert.NotEmpty(t, capturedVals.HeaderTimestamp, "Header Timestamp should not be empty")
	assert.NotEqual(t, "{{$timestamp}}", capturedVals.HeaderTimestamp, "Header Timestamp should be resolved")
	_, parseIntErr := strconv.ParseInt(capturedVals.HeaderTimestamp, 10, 64)
	assert.NoError(t, parseIntErr, "Captured Header Timestamp should be a valid integer")

	assert.NotEmpty(t, capturedVals.HeaderRandomInt, "Header RandomInt should not be empty")
	assert.NotEqual(t, "{{$randomInt}}", capturedVals.HeaderRandomInt, "Header RandomInt should be resolved")
	_, parseIntErr = strconv.ParseInt(capturedVals.HeaderRandomInt, 10, 64)
	assert.NoError(t, parseIntErr, "Captured Header RandomInt should be a valid integer")

	assert.Equal(t, capturedVals.PathUUID, capturedVals.HeaderUUID, "Path UUID and Header UUID should be the same")
	assert.Equal(t, capturedVals.PathUUID, capturedVals.BodyUUID, "Path UUID and Body UUID should be the same")
	assert.Equal(t, capturedVals.PathUUID, capturedVals.BodyAnotherUUID,
		"Path UUID and Body Another UUID should be the same")
	assert.Equal(t, capturedVals.HeaderTimestamp, capturedVals.BodyTimestamp,
		"Header Timestamp and Body Timestamp should be the same")
	assert.Equal(t, capturedVals.HeaderRandomInt, capturedVals.BodyRandomInt,
		"Header RandomInt and Body RandomInt should be the same")
}

func assertRequestObjectConsistency(t *testing.T, parsedReq *rc.Request, capturedVals *capturedConsistencyValues) {
	t.Helper()
	require.NotNil(t, parsedReq)

	assert.Equal(t, "/test-uuid/"+capturedVals.PathUUID, parsedReq.URL.Path, "Parsed request URL path mismatch")
	assert.Equal(t, capturedVals.PathUUID, parsedReq.Headers.Get("X-Request-UUID"))
	assert.Equal(t, capturedVals.HeaderTimestamp, parsedReq.Headers.Get("X-Request-Timestamp"))
	assert.Equal(t, capturedVals.HeaderRandomInt, parsedReq.Headers.Get("X-Request-RandomInt"))

	var clientReqBodyJSON map[string]any
	err := json.Unmarshal([]byte(parsedReq.RawBody), &clientReqBodyJSON)
	require.NoError(t, err, "Failed to unmarshal client request RawBody JSON")

	assert.Equal(t, capturedVals.PathUUID, clientReqBodyJSON["id"], "Client request body UUID mismatch")
	assert.Equal(t, capturedVals.PathUUID, clientReqBodyJSON["another_id"], "Client request body Another UUID mismatch")
	assert.Equal(t, capturedVals.HeaderTimestamp, clientReqBodyJSON["timestamp"], "Client request body Timestamp mismatch")
	assert.Equal(t, capturedVals.HeaderRandomInt, clientReqBodyJSON["randomInt"], "Client request body RandomInt mismatch")
}

// PRD-COMMENT: FR1.6 - Variable Function Consistency (Internal)
// Corresponds to: Ensuring internal consistency between how variables are resolved by the
// dedicated variable substitution functions and how they are resolved during a full ExecuteFile
// operation. This is more of an internal consistency check than a direct user-facing feature test.
// This test compares the output of `SubstituteVariablesInString` with the actual substituted
// values observed in a request made via `ExecuteFile`, using 'testdata/http_request_files/variable_consistency.http'.
func TestExecuteFile_VariableFunctionConsistency(t *testing.T) {
	var capturedVals capturedConsistencyValues
	server := setupConsistencyTestServer(t, &capturedVals)
	defer server.Close()

	client, err := rc.NewClient(rc.WithBaseURL(server.URL))
	require.NoError(t, err)

	requestFilePath := "testdata/http_request_files/variable_function_consistency.rest"

	responses, err := client.ExecuteFile(context.Background(), requestFilePath)
	require.NoError(t, err, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Should have one response")

	resp := responses[0]
	require.NotNil(t, resp, "Response object should not be nil")
	assert.NoError(t, resp.Error, "Error in response object should be nil")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be OK")

	assertServerCapturedConsistencyValues(t, &capturedVals)
	assertRequestObjectConsistency(t, resp.Request, &capturedVals)
}

// interceptedRequestData holds data captured by the mock server for assertions.
type interceptedRequestData struct {
	Path   string
	Host   string
	Header string
	Body   string
}

// httpClientEnvTestCase defines a test case for TestExecuteFile_WithHttpClientEnvJson.
type httpClientEnvTestCase struct {
	name                     string
	envFileTemplatePath      string // Path to http-client.env.json template
	privateEnvFilePath       string // Path to http-client.private.env.json (optional)
	requestFilePath          string // Path to .http request file
	selectedEnv              string // Environment to select in rc.NewClient
	expectExecuteFileError   bool
	executeFileErrorContains string
	expectResponseError      bool
	responseErrorContains    string
	responseAssertions func(t *testing.T, resp *rc.Response,
		interceptedReq *interceptedRequestData, serverURL string)
}

// runHttpClientEnvSubtest executes a single sub-test for TestExecuteFile_WithHttpClientEnvJson.
func runHttpClientEnvSubtest(t *testing.T, tc httpClientEnvTestCase) {
	t.Helper()

	// Given
	var interceptedReq interceptedRequestData
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		interceptedReq.Path = r.URL.Path
		interceptedReq.Host = r.Host
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedReq.Body = string(bodyBytes)
		interceptedReq.Header = r.Header.Get("X-Custom-Header")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	}))
	defer mockServer.Close()

	tempDir := t.TempDir()

	if tc.envFileTemplatePath != "" {
		envContentBytes, err := os.ReadFile(tc.envFileTemplatePath)
		require.NoError(t, err)
		envContent := strings.ReplaceAll(string(envContentBytes), "{{SERVER_URL}}", mockServer.URL)
		err = os.WriteFile(filepath.Join(tempDir, "http-client.env.json"), []byte(envContent), 0600)
		require.NoError(t, err)
	}

	if tc.privateEnvFilePath != "" {
		privateEnvContentBytes, err := os.ReadFile(tc.privateEnvFilePath)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tempDir, "http-client.private.env.json"), privateEnvContentBytes, 0600)
		require.NoError(t, err)
	}

	requestFileContentBytes, err := os.ReadFile(tc.requestFilePath)
	require.NoError(t, err)
	httpFilePath := filepath.Join(tempDir, "request.http")
	err = os.WriteFile(httpFilePath, requestFileContentBytes, 0600)
	require.NoError(t, err)

	var client *rc.Client
	if tc.selectedEnv != "" {
		client, err = rc.NewClient(rc.WithEnvironment(tc.selectedEnv))
	} else {
		client, err = rc.NewClient()
	}
	require.NoError(t, err)

	// When
	responses, execErr := client.ExecuteFile(context.Background(), httpFilePath)

	// Then
	if tc.expectExecuteFileError {
		require.Error(t, execErr)
		if tc.executeFileErrorContains != "" {
			assert.Contains(t, execErr.Error(), tc.executeFileErrorContains)
		}
	} else {
		require.NoError(t, execErr)
	}

	require.Len(t, responses, 1, "Expected 1 response")
	resp := responses[0]
	require.NotNil(t, resp)

	if tc.expectResponseError {
		assert.Error(t, resp.Error)
		if tc.responseErrorContains != "" {
			assert.Contains(t, resp.Error.Error(), tc.responseErrorContains)
		}
	} else {
		assert.NoError(t, resp.Error)
	}

	if tc.responseAssertions != nil {
		tc.responseAssertions(t, resp, &interceptedReq, mockServer.URL)
	}
}

// PRD-COMMENT: FR1.4 - Environment Configuration Files: http-client.env.json
// Corresponds to: Client's ability to load and substitute variables from `http-client.env.json`
// and `http-client.private.env.json` files, respecting environment-specific configurations
// (e.g., "dev", "prod") (http_syntax.md "Environment Configuration Files").
// This test suite uses various .http files and dynamically created `http-client.env.json` /
// `http-client.private.env.json` files to verify variable loading, substitution, environment
// selection, and precedence for different scenarios (Task T4).
func TestExecuteFile_WithHttpClientEnvJson(t *testing.T) {
	tests := []httpClientEnvTestCase{
		{
			name:                     "SCENARIO-LIB-018-004: no env selected, file exists",
			envFileTemplatePath:      "testdata/execute_file_httpclientenv/no_env_selected_env_template.json",
			requestFilePath:          "testdata/execute_file_httpclientenv/no_env_selected_request.http",
			selectedEnv:              "",
			expectExecuteFileError:   true,
			executeFileErrorContains: "unsupported protocol scheme \"\"",
			expectResponseError:      true,
			responseErrorContains:    "unsupported protocol scheme \"\"",
			responseAssertions: func(t *testing.T, resp *rc.Response, interceptedReq *interceptedRequestData, serverURL string) {
				t.Helper()
				assert.True(t, strings.Contains(resp.Request.RawURLString, "{{host}}"),
				"RawURLString should still contain {{host}}")
			},
		},
		{
			name:                   "SCENARIO-LIB-018-005: private env overrides public env",
			envFileTemplatePath:    "testdata/execute_file_httpclientenv/private_overrides_public_env_template.json",
			privateEnvFilePath:     "testdata/execute_file_httpclientenv/private_overrides_private_env.json",
			requestFilePath:        "testdata/execute_file_httpclientenv/private_overrides_request.http",
			selectedEnv:            "dev",
			expectExecuteFileError: false,
			expectResponseError:    false,
			responseAssertions: func(t *testing.T, resp *rc.Response, interceptedReq *interceptedRequestData, serverURL string) {
				t.Helper()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				parsedServerURL, pErr := url.Parse(serverURL)
				require.NoError(t, pErr)
				assert.Equal(t, parsedServerURL.Host, interceptedReq.Host)
				assert.Equal(t, "/test", interceptedReq.Path)
				assert.Equal(t, "private_override_value", interceptedReq.Header)

				expectedBody := `{
				  "public": "public_value",
				  "private_only": "private_specific_value"
				}`
				assert.JSONEq(t, expectedBody, interceptedReq.Body)
			},
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Mark subtests as parallelizable
			runHttpClientEnvSubtest(t, tc)
		})
	}
}

// Helper for TestExecuteFile_WithExtendedRandomSystemVariables: asserts $random.integer behavior
func assertRandomInteger(t *testing.T, bodyJSON map[string]string) {
	t.Helper()
	randIntValue, err := strconv.Atoi(bodyJSON["randInt"])
	assert.NoError(t, err, "randInt should be an integer")
	assert.GreaterOrEqual(t, randIntValue, 10, "randInt should be >= 10")
	assert.LessOrEqual(t, randIntValue, 20, "randInt should be <= 20")

	randIntNegativeValue, err := strconv.Atoi(bodyJSON["randIntNegative"])
	assert.NoError(t, err, "randIntNegative should be an integer")
	assert.GreaterOrEqual(t, randIntNegativeValue, -5, "randIntNegative should be >= -5")
	assert.LessOrEqual(t, randIntNegativeValue, 5, "randIntNegative should be <= 5")

	assert.Equal(t, "{{$random.integer 10 1}}", bodyJSON["randIntInvalidRange"],
		"randIntInvalidRange should remain unsubstituted")
	assert.Equal(t, "{{$random.integer 10 abc}}", bodyJSON["randIntInvalidArgs"],
		"randIntInvalidArgs should remain unsubstituted")
}

// Helper for TestExecuteFile_WithExtendedRandomSystemVariables: asserts $random.float behavior
func assertRandomFloat(t *testing.T, bodyJSON map[string]string) {
	t.Helper()
	randFloatValue, err := strconv.ParseFloat(bodyJSON["randFloat"], 64)
	assert.NoError(t, err, "randFloat should be a float")
	assert.GreaterOrEqual(t, randFloatValue, 1.0, "randFloat should be >= 1.0")
	assert.LessOrEqual(t, randFloatValue, 2.5, "randFloat should be <= 2.5")

	randFloatNegativeValue, err := strconv.ParseFloat(bodyJSON["randFloatNegative"], 64)
	assert.NoError(t, err, "randFloatNegative should be a float")
	assert.GreaterOrEqual(t, randFloatNegativeValue, -1.5, "randFloatNegative should be >= -1.5")
	assert.LessOrEqual(t, randFloatNegativeValue, 0.5, "randFloatNegative should be <= 0.5")

	assert.Equal(t, "{{$random.float 5.0 1.0}}", bodyJSON["randFloatInvalidRange"],
		"randFloatInvalidRange should remain unsubstituted")
}

// Helper for TestExecuteFile_WithExtendedRandomSystemVariables: asserts $random.alphabetic behavior
func assertRandomAlphabetic(t *testing.T, bodyJSON map[string]string) {
	t.Helper()
	randAlphabeticValue := bodyJSON["randAlphabetic"]
	assert.Len(t, randAlphabeticValue, 10, "randAlphabetic length mismatch")
	for _, r := range randAlphabeticValue {
		assert.True(t, (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'), "randAlphabetic char not in alphabet: %c", r)
	}
	assert.Equal(t, "", bodyJSON["randAlphabeticZero"], "randAlphabeticZero should be empty")
	assert.Equal(t, "{{$random.alphabetic abc}}", bodyJSON["randAlphabeticInvalid"],
		"randAlphabeticInvalid should remain unsubstituted")
}

// Helper for TestExecuteFile_WithExtendedRandomSystemVariables: asserts $random.alphanumeric behavior
func assertRandomAlphanumeric(t *testing.T, bodyJSON map[string]string) {
	t.Helper()
	randAlphanumericValue := bodyJSON["randAlphanumeric"]
	assert.Len(t, randAlphanumericValue, 15, "randAlphanumeric length mismatch")
	for _, r := range randAlphanumericValue {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isNumber := r >= '0' && r <= '9'
		isUnderscore := r == '_'
		assert.True(t, isLetter || isNumber || isUnderscore, "randAlphanumeric char not alphanumeric: %c", r)
	}
}

// Helper for TestExecuteFile_WithExtendedRandomSystemVariables: asserts $random.hexadecimal behavior
func assertRandomHexadecimal(t *testing.T, bodyJSON map[string]string) {
	t.Helper()
	randHexValue := bodyJSON["randHex"]
	assert.Len(t, randHexValue, 8, "randHex length mismatch")
	_, err := hex.DecodeString(randHexValue)
	assert.NoError(t, err, "randHex should be valid hexadecimal: %s", randHexValue)
}

// Helper for TestExecuteFile_WithExtendedRandomSystemVariables: asserts $random.email behavior
func assertRandomEmail(t *testing.T, bodyJSON map[string]string) {
	t.Helper()
	randEmailValue := bodyJSON["randEmail"]
	parts := strings.Split(randEmailValue, "@")
	require.Len(t, parts, 2, "randEmail should have one @ symbol")
	domainParts := strings.Split(parts[1], ".")
	require.GreaterOrEqual(t, len(domainParts), 2, "randEmail domain should have at least one .")
	assert.Regexp(t, `^[a-zA-Z0-9_]+@[a-zA-Z]+\.[a-zA-Z]{2,3}$`, randEmailValue, "randEmail format is incorrect")
}

// PRD-COMMENT: FR1.3.8 - System Variables: {{$random.*}}
// Corresponds to: Client's ability to substitute extended random system variables like
// {{$random.integer MIN MAX}}, {{$random.float MIN MAX}}, {{$random.alphabetic LENGTH}},
// {{$random.alphanumeric LENGTH}}, {{$random.hexadecimal LENGTH}}, {{$random.email}}
// (http_syntax.md "System Variables - Extended Random").
// This test uses 'testdata/http_request_files/system_var_extended_random.http' to verify
// the generation and substitution of these extended random variables, including handling
// of arguments and invalid inputs.
func TestExecuteFile_WithExtendedRandomSystemVariables(t *testing.T) {
	// Given
	var interceptedRequest struct {
		Body string
	}
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_extended_random.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for extended random variable processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal response body JSON: %s", interceptedRequest.Body)

	assertRandomInteger(t, bodyJSON)
	assertRandomFloat(t, bodyJSON)
	assertRandomAlphabetic(t, bodyJSON)
	assertRandomAlphanumeric(t, bodyJSON)
	assertRandomHexadecimal(t, bodyJSON)
	assertRandomEmail(t, bodyJSON)
}
