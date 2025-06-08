package restclient

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
	//t.Skip("Skipping due to bugs in {{$env VAR}} substitution (MEMORY 1e157e39-d7fc-4b0e-b273-5f22eb1f27c6, MEMORY d7bc6730-ff10-42fb-8959-482f75102f4b): issues with empty values and var names starting with underscores. See tasks TBD for fixes.")
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
			output := substituteDynamicSystemVariables(tc.input, client.currentDotEnvVars, client.programmaticVars)
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
	assert.Equal(t, "response for items ()", resp3.BodyString)
}

func TestExecuteFile_WithProcessEnvSystemVariable(t *testing.T) {
	//t.Skip("Skipping due to bug in {{$processEnv VAR}} substitution (MEMORY d1edb831-da89-4cde-93ad-a9129eb7b8aa): placeholder not replaced with OS environment variable value. See task TBD for fix.")
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
	//t.Skip("Skipping due to bug in {{$dotenv VAR}} substitution (MEMORY ???): placeholder not replaced with empty string when .env file/OS env var is missing. See task TBD for fix.")
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
	selectedEnv              string // Environment to select in NewClient
	expectExecuteFileError   bool
	executeFileErrorContains string
	expectResponseError      bool
	responseErrorContains    string
	responseAssertions       func(t *testing.T, resp *Response, interceptedReq *interceptedRequestData, serverURL string)
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

	var client *Client
	if tc.selectedEnv != "" {
		client, err = NewClient(WithEnvironment(tc.selectedEnv))
	} else {
		client, err = NewClient()
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

// TestExecuteFile_WithHttpClientEnvJson tests variable substitution from http-client.env.json (Task T4)
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
			responseAssertions: func(t *testing.T, resp *Response, interceptedReq *interceptedRequestData, serverURL string) {
				assert.True(t, strings.Contains(resp.Request.RawURLString, "{{host}}"), "RawURLString should still contain {{host}}")
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
			responseAssertions: func(t *testing.T, resp *Response, interceptedReq *interceptedRequestData, serverURL string) {
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
