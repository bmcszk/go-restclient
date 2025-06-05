package restclient

import (
	"context"
	"encoding/hex" // Added for TestExecuteFile_WithExtendedRandomSystemVariables
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRandomStringFromCharset tests the randomStringFromCharset helper function.
func TestRandomStringFromCharset(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		charset  string
		wantLen  int
		assertFn func(t *testing.T, s string, charset string)
	}{
		{
			name:    "alphabetic_10",
			length:  10,
			charset: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
			wantLen: 10,
			assertFn: func(t *testing.T, s string, charset string) {
				for _, r := range s {
					assert.Contains(t, charset, string(r))
				}
			},
		},
		{
			name:    "numeric_5",
			length:  5,
			charset: "0123456789",
			wantLen: 5,
			assertFn: func(t *testing.T, s string, charset string) {
				for _, r := range s {
					assert.Contains(t, charset, string(r))
				}
			},
		},
		{
			name:    "empty_charset",
			length:  5,
			charset: "",
			wantLen: 0,
		},
		{
			name:    "zero_length",
			length:  0,
			charset: "abc",
			wantLen: 0,
			assertFn: func(t *testing.T, s string, charset string) {
			},
		},
		{
			name:    "negative_length",
			length:  -5,
			charset: "abc",
			wantLen: 0,
			assertFn: func(t *testing.T, s string, charset string) {
			},
		},
	}

	for _, testCase := range tests {
		capturedTC := testCase // Explicitly capture the current test case
		t.Run(capturedTC.name, func(t *testing.T) {
			if capturedTC.charset == "" && capturedTC.length > 0 {
				s := randomStringFromCharset(capturedTC.length, capturedTC.charset)
				assert.Len(t, s, capturedTC.wantLen, "String length mismatch")

			} else {
				s := randomStringFromCharset(capturedTC.length, capturedTC.charset)
				assert.Len(t, s, capturedTC.wantLen, "String length mismatch")
				if capturedTC.assertFn != nil && capturedTC.wantLen > 0 {
					capturedTC.assertFn(t, s, capturedTC.charset)
				}
			}
		})
	}
}

// TestSubstituteDynamicSystemVariables_EnvVars tests the {{$env.VAR_NAME}} substitution.
func TestSubstituteDynamicSystemVariables_EnvVars(t *testing.T) {
	client, _ := NewClient()
	tests := []struct {
		name    string
		input   string
		setup   func(t *testing.T) // For setting env vars
		want    string
		wantErr bool // If we expect a parsing warning (though not directly testable here without log capture)
	}{
		{
			name:  "existing env var",
			input: "Hello {{$env.MY_TEST_VAR}}!",
			setup: func(t *testing.T) { t.Setenv("MY_TEST_VAR", "World") },
			want:  "Hello World!",
		},
		{
			name:  "non-existing env var",
			input: "Value: {{$env.NON_EXISTENT_VAR}}",
			setup: func(t *testing.T) {},
			want:  "Value: ",
		},
		{
			name:  "multiple env vars",
			input: "{{$env.FIRST_VAR}} and {{$env.SECOND_VAR}}",
			setup: func(t *testing.T) {
				t.Setenv("FIRST_VAR", "Apple")
				t.Setenv("SECOND_VAR", "Banana")
			},
			want: "Apple and Banana",
		},
		{
			name:  "env var with underscore and numbers",
			input: "{{$env.MY_VAR_123}}",
			setup: func(t *testing.T) { t.Setenv("MY_VAR_123", "Test123") },
			want:  "Test123",
		},
		{
			name:  "empty env var value",
			input: "Prefix{{$env.EMPTY_VAR}}Suffix",
			setup: func(t *testing.T) { t.Setenv("EMPTY_VAR", "") },
			want:  "PrefixSuffix",
		},
		{
			name:    "malformed - no var name",
			input:   "{{$env.}}",
			setup:   func(t *testing.T) {},
			want:    "{{$env.}}",
			wantErr: true, // Expect original match due to regex non-match or parse fail
		},
		{
			name:    "malformed - invalid char in var name",
			input:   "{{$env.MY-VAR}}", // Hyphen is not allowed by regex
			setup:   func(t *testing.T) { t.Setenv("MY-VAR", "ShouldNotBeUsed") },
			want:    "{{$env.MY-VAR}}",
			wantErr: true,
		},
		{
			name:  "var name starting with underscore",
			input: "{{$env._MY_VAR}}",
			setup: func(t *testing.T) { t.Setenv("_MY_VAR", "StartsWithUnderscore") },
			want:  "StartsWithUnderscore",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t) // Set/unset env vars for this test case
			output := client.substituteDynamicSystemVariables(tc.input, client.currentDotEnvVars)
			assert.Equal(t, tc.want, output)
			// Note: Testing for slog.Warn would require log capture, which is out of scope here.
			// We rely on the fact that if tc.wantErr is true, the output should be the original input.
		})
	}
}

func TestExecuteFile_WithCustomVariables(t *testing.T) {
	// Given
	var requestCount int32
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		currentCount := atomic.AddInt32(&requestCount, 1)
		t.Logf("Mock server received request #%d: %s %s", currentCount, r.Method, r.URL.Path)
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
		case "/items/{{undefined_path_var}}": // SCENARIO-LIB-013-005 (undefined variable left as-is in path)
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response for items (undefined_path_var)")
		default:
			t.Errorf("Unexpected request path to mock server: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/custom_variables.http", struct{ ServerURL string }{ServerURL: server.URL})

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
	assert.Equal(t, "response for items (undefined_path_var)", resp3.BodyString)
}

func TestExecuteFile_WithProcessEnvSystemVariable(t *testing.T) {
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

	client, _ := NewClient()
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
	assert.Equal(t, testEnvVarValue, interceptedRequest.Header, "X-Env-Value header should contain substituted env variable")

	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal request body JSON")

	envPayload, ok := bodyJSON["env_payload"]
	require.True(t, ok, "env_payload not found in body")
	assert.Equal(t, testEnvVarValue, envPayload, "Body env_payload should contain substituted env variable")

	// SCENARIO-LIB-019-002
	undefinedPayload, ok := bodyJSON["undefined_payload"]
	require.True(t, ok, "undefined_payload not found in body")
	assert.Equal(t, fmt.Sprintf("{{$processEnv %s}}", undefinedEnvVarName), undefinedPayload, "Body undefined_payload should be the unresolved placeholder")

	// Check Cache-Control header for unresolved placeholder
	assert.Equal(t, "{{$processEnv UNDEFINED_CACHE_VAR_SHOULD_BE_EMPTY}}", interceptedRequest.CacheControlHeader, "Cache-Control header should be the unresolved placeholder")
}

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

	requestFileContent1 := fmt.Sprintf(`
GET %s/path-{{$dotenv DOTENV_VAR1}}/data
Content-Type: application/json
X-Dotenv-Value: {{$dotenv DOTENV_VAR2}}

{
  "payload": "{{$dotenv DOTENV_VAR1}}",
  "missing_payload": "{{$dotenv MISSING_DOTENV_VAR}}"
}
`, server.URL)
	httpFile1Path := filepath.Join(tempDir, "request1.http")
	err = os.WriteFile(httpFile1Path, []byte(requestFileContent1), 0644)
	require.NoError(t, err)

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

	requestFileContent2 := fmt.Sprintf(`
GET %s/path-{{$dotenv DOTENV_VAR_SHOULD_BE_EMPTY}}/data
User-Agent: test-client

{
  "payload": "{{$dotenv DOTENV_VAR_ALSO_EMPTY}}"
}
`, server.URL)
	httpFile2Path := filepath.Join(tempDir, "request2.http")
	err = os.WriteFile(httpFile2Path, []byte(requestFileContent2), 0644)
	require.NoError(t, err)

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

	clientProgrammaticVars := map[string]interface{}{
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

	client, err := NewClient(WithVars(clientProgrammaticVars))
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
	assert.Equal(t, "overridden_by_programmatic", bodyJSON["overridden_file_var"], "Body field 'overridden_file_var' mismatch")
	assert.Equal(t, "programmatic_wins_over_env", bodyJSON["env_var_check"], "Body field 'env_var_check' mismatch")
	assert.Equal(t, "file_only", bodyJSON["file_only_check"], "Body field 'file_only_check' mismatch")

	// Also check headers received by the server for variable substitution confirmation
	// These were set up in the new programmatic_variables.http file to check different sources
	assert.Equal(t, "overridden_by_programmatic", resp.Request.Headers.Get("X-File-Var"))
	assert.Equal(t, "programmatic_wins_over_env", resp.Request.Headers.Get("X-Env-Var"))
	assert.Equal(t, "file_only", resp.Request.Headers.Get("X-Unused-File-Var"))
}

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

	client, _ := NewClient()

	// Capture current time to compare against, allowing for slight delay
	beforeTime := time.Now().UTC().Unix()

	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_timestamp.http", struct{ ServerURL string }{ServerURL: server.URL})

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

func TestExecuteFile_VariableFunctionConsistency(t *testing.T) {
	// This server will capture the path, headers, and body to check for consistency.
	var capturedPathUUID, capturedHeaderUUID, capturedBodyUUID, capturedBodyAnotherUUID string
	var capturedHeaderTimestamp, capturedBodyTimestamp string
	var capturedHeaderRandomInt, capturedBodyRandomInt string

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) == 3 && pathParts[1] == "test-uuid" {
			capturedPathUUID = pathParts[2]
		} else {
			t.Logf("Unexpected path format: %s", r.URL.Path)
		}

		capturedHeaderUUID = r.Header.Get("X-Request-UUID")
		capturedHeaderTimestamp = r.Header.Get("X-Request-Timestamp")
		capturedHeaderRandomInt = r.Header.Get("X-Request-RandomInt")

		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyJSON map[string]interface{}
		err = json.Unmarshal(bodyBytes, &bodyJSON)
		require.NoError(t, err)

		if id, ok := bodyJSON["id"].(string); ok {
			capturedBodyUUID = id
		}
		if anotherID, ok := bodyJSON["another_id"].(string); ok {
			capturedBodyAnotherUUID = anotherID
		}
		if ts, ok := bodyJSON["timestamp"].(string); ok {
			capturedBodyTimestamp = ts
		}
		if ri, ok := bodyJSON["randomInt"].(string); ok {
			capturedBodyRandomInt = ri
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL)) // Set BaseURL to mock server
	require.NoError(t, err)

	requestFilePath := "testdata/http_request_files/variable_function_consistency.rest"

	responses, err := client.ExecuteFile(context.Background(), requestFilePath)
	require.NoError(t, err, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Should have one response")

	resp := responses[0]
	require.NotNil(t, resp, "Response object should not be nil")
	assert.NoError(t, resp.Error, "Error in response object should be nil")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be OK")

	// Assert that captured values are not the placeholders themselves
	assert.NotEmpty(t, capturedPathUUID, "Path UUID should not be empty")
	assert.NotEqual(t, "{{$uuid}}", capturedPathUUID, "Path UUID should be resolved from {{$uuid}}")
	_, parseUUIDErr := uuid.Parse(capturedPathUUID)
	assert.NoError(t, parseUUIDErr, "Captured Path UUID should be a valid UUID")

	assert.NotEmpty(t, capturedHeaderTimestamp, "Header Timestamp should not be empty")
	assert.NotEqual(t, "{{$timestamp}}", capturedHeaderTimestamp, "Header Timestamp should be resolved")
	_, parseIntErr := strconv.ParseInt(capturedHeaderTimestamp, 10, 64)
	assert.NoError(t, parseIntErr, "Captured Header Timestamp should be a valid integer")

	assert.NotEmpty(t, capturedHeaderRandomInt, "Header RandomInt should not be empty")
	assert.NotEqual(t, "{{$randomInt}}", capturedHeaderRandomInt, "Header RandomInt should be resolved")
	_, parseIntErr = strconv.ParseInt(capturedHeaderRandomInt, 10, 64) // Re-check, should be parsable as int
	assert.NoError(t, parseIntErr, "Captured Header RandomInt should be a valid integer")

	// Assert UUID consistency
	assert.Equal(t, capturedPathUUID, capturedHeaderUUID, "Path UUID and Header UUID should be the same")
	assert.Equal(t, capturedPathUUID, capturedBodyUUID, "Path UUID and Body UUID should be the same")
	assert.Equal(t, capturedPathUUID, capturedBodyAnotherUUID, "Path UUID and Body Another UUID should be the same")

	// Assert Timestamp consistency
	assert.Equal(t, capturedHeaderTimestamp, capturedBodyTimestamp, "Header Timestamp and Body Timestamp should be the same")

	// Assert RandomInt consistency
	assert.Equal(t, capturedHeaderRandomInt, capturedBodyRandomInt, "Header RandomInt and Body RandomInt should be the same")

	// Additionally, verify that the actual substituted values in the request object (client-side) are consistent.
	parsedReq := resp.Request
	require.NotNil(t, parsedReq)

	// Check substituted URL Path
	// capturedPathUUID is what the server received and should be the actual resolved UUID.
	assert.Equal(t, "/test-uuid/"+capturedPathUUID, parsedReq.URL.Path, "Parsed request URL path mismatch")

	// Check substituted Header
	assert.Equal(t, capturedPathUUID, parsedReq.Headers.Get("X-Request-UUID"))
	assert.Equal(t, capturedHeaderTimestamp, parsedReq.Headers.Get("X-Request-Timestamp"))
	assert.Equal(t, capturedHeaderRandomInt, parsedReq.Headers.Get("X-Request-RandomInt"))

	// Check substituted Body
	var finalBodyJSON map[string]interface{}
	err = json.Unmarshal([]byte(parsedReq.RawBody), &finalBodyJSON)
	require.NoError(t, err, "Failed to parse RawBody as JSON")

	assert.Equal(t, capturedPathUUID, finalBodyJSON["id"].(string))
	assert.Equal(t, capturedPathUUID, finalBodyJSON["another_id"].(string))
	assert.Equal(t, capturedHeaderTimestamp, finalBodyJSON["timestamp"].(string))
	assert.Equal(t, capturedHeaderRandomInt, finalBodyJSON["randomInt"].(string))
}

// TestExecuteFile_WithHttpClientEnvJson tests variable substitution from http-client.env.json (Task T4)
func TestExecuteFile_WithHttpClientEnvJson(t *testing.T) {
	// SCENARIO-LIB-018-001: Env selected, http-client.env.json exists, env exists in file
	t.Run("env selected, file exists, env exists in file", func(t *testing.T) {
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
		envContent := `{
			"dev": {
				"host": "` + server.URL + `",
				"token": "dev-token",
				"user_id": "dev-user",
				"common_var": "env_common_dev"
			},
			"prod": {
				"host": "https://prod.example.com",
				"token": "prod-token",
				"user_id": "prod-user",
				"common_var": "env_common_prod"
			}
		}`
		envFilePath := filepath.Join(tempDir, "http-client.env.json")
		err := os.WriteFile(envFilePath, []byte(envContent), 0600)
		require.NoError(t, err)

		// Create request file
		requestFileContent := `
### Test Request with Env Vars
# @name testWithEnv
POST {{host}}/resource/{{user_id}}
Content-Type: application/json
X-Env-Var: {{token}}

{
  "message": "Hello from {{user_id}}",
  "common": "{{common_var}}"
}
`
		httpFilePath := filepath.Join(tempDir, "test_env_vars.http")
		err = os.WriteFile(httpFilePath, []byte(requestFileContent), 0600)
		require.NoError(t, err)

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
		expectedBody := `{
  "message": "Hello from dev-user",
  "common": "env_common_dev"
}`
		assert.JSONEq(t, expectedBody, interceptedRequest.Body)
		assert.Equal(t, "dev", client.selectedEnvironmentName) // Verify client has the env name
		// EnvironmentVariables are used internally; their effect is checked by the substituted values above.
	})

	// SCENARIO-LIB-018-002: Env selected, http-client.env.json exists, but env NOT in file
	t.Run("env selected, file exists, env NOT in file", func(t *testing.T) {
		// Given
		serverURL := "http://localhost:12345" // A dummy URL, server won't actually be hit with {{host}}
		server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
			// This handler might not be reached if {{host}} isn't resolved by any mechanism
			// and the HTTP client fails before sending.
			w.WriteHeader(http.StatusOK)
		})
		defer server.Close()

		tempDir := t.TempDir()
		envContent := `{
			"dev": {
				"host": "` + serverURL /* Use the dummy serverURL here for consistency */ + `",
				"token": "dev-token"
			}
		}`
		envFilePath := filepath.Join(tempDir, "http-client.env.json")
		err := os.WriteFile(envFilePath, []byte(envContent), 0600)
		require.NoError(t, err)

		requestFileContent := `GET {{host}}/path`
		httpFilePath := filepath.Join(tempDir, "test_env_vars_missing_env.http")
		err = os.WriteFile(httpFilePath, []byte(requestFileContent), 0600)
		require.NoError(t, err)

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
	})

	// SCENARIO-LIB-018-003: Env selected, but http-client.env.json does NOT exist
	t.Run("env selected, file does NOT exist", func(t *testing.T) {
		// Given
		server := startMockServer(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		defer server.Close()

		tempDir := t.TempDir()
		// http-client.env.json is NOT created in tempDir

		requestFileContent := `GET {{host}}/path`
		httpFilePath := filepath.Join(tempDir, "test_env_vars_no_env_file.http")
		err := os.WriteFile(httpFilePath, []byte(requestFileContent), 0600)
		require.NoError(t, err)

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
	})

	// SCENARIO-LIB-018-004: No env selected, http-client.env.json exists
	t.Run("no env selected, file exists", func(t *testing.T) {
		// Given
		serverURL := "http://localhost:54321"
		server := startMockServer(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		defer server.Close()

		tempDir := t.TempDir()
		envContent := `{
			"dev": {
				"host": "` + serverURL + `",
				"token": "dev-token"
			}
		}`
		envFilePath := filepath.Join(tempDir, "http-client.env.json")
		err := os.WriteFile(envFilePath, []byte(envContent), 0600)
		require.NoError(t, err)

		requestFileContent := `GET {{host}}/path`
		httpFilePath := filepath.Join(tempDir, "test_no_env_selected.http")
		err = os.WriteFile(httpFilePath, []byte(requestFileContent), 0600)
		require.NoError(t, err)

		client, err := NewClient() // No WithEnvironment option
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
		assert.Empty(t, client.selectedEnvironmentName, "selectedEnvironmentName should be empty")
		// EnvironmentVariables map on ParsedFile would be nil internally, effect is placeholder {{host}} remains.
	})

	// SCENARIO-LIB-018-005: Private env file overrides public env file
	t.Run("private env overrides public env", func(t *testing.T) {
		// Given
		var interceptedRequest struct {
			Path   string
			Host   string
			Header string
			Body   string
		}

		server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
			interceptedRequest.Path = r.URL.Path
			interceptedRequest.Host = r.Host
			bodyBytes, _ := io.ReadAll(r.Body)
			interceptedRequest.Body = string(bodyBytes)
			interceptedRequest.Header = r.Header.Get("X-Custom-Header")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		})
		defer server.Close()

		tempDir := t.TempDir()

		// Create http-client.env.json (public)
		publicEnvContent := `{
			"dev": {
				"host": "` + server.URL + `",
				"public_var": "public_value",
				"override_var": "public_override"
			}
		}`
		publicEnvFilePath := filepath.Join(tempDir, "http-client.env.json")
		err := os.WriteFile(publicEnvFilePath, []byte(publicEnvContent), 0600)
		require.NoError(t, err)

		// Create http-client.private.env.json (private)
		privateEnvContent := `{
			"dev": {
				"override_var": "private_override_value",
				"private_var": "private_specific_value"
			}
		}`
		privateEnvFilePath := filepath.Join(tempDir, "http-client.private.env.json")
		err = os.WriteFile(privateEnvFilePath, []byte(privateEnvContent), 0600)
		require.NoError(t, err)

		requestFileContent := `
### Test Private Env Override
GET {{host}}/test
Content-Type: application/json
X-Custom-Header: {{override_var}}

{
  "public": "{{public_var}}",
  "private_only": "{{private_var}}"
}
`
		httpFilePath := filepath.Join(tempDir, "test_private_override.http")
		err = os.WriteFile(httpFilePath, []byte(requestFileContent), 0600)
		require.NoError(t, err)

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

		parsedServerURL, pErr := url.Parse(server.URL)
		require.NoError(t, pErr)
		assert.Equal(t, parsedServerURL.Host, interceptedRequest.Host) // host from public.env
		assert.Equal(t, "/test", interceptedRequest.Path)
		assert.Equal(t, "private_override_value", interceptedRequest.Header) // override_var from private.env

		expectedBody := `{
  "public": "public_value",
  "private_only": "private_specific_value"
}`
		assert.JSONEq(t, expectedBody, interceptedRequest.Body)
		assert.Equal(t, "dev", client.selectedEnvironmentName)
	})
}

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

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_extended_random.http", struct{ ServerURL string }{ServerURL: server.URL})

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

	// $random.integer
	randIntValue, err := strconv.Atoi(bodyJSON["randInt"])
	assert.NoError(t, err, "randInt should be an integer")
	assert.GreaterOrEqual(t, randIntValue, 10, "randInt should be >= 10")
	assert.LessOrEqual(t, randIntValue, 20, "randInt should be <= 20")

	randIntNegativeValue, err := strconv.Atoi(bodyJSON["randIntNegative"])
	assert.NoError(t, err, "randIntNegative should be an integer")
	assert.GreaterOrEqual(t, randIntNegativeValue, -5, "randIntNegative should be >= -5")
	assert.LessOrEqual(t, randIntNegativeValue, 5, "randIntNegative should be <= 5")

	assert.Equal(t, "{{$random.integer 10 1}}", bodyJSON["randIntInvalidRange"], "randIntInvalidRange should remain unsubstituted")
	assert.Equal(t, "{{$random.integer 10 abc}}", bodyJSON["randIntInvalidArgs"], "randIntInvalidArgs should remain unsubstituted")

	// $random.float
	randFloatValue, err := strconv.ParseFloat(bodyJSON["randFloat"], 64)
	assert.NoError(t, err, "randFloat should be a float")
	assert.GreaterOrEqual(t, randFloatValue, 1.0, "randFloat should be >= 1.0")
	assert.LessOrEqual(t, randFloatValue, 2.5, "randFloat should be <= 2.5")

	randFloatNegativeValue, err := strconv.ParseFloat(bodyJSON["randFloatNegative"], 64)
	assert.NoError(t, err, "randFloatNegative should be a float")
	assert.GreaterOrEqual(t, randFloatNegativeValue, -1.5, "randFloatNegative should be >= -1.5")
	assert.LessOrEqual(t, randFloatNegativeValue, 0.5, "randFloatNegative should be <= 0.5")

	assert.Equal(t, "{{$random.float 5.0 1.0}}", bodyJSON["randFloatInvalidRange"], "randFloatInvalidRange should remain unsubstituted")

	// $random.alphabetic
	randAlphabeticValue := bodyJSON["randAlphabetic"]
	assert.Len(t, randAlphabeticValue, 10, "randAlphabetic length mismatch")
	for _, r := range randAlphabeticValue {
		assert.True(t, (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'), "randAlphabetic char not in alphabet: %c", r)
	}
	assert.Equal(t, "", bodyJSON["randAlphabeticZero"], "randAlphabeticZero should be empty")
	assert.Equal(t, "{{$random.alphabetic abc}}", bodyJSON["randAlphabeticInvalid"], "randAlphabeticInvalid should remain unsubstituted")

	// $random.alphanumeric
	randAlphanumericValue := bodyJSON["randAlphanumeric"]
	assert.Len(t, randAlphanumericValue, 15, "randAlphanumeric length mismatch")
	for _, r := range randAlphanumericValue {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isNumber := r >= '0' && r <= '9'
		isUnderscore := r == '_'
		assert.True(t, isLetter || isNumber || isUnderscore, "randAlphanumeric char not alphanumeric: %c", r)
	}

	// $random.hexadecimal
	randHexValue := bodyJSON["randHex"]
	assert.Len(t, randHexValue, 8, "randHex length mismatch")
	_, err = hex.DecodeString(randHexValue)
	assert.NoError(t, err, "randHex should be valid hexadecimal: %s", randHexValue)

	// $random.email
	randEmailValue := bodyJSON["randEmail"]
	parts := strings.Split(randEmailValue, "@")
	require.Len(t, parts, 2, "randEmail should have one @ symbol")
	domainParts := strings.Split(parts[1], ".")
	require.GreaterOrEqual(t, len(domainParts), 2, "randEmail domain should have at least one .")
	assert.Regexp(t, `^[a-zA-Z0-9_]+@[a-zA-Z]+\.[a-zA-Z]{2,3}$`, randEmailValue, "randEmail format is incorrect")
}

// createTempHTTPFileFromString creates a temporary .http file with the given content.
// It returns the path to the file and registers a cleanup function to remove the temp directory.
func createTempHTTPFileFromString(t *testing.T, content string) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "test-http-inplace-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	filePath := filepath.Join(tempDir, "test.http")
	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

func TestExecuteFile_InPlaceVariables(t *testing.T) {
	t.Run("simple_variable_in_url", func(t *testing.T) {
		// Given
		var capturedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		httpFileContent := fmt.Sprintf(`
@hostname = %s
@path_segment = /api/v1/items

GET {{hostname}}{{path_segment}}/123
`, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, httpFileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When
		responses, err := client.ExecuteFile(context.Background(), requestFilePath)

		// Then
		require.NoError(t, err, "ExecuteFile should not return an error")
		require.Len(t, responses, 1, "Expected 1 response")

		resp := responses[0]
		assert.NoError(t, resp.Error, "Response error should be nil")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")

		// Verify the server received the request with the substituted URL
		expectedPath := "/api/v1/items/123"
		assert.Equal(t, expectedPath, capturedPath, "Captured path by server mismatch")

		// Verify ParsedFile.FileVariables
		parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
		require.NoError(t, pErr)
		require.NotNil(t, parsedFile.FileVariables)
		assert.Equal(t, server.URL, parsedFile.FileVariables["hostname"])
		assert.Equal(t, "/api/v1/items", parsedFile.FileVariables["path_segment"])
	})

	t.Run("variable_in_header", func(t *testing.T) {
		// Given
		var capturedHeaders http.Header
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeaders = r.Header
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		httpFileContent := fmt.Sprintf(`
@auth_token = Bearer_secret_token_123

GET %s/checkheaders
Authorization: {{auth_token}}
User-Agent: test-client
`, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, httpFileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When
		responses, err := client.ExecuteFile(context.Background(), requestFilePath)

		// Then
		require.NoError(t, err, "ExecuteFile should not return an error")
		require.Len(t, responses, 1, "Expected 1 response")

		resp := responses[0]
		assert.NoError(t, resp.Error, "Response error should be nil")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")

		// Verify the server received the request with the substituted header
		assert.Equal(t, "Bearer_secret_token_123", capturedHeaders.Get("Authorization"))
		assert.Equal(t, "test-client", capturedHeaders.Get("User-Agent")) // Ensure other headers are preserved

		// Verify ParsedFile.FileVariables
		parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
		require.NoError(t, pErr)
		require.NotNil(t, parsedFile.FileVariables)
		assert.Equal(t, "Bearer_secret_token_123", parsedFile.FileVariables["auth_token"])
	})

	t.Run("variable_in_body", func(t *testing.T) {
		// Given
		var capturedBody []byte
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			capturedBody, err = io.ReadAll(r.Body)
			require.NoError(t, err)
			defer r.Body.Close()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"created"}`))
		}))
		defer server.Close()

		httpFileContent := fmt.Sprintf(`
@product_name = SuperWidget
@product_id = SW1000
@product_price = 49.99

POST %s/products
Content-Type: application/json

{
  "id": "{{product_id}}",
  "name": "{{product_name}}",
  "price": {{product_price}}
}
`, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, httpFileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When
		responses, err := client.ExecuteFile(context.Background(), requestFilePath)

		// Then
		require.NoError(t, err, "ExecuteFile should not return an error")
		require.Len(t, responses, 1, "Expected 1 response")

		resp := responses[0]
		assert.NoError(t, resp.Error, "Response error should be nil")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")

		// Verify the server received the request with the substituted body
		expectedBodyJSON := `{
  "id": "SW1000",
  "name": "SuperWidget",
  "price": 49.99
}`
		assert.JSONEq(t, expectedBodyJSON, string(capturedBody), "Captured body by server mismatch")

		// Verify ParsedFile.FileVariables
		parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
		require.NoError(t, pErr)
		require.NotNil(t, parsedFile.FileVariables)
		assert.Equal(t, "SuperWidget", parsedFile.FileVariables["product_name"])
		assert.Equal(t, "SW1000", parsedFile.FileVariables["product_id"])
		assert.Equal(t, "49.99", parsedFile.FileVariables["product_price"])
	})

	t.Run("variable_defined_by_another_variable", func(t *testing.T) {
		// Given
		var capturedURL string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedURL = r.URL.String() // Captures path and query
			// Serve response based on expected.hresp
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		requestFilePath := "testdata/execute_inplace_vars/variable_defined_by_another_variable/request.http"
		expectedHrespPath := "testdata/execute_inplace_vars/variable_defined_by_another_variable/expected.hresp"

		client, err := NewClient()
		require.NoError(t, err)

		// Set the mock server URL as a programmatic variable
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
		assert.Equal(t, expectedHeaders.Get("Content-Type"), resp.Headers.Get("Content-Type"), "Content-Type mismatch")

		if expectedBodyStr != "" {
			assert.JSONEq(t, expectedBodyStr, string(resp.Body), "Response body mismatch")
		} else {
			assert.Empty(t, string(resp.Body), "Response body should be empty")
		}

		// Verify the server received the request at the correctly resolved URL
		// The request.http is: GET {{full_url}}?check_base={{base_url}}&check_path={{path}}
		// {{test_server_url}} is programmatically set to server.URL
		// {{base_url}} becomes {{test_server_url}} -> server.URL
		// {{path}} becomes /api/v2/items
		// {{full_url}} becomes {{base_url}}{{path}} -> server.URL/api/v2/items
		// So, capturedURL should be /api/v2/items?check_base=server.URL&check_path=/api/v2/items
		expectedPathAndQuery := fmt.Sprintf("/api/v2/items?check_base=%s&check_path=/api/v2/items", server.URL)
		assert.Equal(t, expectedPathAndQuery, capturedURL, "Captured URL by server mismatch")

		// Verify ParsedFile.FileVariables (should store raw definitions from the .http file)
		parsedFile, pErr := parseRequestFile(requestFilePath, client, make([]string, 0))
		require.NoError(t, pErr)
		require.NotNil(t, parsedFile.FileVariables)
		assert.Equal(t, "http://placeholder.com", parsedFile.FileVariables["test_server_url"], "File variable test_server_url mismatch")
		assert.Equal(t, "{{test_server_url}}", parsedFile.FileVariables["base_url"], "File variable base_url mismatch")
		assert.Equal(t, "/api/v2/items", parsedFile.FileVariables["path"], "File variable path mismatch")
		assert.Equal(t, "{{base_url}}{{path}}", parsedFile.FileVariables["full_url"], "File variable full_url mismatch")
	})

	t.Run("variable_precedence_over_environment", func(t *testing.T) {
		// Given: an .http file with an in-place variable and an environment variable with the same name
		var capturedURLPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedURLPath = r.URL.Path // We only care about the path part
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// The in-place @host should point to the mock server's URL
		fileContent := fmt.Sprintf(`
@host = %s

### Test Request
GET {{host}}/expected_path
`, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, fileContent)
		tempDir := filepath.Dir(requestFilePath)

		// Create a temporary environment file for this test
		envName := "testPrecedenceEnv"
		envFileName := fmt.Sprintf("http-client.env.%s.json", envName)
		envFilePath := filepath.Join(tempDir, envFileName)
		envData := map[string]string{
			"host": "http://env.example.com/should_not_be_used", // This should be overridden by the in-place variable
		}
		envJSON, err := json.Marshal(envData)
		require.NoError(t, err, "Failed to marshal env data to JSON")
		err = os.WriteFile(envFilePath, envJSON, 0644)
		require.NoError(t, err, "Failed to write temp env file")
		defer os.Remove(envFilePath) // Clean up the temp env file

		client, err := NewClient(WithEnvironment(envName))
		require.NoError(t, err)

		// When: the .http file is executed, it should load the environment from the temp file
		_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		// Then: no error should occur and the in-place variable should take precedence
		require.NoError(t, execErr, "ExecuteFile should not return an error")
		assert.Equal(t, "/expected_path", capturedURLPath, "The request path should match, indicating in-place var was used")
	})

	t.Run("variable_substitution_in_header", func(t *testing.T) {
		// Given: an .http file with an in-place variable used in a header
		var capturedHeaderValue string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeaderValue = r.Header.Get("X-Custom-Header")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fileContent := fmt.Sprintf(`
@my_header_value = secret-token

### Test Request With Header Var
GET %s/somepath
X-Custom-Header: {{my_header_value}}
`, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, fileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When: the .http file is executed
		_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		// Then: no error should occur and the header should be substituted correctly
		require.NoError(t, execErr, "ExecuteFile should not return an error")
		assert.Equal(t, "secret-token", capturedHeaderValue, "The X-Custom-Header should be correctly substituted")
	})

	t.Run("variable_substitution_in_body", func(t *testing.T) {
		// Given: an .http file with an in-place variable used in a JSON request body
		var capturedBody []byte
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			capturedBody, err = io.ReadAll(r.Body)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fileContent := fmt.Sprintf(`
@user_id = user123

### Test Request With Body Var
POST %s/users
Content-Type: application/json

{
  "id": "{{user_id}}",
  "status": "active"
}
`, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, fileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When: the .http file is executed
		_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		// Then: no error should occur and the body should be substituted correctly
		require.NoError(t, execErr, "ExecuteFile should not return an error")
		assert.JSONEq(t, `{"id": "user123", "status": "active"}`, string(capturedBody), "The request body should be correctly substituted")
	})

	t.Run("inplace_variable_defined_by_system_variable", func(t *testing.T) {
		// Given: an .http file with an in-place variable defined by a system variable {{$uuid}}
		var capturedURLPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedURLPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fileContent := fmt.Sprintf(`
@my_request_id = {{$uuid}}

### Test Request With System Var In In-Place Var
GET %s/{{my_request_id}}/resource
`, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, fileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When: the .http file is executed
		_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		// Then: no error should occur and the path should contain a resolved UUID
		require.NoError(t, execErr, "ExecuteFile should not return an error")
		require.NotEmpty(t, capturedURLPath, "Captured URL path should not be empty")

		pathSegments := strings.Split(strings.Trim(capturedURLPath, "/"), "/")
		require.Len(t, pathSegments, 2, "URL path should have two segments")
		assert.Len(t, pathSegments[0], 36, "The first path segment (resolved UUID) should be 36 characters long")
		assert.Equal(t, "resource", pathSegments[1], "The second path segment should be 'resource'")
		assert.NotEqual(t, "{{$uuid}}", pathSegments[0], "The UUID part should not be the literal system variable")
		assert.NotEqual(t, "{{my_request_id}}", pathSegments[0], "The UUID part should not be the literal in-place variable")
	})

	t.Run("inplace_variable_defined_by_os_env_variable", func(t *testing.T) {
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

		// server.URL does not have a trailing slash. {{my_home_dir}} will resolve to testEnvVarValue, which starts with a slash.
		// So, %s{{my_home_dir}} ensures no double slash.
		fileContent := fmt.Sprintf(`
@my_home_dir = {{$processEnv %s}}

### Test Request With OS Env Var In In-Place Var
GET %s{{my_home_dir}}/files
`, testEnvVarName, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, fileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When: the .http file is executed
		results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		// Then: no error should occur and the path should contain the resolved OS env variable
		require.NoError(t, execErr, "ExecuteFile should not return an error for in-place OS env var")
		require.Len(t, results, 1, "Should have one result for in-place OS env var")
		require.Nil(t, results[0].Error, "Request execution error should be nil for in-place OS env var")
		// capturedURLPath should be "/testhome/userdir/files"
		assert.Equal(t, testEnvVarValue+"/files", capturedURLPath, "The URL path should be correctly substituted with the OS environment variable via in-place var")
	})

	t.Run("inplace_variable_in_header", func(t *testing.T) {
		// Given: an .http file with an in-place variable used in a header
		const headerKey = "X-Auth-Token"
		const headerValue = "secret-token-12345"

		var capturedHeaders http.Header
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeaders = r.Header
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fileContent := fmt.Sprintf(`
@my_token = %s

### Test Request With In-Place Var in Header
GET %s/some/path
%s: {{my_token}}
`, headerValue, server.URL, headerKey)

		requestFilePath := createTempHTTPFileFromString(t, fileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When: the .http file is executed
		results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		// Then: no error should occur and the header should be correctly substituted
		require.NoError(t, execErr, "ExecuteFile should not return an error")
		require.Len(t, results, 1, "Should have one result")
		require.Nil(t, results[0].Error, "Request execution error should be nil")
		assert.Equal(t, headerValue, capturedHeaders.Get(headerKey), "The header should be correctly substituted with the in-place variable")
	})

	t.Run("inplace_variable_in_body", func(t *testing.T) {
		// Given: an .http file with an in-place variable used in the request body
		const userIdValue = "user-from-var-456"
		const expectedBody = `{"id": "user-from-var-456", "status": "pending"}`

		var capturedBody []byte
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			capturedBody, err = io.ReadAll(r.Body)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fileContent := fmt.Sprintf(`
@my_user_id = %s

### Test Request With In-Place Var in Body
POST %s/submit
Content-Type: application/json

{
  "id": "{{my_user_id}}",
  "status": "pending"
}
`, userIdValue, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, fileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When: the .http file is executed
		results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		// Then: no error should occur and the body should be correctly substituted
		require.NoError(t, execErr, "ExecuteFile should not return an error")
		require.Len(t, results, 1, "Should have one result")
		require.Nil(t, results[0].Error, "Request execution error should be nil")
		assert.JSONEq(t, expectedBody, string(capturedBody), "The request body should be correctly substituted with the in-place variable")
	})

	t.Run("inplace_variable_defined_by_another_inplace_variable", func(t *testing.T) {
		// Given: an .http file with an in-place variable defined by another in-place variable
		const basePathValue = "/api/v1"
		const resourcePathValue = "items"
		var capturedURLPath string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedURLPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fileContent := fmt.Sprintf(`
@base_path = %s
@resource = %s
@full_url_segment = {{base_path}}/{{resource}}/123

### Test Request With Nested In-Place Var in URL
GET %s{{full_url_segment}}
`, basePathValue, resourcePathValue, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, fileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When: the .http file is executed
		results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		// Then: no error should occur and the URL path should be correctly substituted
		require.NoError(t, execErr, "ExecuteFile should not return an error")
		require.Len(t, results, 1, "Should have one result")
		require.Nil(t, results[0].Error, "Request execution error should be nil")
		assert.Equal(t, "/api/v1/items/123", capturedURLPath, "The URL path should be correctly substituted with nested in-place variables")
	})

	t.Run("inplace_variable_defined_by_uuid_system_variable", func(t *testing.T) {
		// Given: an .http file with an in-place variable defined by the {{$uuid}} system variable
		var capturedHeaderValue string
		const headerKey = "X-Request-ID"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeaderValue = r.Header.Get(headerKey)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fileContent := fmt.Sprintf(`
@my_request_uuid = {{$uuid}}

### Test Request With UUID In-Place Var in Header
GET %s/some/path
%s: {{my_request_uuid}}
`, server.URL, headerKey)

		requestFilePath := createTempHTTPFileFromString(t, fileContent)

		client, err := NewClient()
		require.NoError(t, err)

		// When: the .http file is executed
		results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		// Then: no error should occur and the header should contain a valid UUID
		require.NoError(t, execErr, "ExecuteFile should not return an error")
		require.Len(t, results, 1, "Should have one result")
		require.Nil(t, results[0].Error, "Request execution error should be nil")

		// Validate that the captured header value is a valid UUID
		_, err = uuid.Parse(capturedHeaderValue)
		assert.NoError(t, err, "Header value should be a valid UUID. Got: %s", capturedHeaderValue)
		assert.NotEmpty(t, capturedHeaderValue, "Captured UUID header should not be empty")
	})

	t.Run("inplace_variable_defined_by_dot_env_os_variable", func(t *testing.T) {
		// Given: an .http file with an in-place variable defined by an OS environment variable using {{$env.VAR_NAME}}
		const testEnvVarName = "MY_CONFIG_PATH_TEST_DOT_ENV"
		const testEnvVarValue = "/usr/local/appconfig_dotenv" // Using a value that starts with /
		var capturedURLPath string

		t.Setenv(testEnvVarName, testEnvVarValue)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedURLPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fileContent := fmt.Sprintf(`
@my_path_from_env = {{$env.%s}}

### Test Request With OS Env Var ({{$env.VAR}}) In In-Place Var
GET %s{{my_path_from_env}}/data
`, testEnvVarName, server.URL)

		requestFilePath := createTempHTTPFileFromString(t, fileContent)

		client, err := NewClient()
		require.NoError(t, err)

		results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		require.NoError(t, execErr, "ExecuteFile should not return an error")
		require.Len(t, results, 1, "Should have one result")
		require.Nil(t, results[0].Error, "Request execution error should be nil")

		expectedPath := testEnvVarValue + "/data"
		assert.Equal(t, expectedPath, capturedURLPath, "The URL path should be correctly substituted with the OS environment variable via {{$env.VAR_NAME}} in-place var")
	})

	t.Run("inplace_variable_malformed_definitions", func(t *testing.T) {
		tests := []struct {
			name            string
			httpFileContent string
			expectedError   string // Substring of the expected error message from ExecuteFile
		}{
			{
				name: "name_only_no_equals_no_value",
				httpFileContent: `
@name_only_var

### Test Request
GET http://localhost/test
`,
				expectedError: "malformed in-place variable definition, missing '=' or name",
			},
			{
				name: "no_name_equals_value",
				httpFileContent: `
@=value_only_val

### Test Request
GET http://localhost/test
`,
				expectedError: "variable name cannot be empty in definition",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				requestFilePath := createTempHTTPFileFromString(t, tc.httpFileContent)
				client, err := NewClient()
				require.NoError(t, err)

				_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

				require.Error(t, execErr, "ExecuteFile should return an error for malformed variable definition")
				assert.Contains(t, execErr.Error(), "failed to parse request file", "Error message should indicate parsing failure") // Updated general error check
				assert.Contains(t, execErr.Error(), tc.expectedError, "Error message should contain specific malformed reason")
			})
		}
	})

	t.Run("inplace_variable_defined_by_dotenv_system_variable", func(t *testing.T) {
		// Given: a .env file and an HTTP file using {{$dotenv VAR_NAME}} for an in-place variable
		tempDir := t.TempDir()
		dotEnvContent := "DOTENV_VAR_FOR_SYSTEM_TEST=actual_dotenv_value"
		dotEnvFilePath := filepath.Join(tempDir, ".env")
		err := os.WriteFile(dotEnvFilePath, []byte(dotEnvContent), 0600)
		require.NoError(t, err, "Failed to write .env file")

		var capturedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		serverURL, err := url.Parse(server.URL)
		require.NoError(t, err)

		httpFileContent := fmt.Sprintf(`
@my_api_key = {{$dotenv DOTENV_VAR_FOR_SYSTEM_TEST}}

### Test Request
GET http://%s/{{my_api_key}}
`, serverURL.Host)

		requestFilePath := filepath.Join(tempDir, "test.http")
		err = os.WriteFile(requestFilePath, []byte(httpFileContent), 0600)
		require.NoError(t, err, "Failed to write .http file")

		client, err := NewClient()
		require.NoError(t, err)

		// When: the HTTP file is executed
		responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

		// Then: the request should be successful and the variable substituted correctly
		require.NoError(t, execErr, "ExecuteFile returned an unexpected error")
		require.Len(t, responses, 1, "Expected one response")
		require.Nil(t, responses[0].Error, "Response error should be nil")
		assert.Equal(t, "/actual_dotenv_value", capturedPath, "Expected path to be substituted with .env value via {{$dotenv}}")
	})

	// TODO: Add test for @var = {{$randomInt MIN MAX}}
}

// parseHrespBody reads an .hresp file and parses its content to separate
// headers and body. It returns the parsed headers as http.Header and the body as a string.
// The .hresp format expects headers first, then a blank line, then the body.
func parseHrespBody(filePath string) (http.Header, string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read .hresp file %s: %w", filePath, err)
	}

	parts := strings.SplitN(string(content), "\n\n", 2)
	headers := make(http.Header)
	bodyStr := ""

	headerLines := strings.Split(strings.TrimSpace(parts[0]), "\n")
	for _, line := range headerLines {
		if strings.TrimSpace(line) == "" || !strings.Contains(line, ":") {
			// Skip empty lines or lines not containing a colon (likely the HTTP version line or status)
			continue
		}
		headerParts := strings.SplitN(line, ":", 2)
		if len(headerParts) == 2 {
			headers.Add(strings.TrimSpace(headerParts[0]), strings.TrimSpace(headerParts[1]))
		}
	}

	if len(parts) == 2 {
		bodyStr = strings.TrimSpace(parts[1])
	}

	return headers, bodyStr, nil
}
