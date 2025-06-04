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
				// Get offset for time.Local
				_, localOffset := now.In(time.Local).Zone()
				// Get offset for parsedTime
				_, parsedOffset := parsedTime.Zone()
				assert.Equal(t, localOffset, parsedOffset, "%s expected to have local time offset, got %d, want %d", headerName, parsedOffset, localOffset)
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

// TestExecuteFile_InPlaceVars_SimpleURL extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_SimpleURL(t *testing.T) {
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
	file, err := os.Open(requestFilePath)
	require.NoError(t, err)
	defer file.Close()
	// Note: The environment and programmaticVars maps are empty here as per original sub-test.
	// If these new top-level tests need specific environments, this part might need adjustment later.
	parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, server.URL, parsedFile.FileVariables["hostname"])
	assert.Equal(t, "/api/v1/items", parsedFile.FileVariables["path_segment"])
}

// TestExecuteFile_InPlaceVars_HeaderSubstitution extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_HeaderSubstitution(t *testing.T) {
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
	file, err := os.Open(requestFilePath)
	require.NoError(t, err)
	defer file.Close()
	parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "Bearer_secret_token_123", parsedFile.FileVariables["auth_token"])
}

// TestExecuteFile_InPlaceVars_BodySubstitution extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_BodySubstitution(t *testing.T) {
	// Given
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errReadBody error
		capturedBody, errReadBody = io.ReadAll(r.Body)
		require.NoError(t, errReadBody)
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
	file, err := os.Open(requestFilePath)
	require.NoError(t, err)
	defer file.Close()
	parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "SuperWidget", parsedFile.FileVariables["product_name"])
	assert.Equal(t, "SW1000", parsedFile.FileVariables["product_id"])
	assert.Equal(t, "49.99", parsedFile.FileVariables["product_price"])
}

// TestExecuteFile_InPlaceVars_VarDefinedByVar extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_VarDefinedByVar(t *testing.T) {
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
	file, err := os.Open(requestFilePath)
	require.NoError(t, err)
	defer file.Close()
	parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, hostFromServer, parsedFile.FileVariables["my_host"])
	assert.Equal(t, "/api/v2", parsedFile.FileVariables["base_api_path"])
	assert.Equal(t, "http://{{my_host}}{{base_api_path}}", parsedFile.FileVariables["full_api_url"])
	assert.Equal(t, "{{full_api_url}}/items", parsedFile.FileVariables["items_endpoint"])
}

// TestExecuteFile_InPlaceVars_PrecedenceOverEnv extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_PrecedenceOverEnv(t *testing.T) {
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
}

// TestExecuteFile_InPlaceVars_SubstInHeader extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_SubstInHeader(t *testing.T) {
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
}

// TestExecuteFile_InPlaceVars_SubstInBody extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_SubstInBody(t *testing.T) {
	// Given: an .http file with an in-place variable used in a JSON request body
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errReadBody error
		capturedBody, errReadBody = io.ReadAll(r.Body)
		require.NoError(t, errReadBody) // Use errReadBody
		defer r.Body.Close()
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
}

// TestExecuteFile_InPlaceVars_SystemVar extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_SystemVar(t *testing.T) {
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
}

// TestExecuteFile_InPlaceVars_OsEnvVar extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_OsEnvVar(t *testing.T) {
	// Given: an OS environment variable and an .http file with an in-place variable defined by it
	const testEnvVarName = "TEST_USER_HOME_INPLACE_OS" // Changed name slightly to avoid potential collision
	const testEnvVarValue = "/testhome/userosdir"      // Changed value slightly
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
	// capturedURLPath should be "/testhome/userosdir/files"
	assert.Equal(t, testEnvVarValue+"/files", capturedURLPath, "The URL path should be correctly substituted with the OS environment variable via in-place var")
}

// TestExecuteFile_InPlaceVars_InHeader extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_InHeader(t *testing.T) {
	// Given: an .http file with an in-place variable used in a header
	const headerKey = "X-Auth-Token-InPlace"         // Slightly changed key for independence
	const headerValue = "secret-token-inplace-54321" // Slightly changed value

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
}

// TestExecuteFile_InPlaceVars_InBody extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_InBody(t *testing.T) {
	// Given: an .http file with an in-place variable used in the request body
	const userIdValue = "user-from-var-inplace-body-789"                                 // Slightly changed for independence
	const expectedBody = `{"id": "user-from-var-inplace-body-789", "status": "pending"}` // Adjusted expected body

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
}

// TestExecuteFile_InPlaceVars_DefinedByAnotherInPlaceVar tests the scenario where an in-place variable
// is defined by another in-place variable, and this composite variable is used in the request URL.
func TestExecuteFile_InPlaceVars_DefinedByAnotherInPlaceVar(t *testing.T) {
	// Given: an .http file with an in-place variable defined by another in-place variable
	const basePathValue = "/api/v1/nested"   // Changed to avoid potential conflicts if run in parallel
	const resourcePathValue = "items_nested" // Changed to avoid potential conflicts
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
	assert.Equal(t, "/api/v1/nested/items_nested/123", capturedURLPath, "The URL path should be correctly substituted with nested in-place variables")
}

// TestExecuteFile_InPlaceVars_DefinedByUUIDSystemVar tests the scenario where an in-place variable
// is defined by the {{$uuid}} system variable and used in a request header.
func TestExecuteFile_InPlaceVars_DefinedByUUIDSystemVar(t *testing.T) {
	// Given: an .http file with an in-place variable defined by the {{$uuid}} system variable
	var capturedHeaderValue string
	const headerKey = "X-Request-ID-UUID-Test" // Changed to avoid potential conflicts

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaderValue = r.Header.Get(headerKey)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileContent := fmt.Sprintf(`
@my_request_uuid = {{$uuid}}

### Test Request With UUID In-Place Var in Header
GET %s/some/path/uuidtest
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
}

// TestExecuteFile_InPlaceVars_DefinedByDotEnvOsVar tests in-place variable substitution
// where the variable is defined by an OS environment variable using {{$env.VAR_NAME}} syntax.
func TestExecuteFile_InPlaceVars_DefinedByDotEnvOsVar(t *testing.T) {
	// Given: an .http file with an in-place variable defined by an OS environment variable using {{$env.VAR_NAME}}
	const testEnvVarName = "MY_CONFIG_PATH_DOT_ENV_EXTRACTED"       // Modified for isolation
	const testEnvVarValue = "/usr/local/appconfig_dotenv_extracted" // Modified for isolation
	var capturedURLPath string

	t.Setenv(testEnvVarName, testEnvVarValue)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileContent := fmt.Sprintf(`
@my_path_from_env_ext = {{$env.%s}}

### Test Request With OS Env Var ({{$env.VAR}}) In In-Place Var
GET %s{{my_path_from_env_ext}}/data_ext
`, testEnvVarName, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the URL path should be correctly substituted
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, results, 1, "Should have one result")
	require.Nil(t, results[0].Error, "Request execution error should be nil")

	expectedPath := testEnvVarValue + "/data_ext"
	assert.Equal(t, expectedPath, capturedURLPath, "The URL path should be correctly substituted with the OS environment variable via {{$env.VAR_NAME}} in-place var")
}

func TestExecuteFile_InPlaceVariables(t *testing.T) {
	/*
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
				file, err := os.Open(requestFilePath)
				require.NoError(t, err)
				defer file.Close()
				parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
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
				file, err := os.Open(requestFilePath)
				require.NoError(t, err)
				defer file.Close()
				parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
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
				file, err := os.Open(requestFilePath)
				require.NoError(t, err)
				defer file.Close()
				parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
				require.NoError(t, pErr)
				require.NotNil(t, parsedFile.FileVariables)
				assert.Equal(t, "SuperWidget", parsedFile.FileVariables["product_name"])
				assert.Equal(t, "SW1000", parsedFile.FileVariables["product_id"])
				assert.Equal(t, "49.99", parsedFile.FileVariables["product_price"])
			})
	*/

	/*
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
				file, err := os.Open(requestFilePath)
				require.NoError(t, err)
				defer file.Close()
				parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
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
	*/

	/*
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
	*/

	/*
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
	*/

	/*
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
	*/

	/*
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
	*/

	/*
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
	*/

	/*
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
	*/

	/*
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
	*/

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
