package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	// With request-scoped system variables, all {{$guid}} ({{$uuid}}) instances should resolve to the SAME value.
	assert.Equal(t, guidFromURL, guidFromHeader, "GUID from URL and header should be the same")
	assert.Equal(t, guidFromURL, guidFromBody1, "GUID from URL and body1 should be the same")
	// For this test, the .http file uses {{$guid}} twice in the body for different fields.
	// These should now resolve to the same request-scoped GUID.
	assert.Equal(t, guidFromBody1, guidFromBody2, "GUIDs from body (transactionId and correlationId) should be the same")
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
		URL    string
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Env-Value")
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
	assert.Empty(t, undefinedPayload, "Body undefined_payload should be empty for an undefined env variable")
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
				assert.True(t, valURL >= 0 && valURL <= 100, "URL random int (no args) %d out of range [0,100]", valURL)

				valHeader, err := strconv.Atoi(header)
				require.NoError(t, err, "Random int from Header (no args) should be valid int")
				assert.True(t, valHeader >= 0 && valHeader <= 100, "Header random int (no args) %d out of range [0,100]", valHeader)

				var bodyJSON map[string]int
				err = json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body (no args)")
				assert.True(t, bodyJSON["value"] >= 0 && bodyJSON["value"] <= 100, "Body random int (no args) %d out of range [0,100]", bodyJSON["value"])
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
