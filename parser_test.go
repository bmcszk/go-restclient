package restclient

import (
	"bufio"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(h))
	os.Exit(m.Run())
}

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
	parsedFile, err := parseRequests(reader, "test.http", nil, make(map[string]string), func(string) (string, bool) { return "", false }, make(map[string]string), nil)

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
			bufReader := bufio.NewReader(reader)

			// When: parsing the request file
			parsedFile, err := parseRequests(bufReader, "test.http", nil, make(map[string]string), func(string) (string, bool) { return "", false }, make(map[string]string), nil)

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

// assertErrorExpectedInParseRequestFile has been removed as it's no longer used

// assertNoErrorExpectedInParseRequestFile has been removed as it's no longer used
// since all import test cases now expect errors

// Definition for the test case structure for TestParseRequestFile_Imports
type parseRequestFileImportsTestCase struct {
	name     string
	filePath string
}

// Helper subtest runner for TestParseRequestFile_Imports
func runParseRequestFileImportsSubtest(t *testing.T, tc parseRequestFileImportsTestCase) {
	t.Helper()
	// Given: client can be nil as we are testing parsing, not execution.
	// Initial importStack is empty for top-level calls.
	parsedFile, err := parseRequestFile(tc.filePath, nil, make([]string, 0))

	// Then: @import directives are now silently ignored, so we expect no errors
	assert.NoError(t, err, "No error should occur when parsing file with @import directives (they're silently ignored)")
	assert.NotNil(t, parsedFile, "Parsed file should not be nil")
}

func TestParseRequestFile_Imports(t *testing.T) {
	// All import directives are now silently ignored since they're not documented in http_syntax.md

	tests := []parseRequestFileImportsTestCase{
		{
			name:     "SCENARIO-IMPORT-001: Simple import - ignored",
			filePath: "testdata/parser/import_tests/main_simple_import.http",
		},
		{
			name:     "SCENARIO-IMPORT-002: Nested import - ignored",
			filePath: "testdata/parser/import_tests/main_nested_import.http",
		},
		{
			name:     "SCENARIO-IMPORT-003: Variable override - ignored",
			filePath: "testdata/parser/import_tests/main_variable_override.http",
		},
		{
			name:     "SCENARIO-IMPORT-004: Circular import - ignored",
			filePath: "testdata/parser/import_tests/main_circular_import_a.http",
		},
		{
			name:     "SCENARIO-IMPORT-005: Import not found - ignored",
			filePath: "testdata/parser/import_tests/main_import_not_found.http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentTest := tt // Capture range variable
			runParseRequestFileImportsSubtest(t, currentTest)
		})
	}
}
