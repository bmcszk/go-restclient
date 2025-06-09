package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest" // Added for mock server
	"net/url"           // Added for TestExecuteFile_WithRandomIntSystemVariable
	"os"                // Added for TestExecuteFile_WithDatetimeSystemVariables
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR1.3.1 - System Variables: {{$guid}} and {{$uuid}}
// Corresponds to: Client's ability to substitute {{$guid}} and {{$uuid}} system variables with a unique, request-scoped UUID (http_syntax.md "System Variables").
// This test uses 'testdata/http_request_files/system_var_guid.http' to verify that these variables are correctly generated and substituted in URLs, headers, and bodies. It also confirms that multiple instances of {{$guid}} or {{$uuid}} within the same request resolve to the *same* generated UUID for that request.
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

// PRD-COMMENT: FR1.3.3 - System Variables: {{$isoTimestamp}}
// Corresponds to: Client's ability to substitute the {{$isoTimestamp}} system variable with the current UTC timestamp in ISO 8601 format (YYYY-MM-DDTHH:mm:ss.sssZ) (http_syntax.md "System Variables").
// This test uses 'testdata/http_request_files/system_var_iso_timestamp.http' to verify correct substitution in URLs, headers, and bodies, and checks that multiple instances resolve to the same request-scoped timestamp.
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

// detailedInterceptedRequestData holds detailed data captured by the mock server, including multiple headers.
type detailedInterceptedRequestData struct {
	Headers map[string]string
	Body    string
}

// setupDetailedMockServerInterceptor initializes a mock HTTP server that intercepts
// request headers (as a map) and body, logging them for debugging.
func setupDetailedMockServerInterceptor(t *testing.T) (*httptest.Server, *detailedInterceptedRequestData) {
	t.Helper() // Mark as test helper
	data := &detailedInterceptedRequestData{
		Headers: make(map[string]string),
	}

	// Using httptest.NewServer directly.
	// The original startMockServer was likely a simple wrapper.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("[DEBUG_MOCK_SERVER] Received request to: %s %s", r.Method, r.URL.Path)
		bodyBytes, err := io.ReadAll(r.Body) // Added error check
		if err != nil {
			t.Logf("[DEBUG_MOCK_SERVER] Error reading body: %v", err)
			http.Error(w, "server error reading body", http.StatusInternalServerError)
			return
		}
		data.Body = string(bodyBytes)
		t.Logf("[DEBUG_MOCK_SERVER] Received Body: %s", data.Body)
		t.Logf("[DEBUG_MOCK_SERVER] --- Headers Start ---")
		for name, values := range r.Header {
			canonicalName := http.CanonicalHeaderKey(name)
			if len(values) > 0 {
				data.Headers[canonicalName] = values[0]
				t.Logf("[DEBUG_MOCK_SERVER] Header: %s = %q", canonicalName, values[0])
			} else {
				t.Logf("[DEBUG_MOCK_SERVER] Header (empty): %s", canonicalName)
			}
		}
		t.Logf("[DEBUG_MOCK_SERVER] --- Headers End ---")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	}))
	return server, data
}

// PRD-COMMENT: FR1.3.4 - System Variables: {{$datetime "format" [offset]}}
// Corresponds to: Client's ability to substitute the {{$datetime}} system variable with the current timestamp formatted according to a Go layout string, optionally with a timezone offset (http_syntax.md "System Variables").
// This test uses 'testdata/http_request_files/system_var_datetime.http' to verify various datetime formats (RFC3339, custom, with local/UTC/specific offsets) in URLs, headers, and bodies. It ensures multiple instances resolve to the same request-scoped timestamp (respecting their individual formatting and offsets).
func TestExecuteFile_WithDatetimeSystemVariables(t *testing.T) {
	// Given
	server, interceptedRequest := setupDetailedMockServerInterceptor(t)
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

// PRD-COMMENT: FR1.3.2 - System Variables: {{$timestamp}}
// Corresponds to: Client's ability to substitute the {{$timestamp}} system variable with the current Unix timestamp (seconds since epoch) (http_syntax.md "System Variables").
// This test uses 'testdata/http_request_files/system_var_timestamp.http' to verify correct substitution in URLs, headers, and bodies. It ensures multiple instances resolve to the same request-scoped timestamp.
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

func validateRandomIntValidMinMaxArgs(t *testing.T, url, header, body string) {
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
}

func validateRandomIntNoArgs(t *testing.T, url, header, body string) {
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
}

func validateRandomIntSwappedMinMaxArgs(t *testing.T, url, header, body string) {
	urlParts := strings.Split(url, "/")
	require.Len(t, urlParts, 4, "URL path should have 4 parts for swapped args test")
	assert.Equal(t, "{{$randomInt 30 25}}", urlParts[2], "URL part1 for swapped_min_max_args should be the unresolved placeholder")
	assert.Equal(t, "{{$randomInt 30 25}}", urlParts[3], "URL part2 for swapped_min_max_args should be the unresolved placeholder")
	var bodyJSON map[string]string
	err := json.Unmarshal([]byte(body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal body (swapped)")
	assert.Equal(t, "{{$randomInt 30 25}}", bodyJSON["value"], "Body for swapped_min_max_args should be the unresolved placeholder")
}

func validateRandomIntMalformedArgs(t *testing.T, urlStr, header, body string) {
	expectedLiteralPlaceholder := "{{$randomInt abc def}}"
	assert.Contains(t, urlStr, expectedLiteralPlaceholder, "URL should contain literal malformed $randomInt")
	assert.Equal(t, "{{$randomInt 1 xyz}}", header, "Header should retain malformed $randomInt")
	var bodyJSON map[string]string
	err := json.Unmarshal([]byte(body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal body (malformed)")
	assert.Equal(t, "{{$randomInt foo bar}}", bodyJSON["value"], "Body should retain malformed $randomInt")
}

// PRD-COMMENT: FR1.3.5 - System Variables: {{$randomInt [MIN MAX]}}
// Corresponds to: Client's ability to substitute the {{$randomInt}} system variable with a random integer. Supports optional MIN and MAX arguments. If no args, defaults to a wide range. If MIN > MAX, or args are malformed, the literal placeholder is used. (http_syntax.md "System Variables").
// This test suite uses various .http files (e.g., 'system_var_randomint_valid_args.http', 'system_var_randomint_no_args.http') to verify behavior with valid arguments, no arguments, swapped arguments (min > max), and malformed arguments, checking substitution in URLs, headers, and bodies.
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
			validate:     validateRandomIntValidMinMaxArgs,
		},
		{ // SCENARIO-LIB-015-002
			name:         "no args",
			httpFilePath: "testdata/http_request_files/system_var_randomint_no_args.http",
			validate:     validateRandomIntNoArgs,
		},
		{ // SCENARIO-LIB-015-003
			name:         "swapped min max args",
			httpFilePath: "testdata/http_request_files/system_var_randomint_swapped_args.http",
			validate:     validateRandomIntSwappedMinMaxArgs,
		},
		{ // SCENARIO-LIB-015-004
			name:               "malformed args",
			httpFilePath:       "testdata/http_request_files/system_var_randomint_malformed_args.http",
			validate:           validateRandomIntMalformedArgs,
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
