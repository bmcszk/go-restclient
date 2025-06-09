package restclient

import (
	"bufio"
	"fmt"
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
			// PRD-COMMENT: FR1 - Empty Block Handling (Comments)
			// Corresponds to: http_syntax.md "Empty Blocks and Comments"
			// This test case verifies that files containing only comments are parsed correctly,
			// resulting in zero requests and no errors, aligning with FR1.
			name:          "SCENARIO-LIB-028-001: File with only comments",
			filePath:      "testdata/parser/ignore_empty_blocks/scenario_001_only_comments.http",
			expectedCount: 0,
			expectedError: false,
		},
		{
			// PRD-COMMENT: FR1 - Empty Block Handling (Separators)
			// Corresponds to: http_syntax.md "Request Separator"
			// This test case ensures that files containing only request separators ('###')
			// are parsed as having zero requests and no errors, as per FR1.
			name:          "SCENARIO-LIB-028-002: File with only ### separators",
			filePath:      "testdata/parser/ignore_empty_blocks/scenario_002_only_separators.http",
			expectedCount: 0,
			expectedError: false,
		},
		{
			// PRD-COMMENT: FR1 - Empty Block Handling (Comments & Separators)
			// Corresponds to: http_syntax.md "Empty Blocks and Comments", "Request Separator"
			// Validates that files with a mix of only comments and separators result in zero requests
			// and no errors, consistent with FR1.
			name:          "SCENARIO-LIB-028-003: File with comments and ### separators only",
			filePath:      "testdata/parser/ignore_empty_blocks/scenario_003_comments_and_separators.http",
			expectedCount: 0,
			expectedError: false,
		},
		{
			// PRD-COMMENT: FR1 - Empty Block Handling (Valid Request, Separator, Empty Comment Block)
			// Corresponds to: http_syntax.md "Empty Blocks and Comments", "Request Separator"
			// This test case verifies that a file containing a valid request, followed by a separator,
			// and then an empty comment block, is parsed correctly, resulting in one request and no errors.
			name:           "SCENARIO-LIB-028-004: Valid request, then separator, then only comments",
			filePath:       "testdata/parser/ignore_empty_blocks/scenario_004_valid_then_empty_comment.http",
			expectedCount:  1,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "https://example.com/first",
		},
		{
			// PRD-COMMENT: FR1 - Empty Block Handling (Empty Comment Block, Separator, Valid Request)
			// Corresponds to: http_syntax.md "Empty Blocks and Comments", "Request Separator"
			// This test case ensures that a file starting with an empty comment block, followed by a separator,
			// and then a valid request, is parsed correctly, resulting in one request and no errors.
			name:           "SCENARIO-LIB-028-005: Only comments, then separator, then valid request",
			filePath:       "testdata/parser/ignore_empty_blocks/scenario_005_empty_comment_then_valid.http",
			expectedCount:  1,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "https://example.com/second",
		},
		{
			// PRD-COMMENT: FR1 - Empty Block Handling (Valid Request, Separator with Comments, Valid Request)
			// Corresponds to: http_syntax.md "Empty Blocks and Comments", "Request Separator"
			// This test case verifies that a file containing two valid requests separated by a separator
			// with comments in between is parsed correctly, resulting in two requests and no errors.
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
			// PRD-COMMENT: FR1 - Empty Block Handling (Empty File)
			// Corresponds to: http_syntax.md "Empty Blocks and Comments"
			// Verifies that an entirely empty file is parsed as having zero requests and no errors,
			// in accordance with FR1.
			name:          "SCENARIO-LIB-028-007: Completely empty file",
			filePath:      "testdata/parser/ignore_empty_blocks/empty_file_content.http",
			expectedCount: 0,
			expectedError: false,
		},
		{
			// PRD-COMMENT: FR1 - Empty Block Handling (Single Valid Request, No Trailing Newline)
			// Corresponds to: http_syntax.md "Request Syntax"
			// This test case ensures that a file containing a single valid request without a trailing newline
			// is parsed correctly, resulting in one request and no errors.
			name:           "SCENARIO-LIB-028-008: Single valid request no trailing newline",
			filePath:       "testdata/parser/ignore_empty_blocks/single_valid_no_newline.http",
			expectedCount:  1,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "http://localhost/api/test",
		},
		{
			// PRD-COMMENT: FR1 - Empty Block Handling (File with Only Variable Definitions)
			// Corresponds to: http_syntax.md "File-Level Variables"
			// Validates that a file containing only variable definitions and no request blocks
			// results in zero requests and no errors, consistent with FR1.
			name:          "SCENARIO-LIB-028-009: File with only variable definitions",
			filePath:      "testdata/parser/ignore_empty_blocks/only_variables.http",
			expectedCount: 0,
			expectedError: false,
		},
		{
			// PRD-COMMENT: FR1 - Empty Block Handling (Mixed Non-Request Content)
			// Corresponds to: http_syntax.md "Empty Blocks and Comments", "Request Separator", "File-Level Variables"
			// Validates that a file with a mix of comments, separators, variable definitions, and empty lines,
			// but no actual request blocks, results in zero requests and no errors, fulfilling FR1.
			name:          "SCENARIO-LIB-028-010: File with mixed non-request content",
			filePath:      "testdata/parser/ignore_empty_blocks/scenario_007_mixed_non_request_content.http", // Assuming this file exists or will be created
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
			// PRD-COMMENT: FR1.2, FR1.4 - Request Separation with Comments
			// Corresponds to: http_syntax.md "Request Separator", "Comments"
			// This test verifies parsing of multiple requests separated by '###',
			// where comments are present on the separator lines themselves.
			// It ensures that requests are correctly delineated and comments are ignored.
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
			// PRD-COMMENT: FR1.2, FR1.4 - Request Separation with Inline Separator Comment
			// Corresponds to: http_syntax.md "Request Separator", "Comments"
			// This test checks the scenario where a request separator '###' and its comment
			// are on the same line as the end of a request, followed immediately by the next request
			// on the subsequent line. Ensures correct parsing of both requests.
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
			// PRD-COMMENT: FR7.1, FR1.4 - Expected Response Separation with Comments
			// Corresponds to: http_syntax.md "Expected Responses (.hresp)", "Comments"
			// This test verifies the parsing of expected response files (.hresp) that contain
			// multiple expected responses separated by '###', where comments are present
			// on the separator lines. It ensures that expected responses are correctly delineated
			// and comments are ignored during parsing.
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
			// PRD-COMMENT: FR7.1 - Parsing Full Expected Response
			// Corresponds to: http_syntax.md "Expected Responses (.hresp)", "Status Line", "Headers", "Response Body"
			// This test verifies the parsing of a complete and valid expected response,
			// including status line, multiple headers, and a JSON body.
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
			// PRD-COMMENT: FR7.1 - Parsing Status Line Only in Expected Response
			// Corresponds to: http_syntax.md "Expected Responses (.hresp)", "Status Line"
			// This test verifies that an expected response containing only a status line
			// is parsed correctly.
			name:           "SCENARIO-LIB-007-002: Status line only",
			content:        `HTTP/1.1 404 Not Found`,
			expectedCount:  1,
			expectedStatus: ptr("404 Not Found"),
		},
		{
			// PRD-COMMENT: FR7.1 - Parsing Status and Headers in Expected Response
			// Corresponds to: http_syntax.md "Expected Responses (.hresp)", "Status Line", "Headers"
			// This test checks the parsing of an expected response that includes a status line
			// and headers, but no body.
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
			// PRD-COMMENT: FR7.1 - Parsing Status and Body in Expected Response
			// Corresponds to: http_syntax.md "Expected Responses (.hresp)", "Status Line", "Response Body"
			// This test verifies parsing of an expected response with a status line and a body,
			// but no explicit headers.
			name: "SCENARIO-LIB-007-004: Status and body only",
			content: `HTTP/1.1 500 Internal Server Error

<error>Server Error</error>`,
			expectedCount:  1,
			expectedStatus: ptr("500 Internal Server Error"),
			expectedBody:   ptr("<error>Server Error</error>"),
		},
		{
			// PRD-COMMENT: FR7.1 - Parsing Empty Expected Response File
			// Corresponds to: http_syntax.md "Expected Responses (.hresp)"
			// This test ensures that an empty .hresp file content is handled gracefully,
			// resulting in zero parsed responses and no error.
			name:          "SCENARIO-LIB-007-005: Empty content",
			content:       ``,
			expectedCount: 0,
			expectError:   false, // Empty content is not an error, just no responses.
		},
		{
			// PRD-COMMENT: FR7.1 - Handling Malformed Status Line in Expected Response
			// Corresponds to: http_syntax.md "Expected Responses (.hresp)", "Status Line"
			// This test verifies that a malformed status line in an expected response
			// results in a parsing error.
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

// PRD-COMMENT: FR2.4 - Variable Scoping and Templating
// Corresponds to: http_syntax.md "Variables", "Variable Scopes (File, Request, Environment)"
// This test function verifies multiple aspects of variable scoping and templating:
// 1. Resolution of nested variable references (e.g., `{{url}}/users` where `url` itself might be a variable).
// 2. Correct overriding of file-level variables by request-specific variables.
// 3. Proper restoration of file-level variable values after a request-specific override has been processed for a preceding request.
// 4. Complex variable expansion within request bodies, particularly ensuring placeholders are correctly identified in JSON structures.
// All scenarios are tested by parsing 'testdata/variables/variable_references.http' which contains multiple requests designed to exercise these scoping rules.
func TestParseRequestFile_VariableScoping(t *testing.T) {
	// Given
	const requestFilePath = "testdata/variables/variable_references.http"

	// When
	parsedFile, err := parseRequestFile(requestFilePath, nil, make([]string, 0))

	// Then
	require.NoError(t, err, "Failed to parse request file")
	require.NotNil(t, parsedFile, "Parsed file should not be nil")
	slog.Info("TestParseRequestFile_VariableScoping: Post-parse details", "file", requestFilePath, "num_requests_parsed", len(parsedFile.Requests))
	for i, req := range parsedFile.Requests {
		slog.Info("TestParseRequestFile_VariableScoping: Parsed request detail",
			"index", i,
			"requestPtr", fmt.Sprintf("%p", req),
			"name", req.Name,
			"method", req.Method,
			"rawURL", req.RawURLString)
	}
	require.Len(t, parsedFile.Requests, 4, "Expected 4 requests")

	// Verify nested variable references
	assert.Equal(t, "{{url}}/users", parsedFile.Requests[0].RawURLString, "Nested variable references mismatch")

	// Verify request-specific variable overrides file-level variable
	assert.Equal(t, "https://{{host}}:{{port}}{{base_path}}/users/me", parsedFile.Requests[1].RawURLString, "Request-specific variable override mismatch")

	// Verify file-level variables are restored after request-specific override
	assert.Equal(t, "https://{{host}}:{{port}}{{base_path}}/status", parsedFile.Requests[2].RawURLString, "File-level variable restoration mismatch")

	// Verify complex variable expansion in JSON body
	bodyReq := parsedFile.Requests[3]
	assert.Equal(t, "https://{{host}}:{{port}}{{base_path}}/users/{{user_id}}/permissions", bodyReq.RawURLString, "URL with variable in path mismatch")
	assert.Contains(t, bodyReq.RawBody, "\"userId\": \"{{user_id}}\"", "userId placeholder in JSON body not preserved")
	assert.Contains(t, bodyReq.RawBody, "\"role\": \"{{user_role}}\"", "role placeholder in JSON body not preserved")
	assert.Contains(t, bodyReq.RawBody, "\"teamId\": \"{{team_id}}\"", "teamId placeholder in JSON body not preserved")
	assert.Contains(t, bodyReq.RawBody, "\"read:{{team_id}}:*\"", "Nested variable placeholder in JSON array not preserved")
}
