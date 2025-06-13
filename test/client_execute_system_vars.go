package test

import (
	rc "github.com/bmcszk/go-restclient"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest" // Added for mock server
	"net/url"           // Added for TestExecuteFile_WithRandomIntSystemVariable
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR1.3.1 - System Variables: {{$guid}} and {{$uuid}}
// Corresponds to: Client's ability to substitute {{$guid}} and {{$uuid}} system variables
// with a unique, request-scoped UUID (http_syntax.md "System Variables").
// This test uses 'test/data/http_request_files/system_var_guid.http' to verify that these variables
// are correctly generated and substituted in URLs, headers, and bodies. It also confirms that
// multiple instances of {{$guid}} or {{$uuid}} within the same request resolve to the *same*
// generated UUID for that request.
func RunExecuteFile_WithGuidSystemVariable(t *testing.T) {
	t.Helper()
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

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/http_request_files/system_var_guid.http",
		struct{ ServerURL string }{ServerURL: server.URL})

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
	assert.Equal(t, guidFromBody1, guidFromBody2,
		"GUIDs from body (transactionId and correlationId) should be the same")
}

// PRD-COMMENT: FR1.3.3 - System Variables: {{$isoTimestamp}}
// Corresponds to: Client's ability to substitute the {{$isoTimestamp}} system variable
// with the current UTC timestamp in ISO 8601 format (YYYY-MM-DDTHH:mm:ss.sssZ)
// (http_syntax.md "System Variables").
// This test uses 'test/data/http_request_files/system_var_iso_timestamp.http' to verify correct
// substitution in URLs, headers, and bodies, and checks that multiple instances resolve to
// the same request-scoped timestamp.
func RunExecuteFile_WithIsoTimestampSystemVariable(t *testing.T) {
	t.Helper()
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

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/http_request_files/system_var_iso_timestamp.http",
		struct{ ServerURL string }{ServerURL: server.URL})

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
// Corresponds to: Client's ability to substitute the {{$datetime}} system variable with the
// current timestamp formatted according to a Go layout string, optionally with a timezone
// offset (http_syntax.md "System Variables").
// This test uses 'test/data/http_request_files/system_var_datetime.http' to verify various
// datetime formats (RFC3339, custom, with local/UTC/specific offsets) in URLs, headers, and bodies.
// It ensures multiple instances resolve to the same request-scoped timestamp (respecting their
// individual formatting and offsets).
func RunExecuteFile_WithDatetimeSystemVariables(t *testing.T) {
	t.Helper()
	// Given
	server, interceptedRequest := setupDetailedMockServerInterceptor(t)
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t,
		"test/data/http_request_files/system_var_datetime.http",
		struct{ ServerURL string }{ServerURL: server.URL},
	)

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for datetime processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Validate datetime values in headers and body
	validateDatetimeHeaders(t, interceptedRequest)
	validateDatetimeBody(t, interceptedRequest)
}

// checkDateTimeStrUTC validates datetime strings expecting UTC timezone
func checkDateTimeStrUTC(t *testing.T, valueStr string, formatKeyword string,
	headerName string, now time.Time, threshold time.Duration) {
	t.Helper()
	checkDateTimeStrWithTimezone(t, valueStr, formatKeyword, headerName, now, threshold, time.UTC)
}

// checkDateTimeStrLocal validates datetime strings expecting local timezone
func checkDateTimeStrLocal(t *testing.T, valueStr string, formatKeyword string,
	headerName string, now time.Time, threshold time.Duration) {
	t.Helper()
	checkDateTimeStrWithTimezone(t, valueStr, formatKeyword, headerName, now, threshold, time.Local)
}

// checkDateTimeStrWithTimezone is the core validation function
func checkDateTimeStrWithTimezone(t *testing.T, valueStr string, formatKeyword string,
	headerName string, now time.Time, threshold time.Duration, expectedLocation *time.Location) {
	t.Helper()
	if formatKeyword == "timestamp" {
		ts, err := strconv.ParseInt(valueStr, 10, 64)
		require.NoError(t, err, "Failed to parse timestamp from %s: %s", headerName, valueStr)
		parsedTime := time.Unix(ts, 0)
		assert.WithinDuration(t, now, parsedTime, threshold,
			"%s timestamp %s not within threshold of current time %s", headerName, parsedTime, now)
		return
	}

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
	require.NoError(t, err, "Failed to parse datetime string from %s ('%s') with layout '%s'",
		headerName, valueStr, layout)
	assert.WithinDuration(t, now, parsedTime, threshold,
		"%s datetime %s not within threshold of current time %s", headerName, parsedTime, now)
	
	if expectedLocation == time.UTC {
		assert.Equal(t, time.UTC, parsedTime.Location(), "%s expected to be UTC", headerName)
	} else {
		// Get offset for time.Local
		_, localOffset := now.In(time.Local).Zone()
		// Get offset for parsedTime
		_, parsedOffset := parsedTime.Zone()
		assert.Equal(t, localOffset, parsedOffset,
			"%s expected to have local time offset, got %d, want %d", headerName, parsedOffset, localOffset)
	}
}

func validateDatetimeHeaders(t *testing.T, interceptedRequest *detailedInterceptedRequestData) {
	t.Helper()
	now := time.Now()
	threshold := 5 * time.Second

	// Check Headers
	checkDateTimeStrUTC(t, interceptedRequest.Headers["X-Datetime-Rfc1123"], "rfc1123",
		"X-Datetime-RFC1123", now, threshold)
	checkDateTimeStrUTC(t, interceptedRequest.Headers["X-Datetime-Iso8601"], "iso8601",
		"X-Datetime-ISO8601", now, threshold)
	checkDateTimeStrUTC(t, interceptedRequest.Headers["X-Datetime-Timestamp"], "timestamp",
		"X-Datetime-Timestamp", now, threshold)
	checkDateTimeStrUTC(t, interceptedRequest.Headers["X-Datetime-Default"], "iso8601",
		"X-Datetime-Default (ISO8601)", now, threshold)

	checkDateTimeStrLocal(t, interceptedRequest.Headers["X-Localdatetime-Rfc1123"], "rfc1123",
		"X-LocalDatetime-RFC1123", now, threshold)
	checkDateTimeStrLocal(t, interceptedRequest.Headers["X-Localdatetime-Iso8601"], "iso8601",
		"X-LocalDatetime-ISO8601", now, threshold)
	checkDateTimeStrLocal(t, interceptedRequest.Headers["X-Localdatetime-Timestamp"],
		"timestamp", "X-LocalDatetime-Timestamp", now, threshold)
	checkDateTimeStrLocal(t, interceptedRequest.Headers["X-Localdatetime-Default"],
		"iso8601", "X-LocalDatetime-Default (ISO8601)", now, threshold)

	assert.Equal(t, "{{$datetime \"invalidFormat\"}}", interceptedRequest.Headers["X-Datetime-Invalid"],
		"X-Datetime-Invalid should remain unresolved")
}

func validateDatetimeBody(t *testing.T, interceptedRequest *detailedInterceptedRequestData) {
	t.Helper()
	now := time.Now()
	threshold := 5 * time.Second

	// Check Body
	var bodyJSON map[string]string
	err := json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal request body JSON")

	checkDateTimeStrUTC(t, bodyJSON["utc_rfc1123"], "rfc1123", "body.utc_rfc1123", now, threshold)
	checkDateTimeStrUTC(t, bodyJSON["utc_iso8601"], "iso8601", "body.utc_iso8601", now, threshold)
	checkDateTimeStrUTC(t, bodyJSON["utc_timestamp"], "timestamp", "body.utc_timestamp", now, threshold)
	checkDateTimeStrUTC(t, bodyJSON["utc_default_iso"], "iso8601", "body.utc_default_iso (ISO8601)", now, threshold)

	checkDateTimeStrLocal(t, bodyJSON["local_rfc1123"], "rfc1123", "body.local_rfc1123", now, threshold)
	checkDateTimeStrLocal(t, bodyJSON["local_iso8601"], "iso8601", "body.local_iso8601", now, threshold)
	checkDateTimeStrLocal(t, bodyJSON["local_timestamp"], "timestamp", "body.local_timestamp", now, threshold)
	checkDateTimeStrLocal(t, bodyJSON["local_default_iso"], "iso8601",
		"body.local_default_iso (ISO8601)", now, threshold)

	assert.Equal(t, "{{$datetime \"invalidFormat\"}}", bodyJSON["invalid_format_test"],
		"body.invalid_format_test should remain unresolved")
}

// PRD-COMMENT: FR1.3.2 - System Variables: {{$timestamp}}
// Corresponds to: Client's ability to substitute the {{$timestamp}} system variable with
// the current Unix timestamp (seconds since epoch) (http_syntax.md "System Variables").
// This test uses 'test/data/http_request_files/system_var_timestamp.http' to verify correct
// substitution in URLs, headers, and bodies. It ensures multiple instances resolve to
// the same request-scoped timestamp.
func RunExecuteFile_WithTimestampSystemVariable(t *testing.T) {
	t.Helper()
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
	beforeTime := time.Now().UTC().Unix()
	requestFilePath := createTestFileFromTemplate(t, "test/data/http_request_files/system_var_timestamp.http",
		struct{ ServerURL string }{ServerURL: server.URL})

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

func validateRandomIntValidMinMaxArgs(t *testing.T, reqURL, header, body string) {
	t.Helper()
	urlParts := strings.Split(reqURL, "/")
	valURL, err := strconv.Atoi(urlParts[len(urlParts)-2])
	require.NoError(t, err, "Random int from URL should be valid int")
	assert.True(t, valURL >= 10 && valURL <= 20, "URL random int %d out of range [10,20]", valURL)

	valHeader, err := strconv.Atoi(header)
	require.NoError(t, err, "Random int from Header should be valid int")
	assert.True(t, valHeader >= 1 && valHeader <= 5, "Header random int %d out of range [1,5]", valHeader)

	var bodyJSON map[string]int
	err = json.Unmarshal([]byte(body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal body")
	assert.True(t, bodyJSON["value"] >= 100 && bodyJSON["value"] <= 105,
		"Body random int %d out of range [100,105]", bodyJSON["value"])
}

func validateRandomIntNoArgs(t *testing.T, reqURL, header, body string) {
	t.Helper()
	urlParts := strings.Split(reqURL, "/")
	valURL, err := strconv.Atoi(urlParts[len(urlParts)-2])
	require.NoError(t, err, "Random int from URL (no args) should be valid int")
	assert.True(t, valURL >= 0 && valURL <= 1000, "URL random int (no args) %d out of range [0,1000]", valURL)

	valHeader, err := strconv.Atoi(header)
	require.NoError(t, err, "Random int from Header (no args) should be valid int")
	assert.True(t, valHeader >= 0 && valHeader <= 1000,
		"Header random int (no args) %d out of range [0,1000]", valHeader)

	var bodyJSON map[string]int
	err = json.Unmarshal([]byte(body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal body (no args)")
	assert.True(t, bodyJSON["value"] >= 0 && bodyJSON["value"] <= 1000,
		"Body random int (no args) %d out of range [0,1000]", bodyJSON["value"])
}

func validateRandomIntSwappedMinMaxArgs(t *testing.T, reqURL, _ /* header */, body string) {
	t.Helper()
	urlParts := strings.Split(reqURL, "/")
	require.Len(t, urlParts, 4, "URL path should have 4 parts for swapped args test")
	assert.Equal(t, "{{$randomInt 30 25}}", urlParts[2],
		"URL part1 for swapped_min_max_args should be the unresolved placeholder")
	assert.Equal(t, "{{$randomInt 30 25}}", urlParts[3],
		"URL part2 for swapped_min_max_args should be the unresolved placeholder")
	var bodyJSON map[string]string
	err := json.Unmarshal([]byte(body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal body (swapped)")
	assert.Equal(t, "{{$randomInt 30 25}}", bodyJSON["value"],
		"Body for swapped_min_max_args should be the unresolved placeholder")
}

func validateRandomIntMalformedArgs(t *testing.T, urlStr, header, body string) {
	t.Helper()
	expectedLiteralPlaceholder := "{{$randomInt abc def}}"
	assert.Contains(t, urlStr, expectedLiteralPlaceholder, "URL should contain literal malformed $randomInt")
	assert.Equal(t, "{{$randomInt 1 xyz}}", header, "Header should retain malformed $randomInt")
	var bodyJSON map[string]string
	err := json.Unmarshal([]byte(body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal body (malformed)")
	assert.Equal(t, "{{$randomInt foo bar}}", bodyJSON["value"], "Body should retain malformed $randomInt")
}

// PRD-COMMENT: FR1.3.5 - System Variables: {{$randomInt [MIN MAX]}}
// Corresponds to: Client's ability to substitute the {{$randomInt}} system variable with
// a random integer. Supports optional MIN and MAX arguments. If no args, defaults to a wide range.
// If MIN > MAX, or args are malformed, the literal placeholder is used.
// (http_syntax.md "System Variables").
// This test suite uses various .http files (e.g., 'system_var_randomint_valid_args.http',
// 'system_var_randomint_no_args.http') to verify behavior with valid arguments, no arguments,
// swapped arguments (min > max), and malformed arguments, checking substitution in URLs, headers, and bodies.
func RunExecuteFile_WithRandomIntSystemVariable(t *testing.T) {
	t.Helper()
	interceptedRequest, server, client := setupRandomIntTest()
	defer server.Close()

	tests := getRandomIntTestCases()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runRandomIntTestCase(t, tc, client, server.URL, interceptedRequest)
		})
	}
}

// randomIntTestCase represents a test case for random int system variables
type randomIntTestCase struct {
	name               string
	httpFilePath       string
	validate           func(t *testing.T, url, header, body string)
	expectErrorInParse bool
}

// randomIntRequestData holds request data intercepted by mock server for random int tests
type randomIntRequestData struct {
	URL    string
	Header string
	Body   string
}

// setupRandomIntTest sets up the test environment for random int tests
func setupRandomIntTest() (*randomIntRequestData, *httptest.Server, *rc.Client) {
	var interceptedRequest randomIntRequestData
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Random-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	client, _ := rc.NewClient()
	return &interceptedRequest, server, client
}

// getRandomIntTestCases returns test cases for random int system variables
func getRandomIntTestCases() []randomIntTestCase {
	return []randomIntTestCase{
		{ // SCENARIO-LIB-015-001
			name:         "valid min max args",
			httpFilePath: "test/data/http_request_files/system_var_randomint_valid_args.http",
			validate:     validateRandomIntValidMinMaxArgs,
		},
		{ // SCENARIO-LIB-015-002
			name:         "no args",
			httpFilePath: "test/data/http_request_files/system_var_randomint_no_args.http",
			validate:     validateRandomIntNoArgs,
		},
		{ // SCENARIO-LIB-015-003
			name:         "swapped min max args",
			httpFilePath: "test/data/http_request_files/system_var_randomint_swapped_args.http",
			validate:     validateRandomIntSwappedMinMaxArgs,
		},
		{ // SCENARIO-LIB-015-004
			name:               "malformed args",
			httpFilePath:       "test/data/http_request_files/system_var_randomint_malformed_args.http",
			validate:           validateRandomIntMalformedArgs,
			expectErrorInParse: false,
		},
	}
}

// runRandomIntTestCase executes a single random int test case
func runRandomIntTestCase(t *testing.T, tc randomIntTestCase, client *rc.Client, serverURL string,
	interceptedRequest *randomIntRequestData) {
	t.Helper()
	requestFilePath := createTestFileFromTemplate(t, tc.httpFilePath, struct{ ServerURL string }{ServerURL: serverURL})

	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	if tc.expectErrorInParse {
		require.Error(t, err, "Expected an error during ExecuteFile for %s", tc.name)
		return
	}

	validateRandomIntResponse(t, tc, responses, interceptedRequest)
}

// validateRandomIntResponse validates the response from random int test
func validateRandomIntResponse(t *testing.T, tc randomIntTestCase, responses []*rc.Response,
	interceptedRequest *randomIntRequestData) {
	t.Helper()
	require.Len(t, responses, 1, "Expected 1 response for %s", tc.name)
	resp := responses[0]
	assert.NoError(t, resp.Error, "Response error should be nil for %s", tc.name)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status OK for %s", tc.name)

	actualURL := decodeURLIfNeeded(interceptedRequest.URL)
	tc.validate(t, actualURL, interceptedRequest.Header, interceptedRequest.Body)
}

// decodeURLIfNeeded decodes URL if it contains percent encoding
func decodeURLIfNeeded(rawURL string) string {
	if strings.Contains(rawURL, "%") {
		if decodedURL, err := url.PathUnescape(rawURL); err == nil {
			return decodedURL
		}
	}
	return rawURL
}

// PRD-COMMENT: G5 - Comprehensive Faker Library Support: Person/Identity Data
// Corresponds to: Client's ability to substitute comprehensive faker variables for person data
// including {{$randomFirstName}}, {{$randomLastName}}, {{$randomFullName}}, and {{$randomJobTitle}}.
// This test verifies that both VS Code style ({{$randomFirstName}}) and JetBrains style
// ({{$random.firstName}}) syntaxes work correctly and generate realistic person data.
func RunExecuteFile_WithFakerPersonData(t *testing.T) {
	t.Helper()
	// Given
	var interceptedHeaders []http.Header
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedHeaders = append(interceptedHeaders, r.Header.Clone())
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/system_variables/faker_person_data.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 2) // Two requests in the file
	require.Len(t, interceptedHeaders, 2) // Should have captured headers from both requests

	// Validate first request (VS Code style syntax)
	resp1 := responses[0]
	assert.NoError(t, resp1.Error)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	// Check that faker variables were substituted (not empty and not the placeholder)
	vsCodeHeaders := interceptedHeaders[0]
	firstName := vsCodeHeaders.Get("X-Random-First-Name")
	assert.NotEmpty(t, firstName, "First name should not be empty")
	assert.NotContains(t, firstName, "{{", "First name should not contain placeholder")
	assert.Len(t, strings.Fields(firstName), 1, "First name should be a single word")

	lastName := vsCodeHeaders.Get("X-Random-Last-Name")
	assert.NotEmpty(t, lastName, "Last name should not be empty")
	assert.NotContains(t, lastName, "{{", "Last name should not contain placeholder")
	assert.Len(t, strings.Fields(lastName), 1, "Last name should be a single word")

	fullName := vsCodeHeaders.Get("X-Random-Full-Name")
	assert.NotEmpty(t, fullName, "Full name should not be empty")
	assert.NotContains(t, fullName, "{{", "Full name should not contain placeholder")
	assert.Len(t, strings.Fields(fullName), 2, "Full name should contain first and last name")

	jobTitle := vsCodeHeaders.Get("X-Random-Job-Title")
	assert.NotEmpty(t, jobTitle, "Job title should not be empty")
	assert.NotContains(t, jobTitle, "{{", "Job title should not contain placeholder")

	// Validate second request (JetBrains style syntax)
	resp2 := responses[1]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Check JetBrains style faker variables were substituted
	jetBrainsHeaders := interceptedHeaders[1]
	firstNameDot := jetBrainsHeaders.Get("X-Random-First-Name-Dot")
	assert.NotEmpty(t, firstNameDot, "JetBrains first name should not be empty")
	assert.NotContains(t, firstNameDot, "{{", "JetBrains first name should not contain placeholder")
	assert.Len(t, strings.Fields(firstNameDot), 1, "JetBrains first name should be a single word")

	lastNameDot := jetBrainsHeaders.Get("X-Random-Last-Name-Dot")
	assert.NotEmpty(t, lastNameDot, "JetBrains last name should not be empty")
	assert.NotContains(t, lastNameDot, "{{", "JetBrains last name should not contain placeholder")
	assert.Len(t, strings.Fields(lastNameDot), 1, "JetBrains last name should be a single word")

	fullNameDot := jetBrainsHeaders.Get("X-Random-Full-Name-Dot")
	assert.NotEmpty(t, fullNameDot, "JetBrains full name should not be empty")
	assert.NotContains(t, fullNameDot, "{{", "JetBrains full name should not contain placeholder")
	assert.Len(t, strings.Fields(fullNameDot), 2, "JetBrains full name should contain first and last name")

	jobTitleDot := jetBrainsHeaders.Get("X-Random-Job-Title-Dot")
	assert.NotEmpty(t, jobTitleDot, "JetBrains job title should not be empty")
	assert.NotContains(t, jobTitleDot, "{{", "JetBrains job title should not contain placeholder")

	t.Logf("Generated person data - VS Code style: %s %s (%s)", firstName, lastName, jobTitle)
	t.Logf("Generated person data - JetBrains style: %s %s (%s)", firstNameDot, lastNameDot, jobTitleDot)
}

// PRD-COMMENT: G8 - Indirect Environment Variable Lookup: {{$processEnv %VAR}}
// Corresponds to: Client's ability to substitute indirect environment variables using the
// {{$processEnv %VAR}} syntax where VAR is a variable containing the name of the environment
// variable to look up. This provides dynamic environment variable selection.
func RunExecuteFile_WithIndirectEnvironmentVariables(t *testing.T) {
	t.Helper()
	// Given
	var interceptedHeaders http.Header
	var interceptedBody string
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedHeaders = r.Header.Clone()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedBody = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	// Set up environment variables for testing
	_ = os.Setenv("TEST_SECRET_KEY", "secret123")
	_ = os.Setenv("TEST_DATABASE_URL", "postgres://localhost:5432/test")
	_ = os.Setenv("PROD_ENV", "production")
	defer func() {
		_ = os.Unsetenv("TEST_SECRET_KEY")
		_ = os.Unsetenv("TEST_DATABASE_URL")
		_ = os.Unsetenv("PROD_ENV")
	}()

	// Set up programmatic variables that point to environment variable names
	client, _ := rc.NewClient(rc.WithVars(map[string]any{
		"secretKeyVar": "TEST_SECRET_KEY",
		"dbUrlVar":     "TEST_DATABASE_URL",
		"envVar":       "PROD_ENV",
		"missingVar":   "NONEXISTENT_ENV_VAR",
		// Note: undefinedVar is intentionally not defined
	}))
	requestFilePath := createTestFileFromTemplate(t, "test/data/system_variables/indirect_env_lookup.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Validate that indirect environment variables were correctly resolved
	secretKey := interceptedHeaders.Get("X-Secret-Key")
	assert.Equal(t, "secret123", secretKey, "Secret key should be resolved from TEST_SECRET_KEY")

	dbUrl := interceptedHeaders.Get("X-Database-URL")
	assert.Equal(t, "postgres://localhost:5432/test", dbUrl, "Database URL should be resolved from TEST_DATABASE_URL")

	// Missing environment variable should result in empty string
	missingVar := interceptedHeaders.Get("X-Missing-Var")
	assert.Equal(t, "", missingVar, "Missing environment variable should resolve to empty string")

	// Check body content
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedBody), &bodyJSON)
	require.NoError(t, err)

	// Environment variable that exists should be resolved
	assert.Equal(t, "production", bodyJSON["environment"], "Environment should be resolved from PROD_ENV")

	// Undefined variable in programmaticVars should remain as placeholder
	assert.Equal(t, "{{$processEnv %undefinedVar}}", bodyJSON["missing"], 
		"Undefined variable should remain as placeholder")

	t.Logf("Indirect environment variable resolution: secretKey=%s, dbUrl=%s, environment=%s", 
		secretKey, dbUrl, bodyJSON["environment"])
}

// PRD-COMMENT: G5 Phase 1 - Enhanced Faker Library: Contact and Internet Data
// Corresponds to: Client's ability to substitute enhanced faker variables for contact data
// (phone, address, city, state, zip, country) and internet data (URL, domain, user agent, MAC).
// This test verifies that both VS Code style and JetBrains style syntaxes work correctly
// and generate realistic contact and internet data for API testing scenarios.
func RunExecuteFile_WithContactAndInternetFakerData(t *testing.T) {
	t.Helper()
	// Given
	var interceptedHeaders []http.Header
	var interceptedBodies []string
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedHeaders = append(interceptedHeaders, r.Header.Clone())
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedBodies = append(interceptedBodies, string(bodyBytes))
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/system_variables/faker_contact_internet_data.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 2) // Two requests in the file
	require.Len(t, interceptedHeaders, 2) // Should have captured headers from both requests
	require.Len(t, interceptedBodies, 2) // Should have captured bodies from both requests

	// Validate first request (VS Code style syntax)
	resp1 := responses[0]
	assert.NoError(t, resp1.Error)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	// Check VS Code style contact data faker variables
	vsCodeHeaders := interceptedHeaders[0]
	phone := vsCodeHeaders.Get("X-Phone")
	assert.NotEmpty(t, phone, "Phone number should not be empty")
	assert.NotContains(t, phone, "{{", "Phone number should not contain placeholder")
	assert.Regexp(t, `^\(\d{3}\) \d{3}-\d{4}$`, phone, "Phone should match format (XXX) XXX-XXXX")

	address := vsCodeHeaders.Get("X-Address")
	assert.NotEmpty(t, address, "Address should not be empty")
	assert.NotContains(t, address, "{{", "Address should not contain placeholder")
	assert.Regexp(t, `^\d+ .+`, address, "Address should start with a number")

	city := vsCodeHeaders.Get("X-City")
	assert.NotEmpty(t, city, "City should not be empty")
	assert.NotContains(t, city, "{{", "City should not contain placeholder")

	state := vsCodeHeaders.Get("X-State")
	assert.NotEmpty(t, state, "State should not be empty")
	assert.NotContains(t, state, "{{", "State should not contain placeholder")

	zipCode := vsCodeHeaders.Get("X-Zip")
	assert.NotEmpty(t, zipCode, "ZIP code should not be empty")
	assert.NotContains(t, zipCode, "{{", "ZIP code should not contain placeholder")
	assert.Regexp(t, `^\d{5}$`, zipCode, "ZIP code should be 5 digits")

	country := vsCodeHeaders.Get("X-Country")
	assert.NotEmpty(t, country, "Country should not be empty")
	assert.NotContains(t, country, "{{", "Country should not contain placeholder")

	// Check VS Code style internet data faker variables
	testURL := vsCodeHeaders.Get("X-Url")
	assert.NotEmpty(t, testURL, "URL should not be empty")
	assert.NotContains(t, testURL, "{{", "URL should not contain placeholder")
	assert.Regexp(t, `^https?://`, testURL, "URL should start with http:// or https://")

	domain := vsCodeHeaders.Get("X-Domain")
	assert.NotEmpty(t, domain, "Domain should not be empty")
	assert.NotContains(t, domain, "{{", "Domain should not contain placeholder")
	assert.Contains(t, domain, ".", "Domain should contain a dot")

	userAgent := vsCodeHeaders.Get("X-User-Agent")
	assert.NotEmpty(t, userAgent, "User agent should not be empty")
	assert.NotContains(t, userAgent, "{{", "User agent should not contain placeholder")
	assert.Contains(t, userAgent, "Mozilla", "User agent should contain Mozilla")

	macAddress := vsCodeHeaders.Get("X-Mac")
	assert.NotEmpty(t, macAddress, "MAC address should not be empty")
	assert.NotContains(t, macAddress, "{{", "MAC address should not contain placeholder")
	assert.Regexp(t, `^[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}$`, 
		macAddress, "MAC address should match format XX:XX:XX:XX:XX:XX")

	// Validate second request (JetBrains style syntax)
	resp2 := responses[1]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Check JetBrains style faker variables
	jetBrainsHeaders := interceptedHeaders[1]
	phoneDot := jetBrainsHeaders.Get("X-Phone-Dot")
	assert.NotEmpty(t, phoneDot, "JetBrains phone number should not be empty")
	assert.NotContains(t, phoneDot, "{{", "JetBrains phone number should not contain placeholder")
	assert.Regexp(t, `^\(\d{3}\) \d{3}-\d{4}$`, phoneDot, "JetBrains phone should match format (XXX) XXX-XXXX")

	addressDot := jetBrainsHeaders.Get("X-Address-Dot")
	assert.NotEmpty(t, addressDot, "JetBrains address should not be empty")
	assert.NotContains(t, addressDot, "{{", "JetBrains address should not contain placeholder")

	cityDot := jetBrainsHeaders.Get("X-City-Dot")
	assert.NotEmpty(t, cityDot, "JetBrains city should not be empty")
	assert.NotContains(t, cityDot, "{{", "JetBrains city should not contain placeholder")

	urlDot := jetBrainsHeaders.Get("X-Url-Dot")
	assert.NotEmpty(t, urlDot, "JetBrains URL should not be empty")
	assert.NotContains(t, urlDot, "{{", "JetBrains URL should not contain placeholder")
	assert.Regexp(t, `^https?://`, urlDot, "JetBrains URL should start with http:// or https://")

	macAddressDot := jetBrainsHeaders.Get("X-Mac-Dot")
	assert.NotEmpty(t, macAddressDot, "JetBrains MAC address should not be empty")
	assert.NotContains(t, macAddressDot, "{{", "JetBrains MAC address should not contain placeholder")
	assert.Regexp(t, `^[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}$`, 
		macAddressDot, "JetBrains MAC address should match format XX:XX:XX:XX:XX:XX")

	// Validate JSON body content for both requests
	for i, body := range interceptedBodies {
		var bodyJSON map[string]any
		err := json.Unmarshal([]byte(body), &bodyJSON)
		require.NoError(t, err, "Request %d body should be valid JSON", i+1)

		contact, ok := bodyJSON["contact"].(map[string]any)
		require.True(t, ok, "Request %d should have contact object", i+1)

		contactPhone, ok := contact["phone"].(string)
		require.True(t, ok, "Request %d should have contact phone", i+1)
		assert.NotContains(t, contactPhone, "{{", "Request %d contact phone should not contain placeholder", i+1)

		address, ok := contact["address"].(map[string]any)
		require.True(t, ok, "Request %d should have address object", i+1)

		street, ok := address["street"].(string)
		require.True(t, ok, "Request %d should have address street", i+1)
		assert.NotContains(t, street, "{{", "Request %d address street should not contain placeholder", i+1)

		technical, ok := bodyJSON["technical"].(map[string]any)
		require.True(t, ok, "Request %d should have technical object", i+1)

		website, ok := technical["website"].(string)
		require.True(t, ok, "Request %d should have technical website", i+1)
		assert.NotContains(t, website, "{{", "Request %d technical website should not contain placeholder", i+1)
		assert.Regexp(t, `^https?://`, website, "Request %d website should be a valid URL", i+1)
	}

	t.Logf("Generated contact data - VS Code style: %s, %s, %s, %s", phone, address, city, state)
	t.Logf("Generated internet data - VS Code style: %s, %s, %s", testURL, domain, macAddress)
	t.Logf("Generated contact data - JetBrains style: %s, %s, %s", phoneDot, addressDot, cityDot)
	t.Logf("Generated internet data - JetBrains style: %s, %s", urlDot, macAddressDot)
}
