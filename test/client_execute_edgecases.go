package test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath" // Added
	"testing"

	rc "github.com/bmcszk/go-restclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR11.1 - Client Execution: Invalid HTTP Method Handling
// Corresponds to: Client robustness in handling syntactically incorrect HTTP methods 
// within a .http file (http_syntax.md, implicitly by defining valid methods).
// This test verifies that the client correctly identifies an invalid HTTP method in 
// 'test/data/http_request_files/invalid_method.http', reports an error for that request, 
// and handles the overall execution flow appropriately (e.g., by aggregating errors).
func RunExecuteFile_InvalidMethodInFile(t *testing.T) {
	t.Helper()
	// Given
	client, _ := rc.NewClient()
	requestFilePath := "test/data/http_request_files/invalid_method.http"

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 error occurred:")
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
	assert.Contains(t, err.Error(), 
		"request 1 (INVALIDMETHOD /test) processing resulted in error")

	require.Len(t, responses, 1)

	resp1 := responses[0]
	assert.Error(t, resp1.Error, "Expected an error for invalid method/scheme")
	assert.Contains(t, resp1.Error.Error(), "unsupported protocol scheme", 
		"Error message should indicate unsupported protocol scheme")
	assert.Contains(t, resp1.Error.Error(), "Invalidmethod", 
		"Error message should contain the problematic method string as used")
}

// executeFileTestCase defines a test case for TestExecuteFile_IgnoreEmptyBlocks_Client
type executeFileTestCase struct {
	name                           string
	requestFileBasePath            string // Relative path to the base .http file in test/data
	needsServerURLCount            int    // How many times server.URL needs to be Sprintf'd (0, 1, or 2)
	expectedResponses              int
	expectedError                  bool
	expectedErrorMessageSubstrings []string
	responseValidators             []func(t *testing.T, resp *rc.Response)
}

func assertSuccessfulExecutionAndValidateResponses(t *testing.T, tcName string, execErr error, 
	actualResponses []*rc.Response, expectedResponseCount int, 
	responseValidators []func(t *testing.T, resp *rc.Response)) {
	t.Helper()
	assert.NoError(t, execErr, "Did not expect an error for test: %s", tcName)
	require.Len(t, actualResponses, expectedResponseCount, 
		"Number of responses mismatch for test: %s", tcName)
	if len(responseValidators) != expectedResponseCount {
		t.Fatalf("Mismatch between expected responses (%d) and number of validators (%d) for test: %s", 
			expectedResponseCount, len(responseValidators), tcName)
	}
	for i, validator := range responseValidators {
		if i < len(actualResponses) {
			validator(t, actualResponses[i])
		} else {
			t.Errorf("Validator index %d out of bounds for responses (len %d) in test: %s", 
				i, len(actualResponses), tcName)
		}
	}
}

func runExecuteFileSubtest(t *testing.T, client *rc.Client, serverURL string, tc executeFileTestCase) {
	t.Helper() // Mark as test helper

	// Read the base content from the testdata file
	baseContent, err := os.ReadFile(tc.requestFileBasePath)
	require.NoError(t, err, "Failed to read base request file %s for test: %s", tc.requestFileBasePath, tc.name)

	// Prepare the actual request content by injecting the server URL if needed
	var requestFileContent string
	switch tc.needsServerURLCount {
	case 0:
		requestFileContent = string(baseContent)
	case 1:
		requestFileContent = fmt.Sprintf(string(baseContent), serverURL)
	case 2:
		requestFileContent = fmt.Sprintf(string(baseContent), serverURL, serverURL)
	default:
		t.Fatalf("Invalid needsServerURLCount %d for test: %s", tc.needsServerURLCount, tc.name)
	}

	// Create a temporary file to write the processed request content
	tempFile, err := os.CreateTemp(t.TempDir(), filepath.Base(tc.requestFileBasePath)+".*.http") // Use filepath.Base
	require.NoError(t, err, "Failed to create temp file for test: %s", tc.name)
	// TempDir will handle cleanup of tempFile.

	_, err = tempFile.WriteString(requestFileContent)
	require.NoError(t, err, "Failed to write to temp file for test: %s", tc.name)
	require.NoError(t, tempFile.Close(), "Failed to close temp file for test: %s", tc.name)

	// When
	responses, execErr := client.ExecuteFile(context.Background(), tempFile.Name())

	// Then
	if tc.expectedError {
		assert.Error(t, execErr, "Expected an error for test: %s", tc.name)
		for _, sub := range tc.expectedErrorMessageSubstrings {
			assert.Contains(t, execErr.Error(), sub, "Error message for test '%s' should contain '%s'", tc.name, sub)
		}
	} else {
		assertSuccessfulExecutionAndValidateResponses(
			t, tc.name, execErr, responses, tc.expectedResponses, tc.responseValidators)
	}
}

// setupIgnoreEmptyBlocksMockServer sets up a mock HTTP server for TestExecuteFile_IgnoreEmptyBlocks_Client.
func setupIgnoreEmptyBlocksMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return startMockServer(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/first":
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response from /first")
		case "/second":
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response from /second")
		case "/req1":
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprint(w, "response from /req1")
		case "/req2":
			assert.Equal(t, http.MethodPost, r.Method)
			bodyBytes, _ := io.ReadAll(r.Body)
			assert.JSONEq(t, `{"key": "value"}`, string(bodyBytes))
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, "response from /req2")
		default:
			t.Errorf("Unexpected request to mock server: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// PRD-COMMENT: FR2.4 - Parser: Request Separators and Comments /
// FR11.2 - Client Execution: Handling of Non-Request Content
// Corresponds to: Client and parser behavior regarding non-executable content within .http files,
// such as comments, empty blocks between request separators ('###'), and files containing only
// variable definitions (http_syntax.md "Request Separation", "Comments", "Variables").
// This test suite verifies various scenarios:
// 1. Requests correctly parsed and executed when separated by comments or empty lines around separators.
// 2. Correct error handling (e.g., 'no requests found') for files that only contain variable
// definitions or are otherwise empty of executable requests.
// It uses test case templates from 'test/data/execute_file_ignore_empty_blocks/'
// (e.g., 'scenario_004_template.http', 'only_vars.http') to dynamically create test files.
func RunExecuteFile_IgnoreEmptyBlocks_Client(t *testing.T) {
	t.Helper()
	// Given common setup for all subtests
	server := setupIgnoreEmptyBlocksMockServer(t)
	defer server.Close()
	client, _ := rc.NewClient()

	testDataDir := "test/data/execute_file_ignore_empty_blocks"

	tests := []executeFileTestCase{
		{
			name:                "SCENARIO-LIB-028-004: Valid request, then separator, then only comments",
			requestFileBasePath: filepath.Join(testDataDir, "scenario_004_template.http"),
			needsServerURLCount: 1,
			expectedResponses:   1,
			expectedError:       false,
			responseValidators: []func(t *testing.T, resp *rc.Response){
				func(t *testing.T, resp *rc.Response) {
					t.Helper()
					assert.NoError(t, resp.Error)
					assert.Equal(t, http.StatusOK, resp.StatusCode)
					assert.Equal(t, "response from /first", resp.BodyString)
				},
			},
		},
		{
			name:                "SCENARIO-LIB-028-005: Only comments, then separator, then valid request",
			requestFileBasePath: filepath.Join(testDataDir, "scenario_005_template.http"),
			needsServerURLCount: 1,
			expectedResponses:   1,
			expectedError:       false,
			responseValidators: []func(t *testing.T, resp *rc.Response){
				func(t *testing.T, resp *rc.Response) {
					t.Helper()
					assert.NoError(t, resp.Error)
					assert.Equal(t, http.StatusOK, resp.StatusCode)
					assert.Equal(t, "response from /second", resp.BodyString)
				},
			},
		},
		{
			name: "SCENARIO-LIB-028-006: Valid request, separator with comments, " +
				"then another valid request",
			requestFileBasePath: filepath.Join(testDataDir, "scenario_006_template.http"),
			needsServerURLCount: 2,
			expectedResponses:   2,
			expectedError:       false,
			responseValidators: []func(t *testing.T, resp *rc.Response){
				func(t *testing.T, resp *rc.Response) { // For GET /req1
					t.Helper()
					assert.NoError(t, resp.Error)
					assert.Equal(t, http.StatusAccepted, resp.StatusCode)
					assert.Equal(t, "response from /req1", resp.BodyString)
				},
				func(t *testing.T, resp *rc.Response) { // For POST /req2
					t.Helper()
					assert.NoError(t, resp.Error)
					assert.Equal(t, http.StatusCreated, resp.StatusCode)
					assert.Equal(t, "response from /req2", resp.BodyString)
				},
			},
		},
		{
			name:                           "File with only variable definitions - ExecuteFile",
			requestFileBasePath:            filepath.Join(testDataDir, "only_vars.http"),
			needsServerURLCount:            0,
			expectedResponses:              0,
			expectedError:                  true,
			expectedErrorMessageSubstrings: []string{"no requests found in file"},
			responseValidators:             []func(t *testing.T, resp *rc.Response){},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			runExecuteFileSubtest(t, client, server.URL, tt)
		})
	}
}
