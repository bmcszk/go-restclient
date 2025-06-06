package restclient

import (
	"bufio"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type parseRequestsIgnoreEmptyBlocksTestCase struct {
	name           string
	filePath       string // Changed from fileContent
	expectedCount  int
	expectedError  bool
	firstReqMethod string // if expectedCount > 0
	firstReqURL    string // if expectedCount > 0
	lastReqMethod  string // if expectedCount > 1
	lastReqURL     string // if expectedCount > 1
}

func runParseRequestsIgnoreEmptyBlocksSubtest(t *testing.T, tc parseRequestsIgnoreEmptyBlocksTestCase) {
	t.Helper()
	// Given: test case setup (reading from tc.filePath)
	file, err := os.Open(tc.filePath)
	require.NoError(t, err, "Failed to open test data file: %s for test: %s", tc.filePath, tc.name)
	defer file.Close()
	reader := bufio.NewReader(file)

	// When: parsing the request file
	// Using nil for client, empty importStack, empty initialFileVariables, dummy envLookup, empty globalVariables, and nil initialRequests
	// as these are not relevant for testing parseRequests in isolation for ignoring empty blocks.
	parsedFile, err := ParseRequests(reader, "test.http", nil, make(map[string]string), func(string) (string, bool) { return "", false }, make(map[string]string), nil)

	// Then: assert expected outcomes
	if tc.expectedError {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
		require.NotNil(t, parsedFile, "parsedFile should not be nil on no error for test: %s", tc.name)
		assert.Len(t, parsedFile.Requests, tc.expectedCount, "Number of parsed requests mismatch for test: %s", tc.name)

		if tc.expectedCount > 0 && len(parsedFile.Requests) > 0 {
			assert.Equal(t, tc.firstReqMethod, parsedFile.Requests[0].Method, "First request method mismatch for test: %s", tc.name)
			assert.Equal(t, tc.firstReqURL, parsedFile.Requests[0].RawURLString, "First request URL mismatch for test: %s", tc.name)
		}
		if tc.expectedCount > 1 && len(parsedFile.Requests) > 1 {
			assert.Equal(t, tc.lastReqMethod, parsedFile.Requests[tc.expectedCount-1].Method, "Last request method mismatch for test: %s", tc.name)
			assert.Equal(t, tc.lastReqURL, parsedFile.Requests[tc.expectedCount-1].RawURLString, "Last request URL mismatch for test: %s", tc.name)
		}
	}
}

func TestParseRequests_IgnoreEmptyBlocks(t *testing.T) {
	tests := []parseRequestsIgnoreEmptyBlocksTestCase{
		{
			name:          "SCENARIO-LIB-028-001: File with only comments",
			filePath:      "testdata/parser/ignore_empty_blocks/scenario_001_only_comments.http",
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:          "SCENARIO-LIB-028-002: File with only ### separators",
			filePath:      "testdata/parser/ignore_empty_blocks/scenario_002_only_separators.http",
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:          "SCENARIO-LIB-028-003: File with comments and ### separators only",
			filePath:      "testdata/parser/ignore_empty_blocks/scenario_003_comments_and_separators.http",
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:           "SCENARIO-LIB-028-004: Valid request, then separator, then only comments",
			filePath:       "testdata/parser/ignore_empty_blocks/scenario_004_valid_then_empty_comment.http",
			expectedCount:  1,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "https://example.com/first",
		},
		{
			name:           "SCENARIO-LIB-028-005: Only comments, then separator, then valid request",
			filePath:       "testdata/parser/ignore_empty_blocks/scenario_005_empty_comment_then_valid.http",
			expectedCount:  1,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "https://example.com/second",
		},
		{
			name:           "SCENARIO-LIB-028-006: Valid request, separator with comments, then another valid request",
			filePath:       "testdata/parser/ignore_empty_blocks/scenario_006_two_valid_with_empty_comment.http",
			expectedCount:  2,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "https://example.com/req1",
			lastReqMethod:  "POST",
			lastReqURL:     "https://example.com/req2",
		},
		{
			name:          "Empty file content",
			filePath:      "testdata/parser/ignore_empty_blocks/empty_file_content.http",
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:           "Single valid request no trailing newline",
			filePath:       "testdata/parser/ignore_empty_blocks/single_valid_no_newline.http",
			expectedCount:  1,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "http://localhost/api/test",
		},
		{
			name:          "File with only variable definitions",
			filePath:      "testdata/parser/ignore_empty_blocks/only_variables.http",
			expectedCount: 0,
			expectedError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentTest := tt
			runParseRequestsIgnoreEmptyBlocksSubtest(t, currentTest)
		})
	}
}

func TestParseRequests_SeparatorComments(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		expectedCount int
		req1Method    string
		req1URL       string
		req2Method    string
		req2URL       string
	}{
		{
			name: "SCENARIO-LIB-027-001 & SCENARIO-LIB-027-004 combined",
			fileContent: `
GET https://example.com/api/resource1

### This is a comment for the first request block

POST https://example.com/api/resource2

### This line is just a separator comment

PUT https://example.com/api/resource3
`, // SCENARIO-LIB-027-003 style is implicitly handled by line-by-line parsing.
			expectedCount: 3,
			req1Method:    "GET",
			req1URL:       "https://example.com/api/resource1",
			// req2 is POST to /resource2, req3 is PUT to /resource3
		},
		{
			name: "SCENARIO-LIB-027-003 style: Separator comment no newline before next request",
			fileContent: `
GET https://example.com/api/item1 ### Comment for item1
POST https://example.com/api/item2
`,
			expectedCount: 2,
			req1Method:    "GET",
			req1URL:       "https://example.com/api/item1",
			req2Method:    "POST",
			req2URL:       "https://example.com/api/item2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: test case setup (input string tt.fileContent)
			reader := strings.NewReader(tt.fileContent)

			// When: parsing the request file
			parsedFile, err := ParseRequests(reader, "test.http", nil, make(map[string]string), func(string) (string, bool) { return "", false }, make(map[string]string), nil)

			// Then: assert expected outcomes
			require.NoError(t, err)
			require.NotNil(t, parsedFile)
			assert.Len(t, parsedFile.Requests, tt.expectedCount)

			if tt.expectedCount > 0 && len(parsedFile.Requests) > 0 {
				assert.Equal(t, tt.req1Method, parsedFile.Requests[0].Method)
				assert.Equal(t, tt.req1URL, parsedFile.Requests[0].RawURLString)
			}
			if tt.expectedCount > 1 && len(parsedFile.Requests) > 1 {
				// For simplicity, checking specific requests based on scenario structure
				switch tt.name {
				case "SCENARIO-LIB-027-001 & SCENARIO-LIB-027-004 combined":
					assert.Equal(t, "POST", parsedFile.Requests[1].Method)
					assert.Equal(t, "https://example.com/api/resource2", parsedFile.Requests[1].RawURLString)
					assert.Equal(t, "PUT", parsedFile.Requests[2].Method)
					assert.Equal(t, "https://example.com/api/resource3", parsedFile.Requests[2].RawURLString)
				case "SCENARIO-LIB-027-003 style: Separator comment no newline before next request":
					assert.Equal(t, tt.req2Method, parsedFile.Requests[1].Method)
					assert.Equal(t, tt.req2URL, parsedFile.Requests[1].RawURLString)
				}
			}
		})
	}
}

func TestParseExpectedResponses_SeparatorComments(t *testing.T) {
	tests := []struct {
		name           string
		fileContent    string
		expectedCount  int
		expResp1Status string
		expResp2Status string
	}{
		{
			name: "SCENARIO-LIB-027-002: Separator comment in response file",
			fileContent: `
HTTP/1.1 200 OK
Content-Type: application/json

{"status": "success"}

### This is a comment for the first response block

HTTP/1.1 404 Not Found
`,
			expectedCount:  2,
			expResp1Status: "200 OK",
			expResp2Status: "404 Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: test case setup (input string tt.fileContent)
			reader := strings.NewReader(tt.fileContent)

			// When: parsing the expected responses
			parsedResponses, err := parseExpectedResponses(reader, "test.hresp")

			// Then: assert expected outcomes
			require.NoError(t, err)
			require.NotNil(t, parsedResponses)
			assert.Len(t, parsedResponses, tt.expectedCount)

			if tt.expectedCount > 0 && len(parsedResponses) > 0 {
				require.NotNil(t, parsedResponses[0].Status)
				assert.Equal(t, tt.expResp1Status, *parsedResponses[0].Status)
			}
			if tt.expectedCount > 1 && len(parsedResponses) > 1 {
				require.NotNil(t, parsedResponses[1].Status)
				assert.Equal(t, tt.expResp2Status, *parsedResponses[1].Status)
			}
		})
	}
}

func assertSingleExpectedResponseValid(t *testing.T, resp *ExpectedResponse, expectedStatus *string, expectedHeaders map[string]string, expectedBodyJSON string, expectedBody *string) {
	t.Helper()
	if expectedStatus != nil {
		require.NotNil(t, resp.Status, "Response status should not be nil when expectedStatus is provided")
		assert.Equal(t, *expectedStatus, *resp.Status)
	}
	for k, v := range expectedHeaders {
		assert.Equal(t, v, resp.Headers.Get(k))
	}
	if expectedBodyJSON != "" {
		require.NotNil(t, resp.Body, "Response body should not be nil when expectedBodyJSON is provided")
		assert.JSONEq(t, expectedBodyJSON, *resp.Body)
	} else if expectedBody != nil {
		require.NotNil(t, resp.Body, "Response body should not be nil when expectedBody is provided")
		assert.Equal(t, *expectedBody, *resp.Body)
	}
}

func TestParseExpectedResponses_Simple(t *testing.T) {
	// Given
	tests := []struct {
		name             string
		content          string
		expectedCount    int
		expectedStatus   *string
		expectedHeaders  map[string]string
		expectedBodyJSON string
		expectedBody     *string
		expectError      bool
	}{
		{
			name: "SCENARIO-LIB-007-001: Full valid response",
			content: `HTTP/1.1 200 OK
Content-Type: application/json
X-Test-Header: TestValue

{
  "message": "success"
}`,
			expectedCount:  1,
			expectedStatus: ptr("200 OK"),
			expectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"X-Test-Header": "TestValue",
			},
			expectedBodyJSON: `{"message": "success"}`,
		},
		{
			name:           "SCENARIO-LIB-007-002: Status line only",
			content:        `HTTP/1.1 404 Not Found`,
			expectedCount:  1,
			expectedStatus: ptr("404 Not Found"),
		},
		{
			name: "SCENARIO-LIB-007-003: Status and headers only",
			content: `HTTP/1.1 201 Created
Cache-Control: no-cache`,
			expectedCount:  1,
			expectedStatus: ptr("201 Created"),
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache",
			},
		},
		{
			name: "SCENARIO-LIB-007-004: Status and body only",
			content: `HTTP/1.1 500 Internal Server Error

<error>Server Error</error>`,
			expectedCount:  1,
			expectedStatus: ptr("500 Internal Server Error"),
			expectedBody:   ptr("<error>Server Error</error>"),
		},
		{
			name:          "SCENARIO-LIB-007-005: Empty content",
			content:       ``,
			expectedCount: 0,
			expectError:   false, // Empty content is not an error, just no responses.
		},
		{
			name:          "SCENARIO-LIB-007-006: Malformed status line",
			content:       `HTTP/1.1OK`, // No space
			expectedCount: 0,
			expectError:   true,
		},
		{
			name: "Multiple responses",
			content: `HTTP/1.1 200 OK
Content-Type: text/plain

Response 1
###
HTTP/1.1 201 Created

Response 2`,
			expectedCount: 2,
			// We'll just check count for this multi-response, details in _SeparatorComments
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			reader := strings.NewReader(tt.content)

			// When
			parsedResponses, err := parseExpectedResponses(reader, "test.hresp")

			// Then
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, parsedResponses, tt.expectedCount)

			if tt.expectedCount == 1 && len(parsedResponses) == 1 {
				resp := parsedResponses[0]
				assertSingleExpectedResponseValid(t, resp, tt.expectedStatus, tt.expectedHeaders, tt.expectedBodyJSON, tt.expectedBody)
			}
		})
	}
}

func assertErrorExpectedInParseRequestFile(t *testing.T, err error, parsedFile *ParsedFile, testName string, errorContains string) {
	t.Helper()
	require.Error(t, err, "Expected an error for test case: %s", testName)
	if errorContains != "" {
		assert.Contains(t, err.Error(), errorContains, "Error message mismatch for test case: %s", testName)
	}
	assert.Nil(t, parsedFile, "parsedFile should be nil on error for test case: %s", testName)
}

func assertNoErrorExpectedInParseRequestFile(t *testing.T, err error, parsedFile *ParsedFile, testName string, expectedRequests int, expectedVariables map[string]string, requestChecks func(t *testing.T, testName string, requests []*Request, fileVars map[string]string)) {
	t.Helper()
	require.NoError(t, err, "Did not expect an error for test case: %s", testName)
	require.NotNil(t, parsedFile, "parsedFile should not be nil on no error for test case: %s", testName)

	if parsedFile != nil { // Defensive check, though NotNil was already asserted
		assert.Len(t, parsedFile.Requests, expectedRequests, "Number of parsed requests mismatch for test case: %s", testName)

		// Check merged file-level variables
		for key, expectedValue := range expectedVariables {
			actualValue, exists := parsedFile.FileVariables[key]
			assert.True(t, exists, "Expected file variable '%s' not found in test case: %s", key, testName)
			assert.Equal(t, expectedValue, actualValue, "Value for file variable '%s' mismatch in test case: %s", key, testName)
		}

		if requestChecks != nil {
			requestChecks(t, testName, parsedFile.Requests, parsedFile.FileVariables)
		}
	}
}

// Definition for the test case structure for TestParseRequestFile_Imports
type parseRequestFileImportsTestCase struct {
	name              string
	filePath          string
	expectedRequests  int
	expectedVariables map[string]string
	requestChecks     func(t *testing.T, testName string, requests []*Request, fileVariables map[string]string)
	expectError       bool
	errorContains     string
}

// Helper to check requests for SCENARIO-IMPORT-001: Simple import
func checkSimpleImportRequests(t *testing.T, testName string, requests []*Request, fileVariables map[string]string) {
	t.Helper()
	require.Len(t, requests, 2)
	// Request from imported_file_1.http
	assert.Equal(t, "GET", requests[0].Method)
	assert.Equal(t, "{{host}}/imported_request_1", requests[0].RawURLString)
	bodyReader, err := requests[0].GetBody()
	require.NoError(t, err, "Failed to get body for request 0 in simple import test case: %s", testName)
	require.NotNil(t, bodyReader, "Body reader should not be nil for request 0 in simple import test case: %s", testName)
	defer bodyReader.Close()

	bodyBytes, err := io.ReadAll(bodyReader)
	require.NoError(t, err, "Failed to read body for request 0 in simple import test case: %s", testName)
	bodyString := string(bodyBytes)
	assert.Contains(t, bodyString, "\"key\": \"{{imported_var}}\"")

	// Request from main_simple_import.http
	assert.Equal(t, "GET", requests[1].Method)
	assert.Equal(t, "{{host}}/main_request", requests[1].RawURLString)
}

// Helper to check requests for SCENARIO-IMPORT-002: Nested import
func checkNestedImportRequests(t *testing.T, testName string, requests []*Request, fileVariables map[string]string) {
	t.Helper()
	require.Len(t, requests, 3)
	// Requests from imported_file_3_level_2.http (innermost)
	assert.Equal(t, "GET", requests[0].Method)
	assert.Equal(t, "{{host}}/level2_request", requests[0].RawURLString)
	bodyReader, err := requests[0].GetBody()
	require.NoError(t, err, "Failed to get body for request 0 in nested import test case: %s", testName)
	require.NotNil(t, bodyReader, "Body reader should not be nil for request 0 in nested import test case: %s", testName)
	defer bodyReader.Close()

	bodyBytes, err := io.ReadAll(bodyReader)
	require.NoError(t, err, "Failed to read body for request 0 in nested import test case: %s", testName)
	bodyString := string(bodyBytes)
	assert.Contains(t, bodyString, "\"level1_key\": \"{{level1_var}}\"")
	assert.Contains(t, bodyString, "\"level2_key\": \"{{level2_var}}\"")

	// Request from imported_file_2_level_1.http
	assert.Equal(t, "GET", requests[1].Method)
	assert.Equal(t, "{{host}}/level1_request", requests[1].RawURLString)

	// Request from main_nested_import.http (outermost)
	assert.Equal(t, "GET", requests[2].Method)
	assert.Equal(t, "{{host}}/main_nested_request", requests[2].RawURLString)
}

// Helper to check requests for SCENARIO-IMPORT-003: Variable override
func checkVariableOverrideRequests(t *testing.T, testName string, requests []*Request, fileVariables map[string]string) {
	t.Helper()
	require.Len(t, requests, 1)
	assert.Equal(t, "GET", requests[0].Method)
	assert.Equal(t, "{{host}}/override", requests[0].RawURLString)
	assert.Equal(t, "{{var1}}", requests[0].Headers.Get("X-Var1"))
	assert.Equal(t, "{{var2}}", requests[0].Headers.Get("X-Var2"))
	assert.Equal(t, "{{var_from_main}}", requests[0].Headers.Get("X-Var-Main"))
}

// Helper subtest runner for TestParseRequestFile_Imports
func runParseRequestFileImportsSubtest(t *testing.T, tc parseRequestFileImportsTestCase) {
	t.Helper()
	// Given: client can be nil as we are testing parsing, not execution.
	// Initial importStack is empty for top-level calls.
	parsedFile, err := parseRequestFile(tc.filePath, nil, make([]string, 0))

	// Then: assert expected outcomes
	if tc.expectError {
		assertErrorExpectedInParseRequestFile(t, err, parsedFile, tc.name, tc.errorContains)
	} else {
		assertNoErrorExpectedInParseRequestFile(t, err, parsedFile, tc.name, tc.expectedRequests, tc.expectedVariables, tc.requestChecks)
	}
}

func TestParseRequestFile_Imports(t *testing.T) {
	// Mock environment variable lookup function for tests
	// envLookup := func(key string) (string, bool) { // This was part of parseRequests, not parseRequestFile directly
	// 	return "", false
	// }

	tests := []parseRequestFileImportsTestCase{

		{
			name:             "SCENARIO-IMPORT-001: Simple import",
			filePath:         "testdata/parser/import_tests/main_simple_import.http",
			expectedRequests: 2,
			expectedVariables: map[string]string{
				"host":         "http://localhost:8080",
				"imported_var": "imported_value",
			},
			requestChecks: checkSimpleImportRequests,
		},
		{
			name:             "SCENARIO-IMPORT-002: Nested import",
			filePath:         "testdata/parser/import_tests/main_nested_import.http",
			expectedRequests: 3,
			expectedVariables: map[string]string{
				"host":       "http://localhost:8080", // from imported_file_3_level_2.http
				"level1_var": "level1_value",          // from imported_file_2_level_1.http
				"level2_var": "level2_value",          // from imported_file_3_level_2.http
			},
			requestChecks: checkNestedImportRequests,
		},
		{
			name:             "SCENARIO-IMPORT-003: Variable override",
			filePath:         "testdata/parser/import_tests/main_variable_override.http",
			expectedRequests: 1,
			expectedVariables: map[string]string{
				"host":          "http://main-override.com", // Overridden by main file
				"var1":          "value1_from_imported",
				"var2":          "value2_from_imported",
				"var_from_main": "main_value", // Defined in main file
			},
			requestChecks: checkVariableOverrideRequests,
		},
		{
			name:          "SCENARIO-IMPORT-004: Circular import",
			filePath:      "testdata/parser/import_tests/main_circular_import_a.http",
			expectError:   true,
			errorContains: "circular import detected",
		},
		{
			name:        "SCENARIO-IMPORT-005: Import not found",
			filePath:    "testdata/parser/import_tests/main_import_not_found.http",
			expectError: true,
			// error message for os.Open on a non-existent file typically includes "no such file or directory"
			errorContains: "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentTest := tt // Capture range variable
			runParseRequestFileImportsSubtest(t, currentTest)
		})
	}
}
