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
			output := client.substituteDynamicSystemVariables(tc.input)
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

func TestExecuteFile_WithGuidSystemVariable(t *testing.T) {
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
		interceptedRequest.Header = r.Header.Get("X-Request-ID")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_guid.http", struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for GUID processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// SCENARIO-LIB-014-001: {{$guid}} in URL
	urlParts := strings.Split(interceptedRequest.URL, "/")
	require.True(t, len(urlParts) >= 2, "URL should have at least two parts after splitting by /")
	guidFromURL := urlParts[len(urlParts)-1]
	_, err = uuid.Parse(guidFromURL)
	assert.NoError(t, err, "GUID from URL should be a valid UUID: %s", guidFromURL)

	// SCENARIO-LIB-014-002: {{$guid}} in header
	guidFromHeader := interceptedRequest.Header
	_, err = uuid.Parse(guidFromHeader)
	assert.NoError(t, err, "GUID from X-Request-ID header should be a valid UUID: %s", guidFromHeader)

	// SCENARIO-LIB-014-003: {{$guid}} in body
	// SCENARIO-LIB-014-004: Multiple {{$guid}} in one request yield different GUIDs
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal response body JSON")

	guidFromBody1, ok1 := bodyJSON["transactionId"]
	require.True(t, ok1, "transactionId not found in body")
	_, err = uuid.Parse(guidFromBody1)
	assert.NoError(t, err, "GUID from body (transactionId) should be a valid UUID: %s", guidFromBody1)

	guidFromBody2, ok2 := bodyJSON["correlationId"]
	require.True(t, ok2, "correlationId not found in body")
	_, err = uuid.Parse(guidFromBody2)
	assert.NoError(t, err, "GUID from body (correlationId) should be a valid UUID: %s", guidFromBody2)

	guidFromRandomUuidAlias, ok3 := bodyJSON["randomUuidAlias"]
	require.True(t, ok3, "randomUuidAlias not found in body")
	_, err = uuid.Parse(guidFromRandomUuidAlias)
	assert.NoError(t, err, "GUID from body (randomUuidAlias) should be a valid UUID: %s", guidFromRandomUuidAlias)
	assert.Equal(t, guidFromURL, guidFromRandomUuidAlias, "GUID from URL and randomUuidAlias should be the same")

	// With request-scoped system variables, all {{$guid}} ({{$uuid}}) instances should resolve to the SAME value.
	assert.Equal(t, guidFromURL, guidFromHeader, "GUID from URL and header should be the same")
	assert.Equal(t, guidFromURL, guidFromBody1, "GUID from URL and body1 should be the same")
	// For this test, the .http file uses {{$guid}} twice in the body for different fields.
	// These should now resolve to the same request-scoped GUID.
	assert.Equal(t, guidFromBody1, guidFromBody2, "GUIDs from body (transactionId and correlationId) should be the same")
}

func TestExecuteFile_WithIsoTimestampSystemVariable(t *testing.T) {
	// Given
	var interceptedRequest struct {
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Timestamp-Header")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_iso_timestamp.http", struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for $isoTimestamp processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check header
	_, err = time.Parse(time.RFC3339Nano, interceptedRequest.Header)
	assert.NoError(t, err, "X-Timestamp-Header should be a valid ISO8601 timestamp: %s", interceptedRequest.Header)

	// Check body
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal response body JSON")

	isoTimeFromBody, ok := bodyJSON["requestTime"]
	require.True(t, ok, "requestTime not found in body")
	_, err = time.Parse(time.RFC3339Nano, isoTimeFromBody)
	assert.NoError(t, err, "Body requestTime should be a valid ISO8601 timestamp: %s", isoTimeFromBody)

	assert.Equal(t, interceptedRequest.Header, isoTimeFromBody, "ISO Timestamp from header and body should be the same")
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

func TestExecuteFile_WithDatetimeSystemVariables(t *testing.T) {
	// Given
	var interceptedRequest struct {
		Headers map[string]string
		Body    string
	}
	interceptedRequest.Headers = make(map[string]string)

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		for name, values := range r.Header {
			if len(values) > 0 {
				interceptedRequest.Headers[name] = values[0]
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t,
		"testdata/http_request_files/system_var_datetime.http",
		struct{ ServerURL string }{ServerURL: server.URL},
	)

	// Log the content of the generated temporary file for debugging
	tempFileContent, errRead := os.ReadFile(requestFilePath)
	require.NoError(t, errRead, "Failed to read temporary file for debugging: %s", requestFilePath)
	t.Logf("[DEBUG_TEST] Content of temporary file '%s':\n%s", requestFilePath, string(tempFileContent))

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for datetime processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	now := time.Now()
	threshold := 5 * time.Second // Allow 5s difference for timestamp checks

	// Helper to check datetime strings
	checkDateTimeStr := func(t *testing.T, valueStr string, formatKeyword string, isUTC bool, headerName string) {
		t.Helper()
		if formatKeyword == "timestamp" {
			ts, err := strconv.ParseInt(valueStr, 10, 64)
			require.NoError(t, err, "Failed to parse timestamp from %s: %s", headerName, valueStr)
			parsedTime := time.Unix(ts, 0)
			assert.WithinDuration(t, now, parsedTime, threshold, "%s timestamp %s not within threshold of current time %s", headerName, parsedTime, now)
		} else {
			var layout string
			switch formatKeyword {
			case "rfc1123":
				layout = time.RFC1123
			case "iso8601":
				layout = time.RFC3339 // Go's RFC3339 is ISO8601 compliant
			default:
				t.Fatalf("Unhandled format keyword: %s for %s", formatKeyword, headerName)
			}
			parsedTime, err := time.Parse(layout, valueStr)
			require.NoError(t, err, "Failed to parse datetime string from %s ('%s') with layout '%s'", headerName, valueStr, layout)
			assert.WithinDuration(t, now, parsedTime, threshold, "%s datetime %s not within threshold of current time %s", headerName, parsedTime, now)
			if isUTC {
				assert.Equal(t, time.UTC, parsedTime.Location(), "%s expected to be UTC", headerName)
			} else {
				assert.Equal(t, time.Local, parsedTime.Location(), "%s expected to be Local time", headerName)
			}
		}
	}

	// Check Headers
	checkDateTimeStr(t, interceptedRequest.Headers["X-Datetime-Rfc1123"], "rfc1123", true, "X-Datetime-RFC1123")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Datetime-Iso8601"], "iso8601", true, "X-Datetime-ISO8601")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Datetime-Timestamp"], "timestamp", true, "X-Datetime-Timestamp")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Datetime-Default"], "iso8601", true, "X-Datetime-Default (ISO8601)")

	checkDateTimeStr(t, interceptedRequest.Headers["X-Localdatetime-Rfc1123"], "rfc1123", false, "X-LocalDatetime-RFC1123")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Localdatetime-Iso8601"], "iso8601", false, "X-LocalDatetime-ISO8601")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Localdatetime-Timestamp"], "timestamp", false, "X-LocalDatetime-Timestamp")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Localdatetime-Default"], "iso8601", false, "X-LocalDatetime-Default (ISO8601)")

	assert.Equal(t, "{{$datetime \"invalidFormat\"}}", interceptedRequest.Headers["X-Datetime-Invalid"], "X-Datetime-Invalid should remain unresolved")

	// Check Body
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal request body JSON")

	checkDateTimeStr(t, bodyJSON["utc_rfc1123"], "rfc1123", true, "body.utc_rfc1123")
	checkDateTimeStr(t, bodyJSON["utc_iso8601"], "iso8601", true, "body.utc_iso8601")
	checkDateTimeStr(t, bodyJSON["utc_timestamp"], "timestamp", true, "body.utc_timestamp")
	checkDateTimeStr(t, bodyJSON["utc_default_iso"], "iso8601", true, "body.utc_default_iso (ISO8601)")

	checkDateTimeStr(t, bodyJSON["local_rfc1123"], "rfc1123", false, "body.local_rfc1123")
	checkDateTimeStr(t, bodyJSON["local_iso8601"], "iso8601", false, "body.local_iso8601")
	checkDateTimeStr(t, bodyJSON["local_timestamp"], "timestamp", false, "body.local_timestamp")
	checkDateTimeStr(t, bodyJSON["local_default_iso"], "iso8601", false, "body.local_default_iso (ISO8601)")

	assert.Equal(t, "{{$datetime \"invalidFormat\"}}", bodyJSON["invalid_format_test"], "body.invalid_format_test should remain unresolved")
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

func TestExecuteFile_WithTimestampSystemVariable(t *testing.T) {
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
	beforeTime := time.Now().UTC().Unix()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_timestamp.http", struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for $timestamp processing")
	require.Len(t, responses, 1, "Expected 1 response")
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	afterTime := time.Now().UTC().Unix()

	// SCENARIO-LIB-016-001
	urlParts := strings.Split(interceptedRequest.URL, "/")
	require.True(t, len(urlParts) >= 2, "URL path should have at least two parts")
	timestampFromURLStr := urlParts[len(urlParts)-1]
	timestampFromURL, parseErrURL := strconv.ParseInt(timestampFromURLStr, 10, 64)
	assert.NoError(t, parseErrURL)
	assert.GreaterOrEqual(t, timestampFromURL, beforeTime, "Timestamp from URL should be >= time before request")
	assert.LessOrEqual(t, timestampFromURL, afterTime, "Timestamp from URL should be <= time after request")

	timestampFromHeader, parseErrHeader := strconv.ParseInt(interceptedRequest.Header, 10, 64)
	assert.NoError(t, parseErrHeader)
	assert.GreaterOrEqual(t, timestampFromHeader, beforeTime, "Timestamp from Header should be >= time before request")
	assert.LessOrEqual(t, timestampFromHeader, afterTime, "Timestamp from Header should be <= time after request")

	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err)
	timestampFromBody1Str, ok1 := bodyJSON["event_time"]
	require.True(t, ok1)
	timestampFromBody1, parseErrBody1 := strconv.ParseInt(timestampFromBody1Str, 10, 64)
	assert.NoError(t, parseErrBody1)
	assert.GreaterOrEqual(t, timestampFromBody1, beforeTime)
	assert.LessOrEqual(t, timestampFromBody1, afterTime)

	timestampFromBody2Str, ok2 := bodyJSON["processed_at"]
	require.True(t, ok2)
	timestampFromBody2, parseErrBody2 := strconv.ParseInt(timestampFromBody2Str, 10, 64)
	assert.NoError(t, parseErrBody2)
	assert.GreaterOrEqual(t, timestampFromBody2, beforeTime)
	assert.LessOrEqual(t, timestampFromBody2, afterTime)

	// SCENARIO-LIB-016-002
	assert.Equal(t, timestampFromURL, timestampFromHeader)
	assert.Equal(t, timestampFromHeader, timestampFromBody1)
	assert.Equal(t, timestampFromBody1, timestampFromBody2)
}

func TestExecuteFile_WithRandomIntSystemVariable(t *testing.T) {
	// Given common setup for all subtests
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Random-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()
	client, _ := NewClient()

	tests := []struct {
		name               string
		httpFilePath       string
		validate           func(t *testing.T, url, header, body string)
		expectErrorInParse bool
	}{
		{ // SCENARIO-LIB-015-001
			name:         "valid min max args",
			httpFilePath: "testdata/http_request_files/system_var_randomint_valid_args.http",
			validate: func(t *testing.T, url, header, body string) {
				urlParts := strings.Split(url, "/")
				valURL, err := strconv.Atoi(urlParts[len(urlParts)-2])
				require.NoError(t, err, "Random int from URL should be valid int")
				assert.True(t, valURL >= 10 && valURL <= 20, "URL random int %d out of range [10,20]", valURL)

				valHeader, err := strconv.Atoi(header)
				require.NoError(t, err, "Random int from Header should be valid int")
				assert.True(t, valHeader >= 1 && valHeader <= 5, "Header random int %d out of range [1,5]", valHeader)

				var bodyJSON map[string]int
				err = json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body")
				assert.True(t, bodyJSON["value"] >= 100 && bodyJSON["value"] <= 105, "Body random int %d out of range [100,105]", bodyJSON["value"])
			},
		},
		{ // SCENARIO-LIB-015-002
			name:         "no args",
			httpFilePath: "testdata/http_request_files/system_var_randomint_no_args.http",
			validate: func(t *testing.T, url, header, body string) {
				urlParts := strings.Split(url, "/")
				valURL, err := strconv.Atoi(urlParts[len(urlParts)-2])
				require.NoError(t, err, "Random int from URL (no args) should be valid int")
				assert.True(t, valURL >= 0 && valURL <= 1000, "URL random int (no args) %d out of range [0,1000]", valURL)

				valHeader, err := strconv.Atoi(header)
				require.NoError(t, err, "Random int from Header (no args) should be valid int")
				assert.True(t, valHeader >= 0 && valHeader <= 1000, "Header random int (no args) %d out of range [0,1000]", valHeader)

				var bodyJSON map[string]int
				err = json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body (no args)")
				assert.True(t, bodyJSON["value"] >= 0 && bodyJSON["value"] <= 1000, "Body random int (no args) %d out of range [0,1000]", bodyJSON["value"])
			},
		},
		{ // SCENARIO-LIB-015-003
			name:         "swapped min max args",
			httpFilePath: "testdata/http_request_files/system_var_randomint_swapped_args.http",
			validate: func(t *testing.T, url, header, body string) {
				urlParts := strings.Split(url, "/")
				require.Len(t, urlParts, 4, "URL path should have 4 parts for swapped args test")
				assert.Equal(t, "{{$randomInt 30 25}}", urlParts[2], "URL part1 for swapped_min_max_args should be the unresolved placeholder")
				assert.Equal(t, "{{$randomInt 30 25}}", urlParts[3], "URL part2 for swapped_min_max_args should be the unresolved placeholder")
				var bodyJSON map[string]string
				err := json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body (swapped)")
				assert.Equal(t, "{{$randomInt 30 25}}", bodyJSON["value"], "Body for swapped_min_max_args should be the unresolved placeholder")
			},
		},
		{ // SCENARIO-LIB-015-004
			name:         "malformed args",
			httpFilePath: "testdata/http_request_files/system_var_randomint_malformed_args.http",
			validate: func(t *testing.T, urlStr, header, body string) {
				expectedLiteralPlaceholder := "{{$randomInt abc def}}"
				assert.Contains(t, urlStr, expectedLiteralPlaceholder, "URL should contain literal malformed $randomInt")
				assert.Equal(t, "{{$randomInt 1 xyz}}", header, "Header should retain malformed $randomInt")
				var bodyJSON map[string]string
				err := json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body (malformed)")
				assert.Equal(t, "{{$randomInt foo bar}}", bodyJSON["value"], "Body should retain malformed $randomInt")
			},
			expectErrorInParse: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Given specific setup for this subtest
			requestFilePath := createTestFileFromTemplate(t, tc.httpFilePath, struct{ ServerURL string }{ServerURL: server.URL})

			// When
			responses, err := client.ExecuteFile(context.Background(), requestFilePath)

			// Then
			if tc.expectErrorInParse {
				require.Error(t, err, "Expected an error during ExecuteFile for %s", tc.name)
				return
			}
			require.NoError(t, err, "ExecuteFile should not return an error for %s", tc.name)
			require.Len(t, responses, 1, "Expected 1 response for %s", tc.name)
			resp := responses[0]
			assert.NoError(t, resp.Error, "Response error should be nil for %s", tc.name)
			assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status OK for %s", tc.name)

			actualURL := interceptedRequest.URL
			if strings.Contains(actualURL, "%") {
				decodedURL, decodeErr := url.PathUnescape(actualURL)
				if decodeErr == nil {
					actualURL = decodedURL
				}
			}
			tc.validate(t, actualURL, interceptedRequest.Header, interceptedRequest.Body)
		})
	}
}

func TestExecuteFile_WithDatetimeSystemVariable(t *testing.T) {
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
	beforeTime := time.Now().UTC().Unix()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_timestamp.http", struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for $timestamp processing")
	require.Len(t, responses, 1, "Expected 1 response")
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	afterTime := time.Now().UTC().Unix()

	// SCENARIO-LIB-016-001
	urlParts := strings.Split(interceptedRequest.URL, "/")
	require.True(t, len(urlParts) >= 2, "URL path should have at least two parts")
	timestampFromURLStr := urlParts[len(urlParts)-1]
	timestampFromURL, parseErrURL := strconv.ParseInt(timestampFromURLStr, 10, 64)
	assert.NoError(t, parseErrURL)
	assert.GreaterOrEqual(t, timestampFromURL, beforeTime, "Timestamp from URL should be >= time before request")
	assert.LessOrEqual(t, timestampFromURL, afterTime, "Timestamp from URL should be <= time after request")

	timestampFromHeader, parseErrHeader := strconv.ParseInt(interceptedRequest.Header, 10, 64)
	assert.NoError(t, parseErrHeader)
	assert.GreaterOrEqual(t, timestampFromHeader, beforeTime, "Timestamp from Header should be >= time before request")
	assert.LessOrEqual(t, timestampFromHeader, afterTime, "Timestamp from Header should be <= time after request")

	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err)
	timestampFromBody1Str, ok1 := bodyJSON["event_time"]
	require.True(t, ok1)
	timestampFromBody1, parseErrBody1 := strconv.ParseInt(timestampFromBody1Str, 10, 64)
	assert.NoError(t, parseErrBody1)
	assert.GreaterOrEqual(t, timestampFromBody1, beforeTime)
	assert.LessOrEqual(t, timestampFromBody1, afterTime)

	timestampFromBody2Str, ok2 := bodyJSON["processed_at"]
	require.True(t, ok2)
	timestampFromBody2, parseErrBody2 := strconv.ParseInt(timestampFromBody2Str, 10, 64)
	assert.NoError(t, parseErrBody2)
	assert.GreaterOrEqual(t, timestampFromBody2, beforeTime)
	assert.LessOrEqual(t, timestampFromBody2, afterTime)

	// SCENARIO-LIB-016-002
	assert.Equal(t, timestampFromURL, timestampFromHeader)
	assert.Equal(t, timestampFromHeader, timestampFromBody1)
	assert.Equal(t, timestampFromBody1, timestampFromBody2)
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
		parsedFile, pErr := parseRequestFile(requestFilePath, client)
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
		parsedFile, pErr := parseRequestFile(requestFilePath, client)
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
		parsedFile, pErr := parseRequestFile(requestFilePath, client)
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
			capturedURL = r.URL.String()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		// Use server.URL to make the test robust against dynamic port allocation
		parsedServerURL, psuErr := url.Parse(server.URL)
		require.NoError(t, psuErr)
		hostFromServer := parsedServerURL.Host // e.g., 127.0.0.1:PORT

		httpFileContent := fmt.Sprintf(`
@my_host = %s
@base_api_path = /api/v2
@full_api_url = http://{{my_host}}{{base_api_path}}
@items_endpoint = {{full_api_url}}/items

GET {{items_endpoint}}?host_check={{my_host}}
`, hostFromServer)

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

		// Verify the server received the request at the correctly resolved URL
		expectedPathAndQuery := fmt.Sprintf("/api/v2/items?host_check=%s", hostFromServer)
		assert.Equal(t, expectedPathAndQuery, capturedURL)

		// Verify ParsedFile.FileVariables (should store raw definitions)
		parsedFile, pErr := parseRequestFile(requestFilePath, client)
		require.NoError(t, pErr)
		require.NotNil(t, parsedFile.FileVariables)
		assert.Equal(t, hostFromServer, parsedFile.FileVariables["my_host"])
		assert.Equal(t, "/api/v2", parsedFile.FileVariables["base_api_path"])
		assert.Equal(t, "http://{{my_host}}{{base_api_path}}", parsedFile.FileVariables["full_api_url"])
		assert.Equal(t, "{{full_api_url}}/items", parsedFile.FileVariables["items_endpoint"])
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

	// TODO: Add more sub-tests for other scenarios:
	// - variable in header
	// - variable in body
	// - variable defined with another variable (e.g., @var2 = {{var1}})
	// - variable defined with system variable (e.g., @var_uuid = {{$uuid}})
	// - variable defined with OS env variable (e.g., @var_env = {{$env.MY_TEST_VAR}})
	// - malformed definitions (e.g., @name_no_value, @=value)
}
