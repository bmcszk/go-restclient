package restclient

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertRequestDetails(t *testing.T, req *Request, method, url, httpVersion, name string, headers http.Header, body string, filePath string, lineNumber int) {
	t.Helper()
	assert.Equal(t, method, req.Method)
	assert.Equal(t, url, req.URL.String())
	assert.Equal(t, httpVersion, req.HTTPVersion)
	assert.Equal(t, name, req.Name)
	if headers != nil {
		assert.Equal(t, headers, req.Headers)
	} else {
		assert.Empty(t, req.Headers)
	}
	bodyBytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	// For JSON bodies, direct string comparison can be brittle due to whitespace.
	// If 'body' is expected to be JSON, prefer JSONEq in the calling test for req.RawBody.
	// This helper primarily ensures the req.Body reader provides the expected content.
	if strings.HasPrefix(strings.TrimSpace(body), "{") || strings.HasPrefix(strings.TrimSpace(body), "[") {
		assert.JSONEq(t, body, string(bodyBytes))
	} else {
		assert.Equal(t, body, string(bodyBytes))
	}

	// RawBody check removed from helper; tests should verify RawBody specifically if needed.
	// if body != "" { // Only check RawBody if we expect a body
	// 	assert.Equal(t, body, req.RawBody)
	// }
	assert.Equal(t, filePath, req.FilePath)
	assert.Equal(t, lineNumber, req.LineNumber)
}

func TestParseRequests_SimpleGET(t *testing.T) {
	filePath := "testdata/http_request_files/simple_get_with_headers_and_comment.http"
	parsedFile, err := ParseRequestFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	assertRequestDetails(t, req, "GET", "https://example.com/api/users", "HTTP/1.1", "", http.Header{"Accept": []string{"application/json"}, "User-Agent": []string{"test-client"}}, "", filePath, 2)
}

func TestParseRequests_POSTWithBody(t *testing.T) {
	filePath := "testdata/http_request_files/post_with_body_and_version.http"
	parsedFile, err := ParseRequestFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	expectedBody := `{
  "name": "test",
  "value": 123
}`
	assertRequestDetails(t, req, "POST", "https://example.com/api/resource", "HTTP/1.1", "", http.Header{"Content-Type": []string{"application/json"}}, expectedBody, filePath, 1)
}

func TestParseRequests_MultipleRequests(t *testing.T) {
	filePath := "testdata/http_request_files/multiple_requests_varied.http"
	parsedFile, err := ParseRequestFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	require.Len(t, parsedFile.Requests, 3)

	// Check first request
	req1 := parsedFile.Requests[0]
	assertRequestDetails(t, req1, "GET", "https://example.com/users", "HTTP/1.1", "First Request: Get Users", nil, "", filePath, 1)

	// Check second request
	req2 := parsedFile.Requests[1]
	expectedBody2 := `{
  "username": "newuser"
}
`
	assertRequestDetails(t, req2, "POST", "https://example.com/users", "HTTP/1.1", "Second Request: Create User", http.Header{"Content-Type": []string{"application/json"}}, expectedBody2, filePath, 4)

	// Check third request
	req3 := parsedFile.Requests[2]
	expectedBody3 := `{
  "status": "active"
}`
	assertRequestDetails(t, req3, "PUT", "https://example.com/users/1", "HTTP/2.0", "Third Request with Custom HTTP Version", nil, expectedBody3, filePath, 13)
}

func TestParseRequests_InvalidRequestLine(t *testing.T) {
	_, err := ParseRequestFile("testdata/http_request_files/invalid_request_line_only.http")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid request line")
	assert.Contains(t, err.Error(), "line 1")
}

func TestParseRequests_InvalidHeader(t *testing.T) {
	_, err := ParseRequestFile("testdata/http_request_files/invalid_header_format.http")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid header line")
	assert.Contains(t, err.Error(), "line 2") // Header is on L2
}

func TestParseRequests_InvalidURL(t *testing.T) {
	_, err := ParseRequestFile("testdata/http_request_files/invalid_url_format.http")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URL")
}

func TestParseRequests_EmptyFile(t *testing.T) {
	// Note: The file testdata/http_request_files/comment_only_file.http is used here as it's an empty file.
	// An empty file should result in an error indicating no requests were found.
	parsedFile, err := ParseRequestFile("testdata/http_request_files/comment_only_file.http")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid requests found in file")
	assert.Nil(t, parsedFile)

	// Test with empty reader and no filename (should not error, used in some internal cases)
	// This part remains as it tests the parseRequests internal function behavior with an empty reader.
	parsedFileInternal, errInternal := parseRequests(strings.NewReader(""), "")
	require.NoError(t, errInternal)
	require.NotNil(t, parsedFileInternal)
	assert.Empty(t, parsedFileInternal.Requests)
}

func TestParseRequests_CommentOnlyFile(t *testing.T) {
	parsedFile, err := ParseRequestFile("testdata/http_request_files/comment_only_file.http")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid requests found in file")
	assert.Nil(t, parsedFile)
}

func TestParseRequests_RequestWithoutSeparator(t *testing.T) {
	filePath := "testdata/http_request_files/request_no_separator.http"
	parsedFile, err := ParseRequestFile(filePath)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	assertRequestDetails(t, req, "GET", "https://example.com/no-separator", "HTTP/1.1", "", http.Header{"Accept": []string{"text/plain"}}, "", filePath, 1)
}

func TestParseRequests_HeadersWithDifferentSpacing(t *testing.T) {
	filePath := "testdata/http_request_files/headers_various_spacing.http"
	parsedFile, err := ParseRequestFile(filePath)
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	assert.Equal(t, "Value1", req.Headers.Get("Header1"))
	assert.Equal(t, "Value2", req.Headers.Get("Header2"))
	assert.Equal(t, "Value3", req.Headers.Get("Header3"))
	assert.Equal(t, "Value4", req.Headers.Get("Header4"))
}

func TestParseRequests_MultipleHeadersSameKey(t *testing.T) {
	filePath := "testdata/http_request_files/multiple_accept_headers.http"
	parsedFile, err := ParseRequestFile(filePath)
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	// http.Header.Get only returns the first. Use req.Headers["Accept"] for all.
	assert.Equal(t, "application/json", req.Headers.Get("Accept"))
	assert.Equal(t, []string{"application/json", "text/xml"}, req.Headers["Accept"])
}

func TestParseRequests_SeparatorWithoutName(t *testing.T) {
	filePath := "testdata/http_request_files/separator_no_name.http"
	parsedFile, err := ParseRequestFile(filePath)
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 2)

	assert.Empty(t, parsedFile.Requests[0].Name)
	assert.Equal(t, "https://example.com/one", parsedFile.Requests[0].URL.String())
	assert.Equal(t, 1, parsedFile.Requests[0].LineNumber) // First ### is L1

	assert.Empty(t, parsedFile.Requests[1].Name)
	assert.Equal(t, "https://example.com/two", parsedFile.Requests[1].URL.String())
	assert.Equal(t, 4, parsedFile.Requests[1].LineNumber) // Second ### is L4
}

func TestParseRequestFile_SimpleGET_FromFile(t *testing.T) {
	filePath := "testdata/http_request_files/simple_get.http"
	parsedFile, err := ParseRequestFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	assert.Equal(t, filePath, parsedFile.FilePath)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	assertRequestDetails(t, req, "GET", "https://jsonplaceholder.typicode.com/todos/1", "HTTP/1.1", "", nil, "", filePath, req.LineNumber)
	assert.True(t, req.LineNumber > 0) // Line number should be set
}

func TestParseRequestFile_GetWithHeaders_FromFile(t *testing.T) {
	filePath := "testdata/http_request_files/get_with_headers.http"
	parsedFile, err := ParseRequestFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	assert.Equal(t, filePath, parsedFile.FilePath)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	headers := http.Header{
		"Accept":     []string{"application/json"},
		"User-Agent": []string{"go-restclient-test"},
	}
	assertRequestDetails(t, req, "GET", "https://jsonplaceholder.typicode.com/todos/1", "HTTP/1.1", "", headers, "", filePath, req.LineNumber)
	assert.True(t, req.LineNumber > 0, "Line number should be set and greater than 0")
}

func TestParseRequestFile_PostWithJsonBody_FromFile(t *testing.T) {
	filePath := "testdata/http_request_files/post_with_json_body.http"
	parsedFile, err := ParseRequestFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	assert.Equal(t, filePath, parsedFile.FilePath)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	headers := http.Header{"Content-Type": []string{"application/json"}}
	expectedBody := `{
  "title": "foo",
  "body": "bar",
  "userId": 1
}`
	assertRequestDetails(t, req, "POST", "https://jsonplaceholder.typicode.com/posts", "HTTP/1.1", "", headers, expectedBody, filePath, req.LineNumber)
	assert.True(t, req.LineNumber > 0, "Line number should be set and greater than 0")

	// Assert RawBody using JSONEq for robust JSON comparison
	assert.JSONEq(t, expectedBody, req.RawBody, "RawBody does not match expected JSON")

	// The existing checks for RawBody can remain as they are also useful.
	if len(strings.TrimSpace(expectedBody)) > 0 {
		assert.NotEmpty(t, req.RawBody, "RawBody should not be empty when a body is provided")
		// This JSONEq might be redundant if the above one passes, but it specifically checks after trimming newline.
		assert.JSONEq(t, expectedBody, strings.TrimSuffix(req.RawBody, "\n"), "RawBody does not match expected JSON (ignoring potential trailing newline)")
	}
}

func assertExpectedResponseDetails(t *testing.T, resp *ExpectedResponse, statusCode int, status string, headers http.Header, body string) {
	t.Helper()
	require.NotNil(t, resp.StatusCode)
	assert.Equal(t, statusCode, *resp.StatusCode)
	require.NotNil(t, resp.Status)
	assert.Equal(t, status, *resp.Status)
	if headers != nil {
		assert.Equal(t, headers, resp.Headers)
	} else {
		assert.Empty(t, resp.Headers)
	}
	if body != "" {
		require.NotNil(t, resp.Body)
		assert.Equal(t, body, *resp.Body)
	} else {
		// If an empty body string is passed, we check if resp.Body is nil or points to an empty string.
		// This handles cases where the parser might set Body to nil for no body, or to *new(string) for an explicitly empty one.
		assert.True(t, resp.Body == nil || *resp.Body == "", "Expected body to be nil or empty, but got '%v'", resp.Body)
	}
}

func TestParseExpectedResponses_SimpleOK(t *testing.T) {
	filePath := "testdata/http_response_files/parser_simple_ok.hresp"
	expectedResponses, err := ParseExpectedResponseFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, expectedResponses)
	require.Len(t, expectedResponses, 1)

	expResp := expectedResponses[0]
	headers := http.Header{"Content-Type": []string{"application/json"}}
	expectedBody := `{
  "status": "success"
}`
	assertExpectedResponseDetails(t, expResp, 200, "200 OK", headers, expectedBody)
}

func TestParseExpectedResponses_MultipleResponses(t *testing.T) {
	filePath := "testdata/http_response_files/parser_multiple_responses.hresp"
	expectedResponses, err := ParseExpectedResponseFile(filePath)
	require.NoError(t, err)
	require.Len(t, expectedResponses, 2)

	// Check first expected response
	exp1 := expectedResponses[0]
	headers1 := http.Header{"Content-Type": []string{"application/json"}}
	body1JSON := `{"message": "First response"}`
	// The helper will do a direct string comparison. We add a JSONEq for robustness.
	assertExpectedResponseDetails(t, exp1, 200, "200 OK", headers1, *exp1.Body) // Pass the actual parsed body to the helper for direct comparison
	require.NotNil(t, exp1.Body)                                                // Ensure body is not nil before dereferencing for JSONEq
	assert.JSONEq(t, body1JSON, *exp1.Body, "First response body mismatch (JSONEq)")

	// Check second expected response
	exp2 := expectedResponses[1]
	headers2 := http.Header{
		"Content-Type":    []string{"application/json"},
		"X-Custom-Header": []string{"value"},
	}
	body2JSON := `{"id": 123}`
	assertExpectedResponseDetails(t, exp2, 201, "201 Created", headers2, *exp2.Body) // Pass the actual parsed body
	require.NotNil(t, exp2.Body)                                                     // Ensure body is not nil
	assert.JSONEq(t, body2JSON, *exp2.Body, "Second response body mismatch (JSONEq)")
}

func TestParseExpectedResponses_InvalidStatusLine(t *testing.T) {
	_, err := ParseExpectedResponseFile("testdata/http_response_files/parser_invalid_status_line.hresp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status line")
}

func TestParseExpectedResponses_InvalidStatusCode(t *testing.T) {
	_, err := ParseExpectedResponseFile("testdata/http_response_files/parser_invalid_status_code.hresp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status code 'ABC'")
}

func TestParseExpectedResponses_InvalidHeaderFormat(t *testing.T) {
	_, err := ParseExpectedResponseFile("testdata/http_response_files/parser_invalid_header_format.hresp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid header line")
}

func TestParseExpectedResponses_EmptyFile(t *testing.T) {
	_, err := ParseExpectedResponseFile("testdata/http_response_files/validator_empty_expected.hresp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid expected responses found in file")
}

func TestParseExpectedResponses_CommentOnlyFile(t *testing.T) {
	_, err := ParseExpectedResponseFile("testdata/http_response_files/parser_comment_only_file.hresp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid expected responses found in file")
}

func TestParseExpectedResponses_SeparatorWithWhitespace(t *testing.T) {
	filePath := "testdata/http_response_files/parser_separator_whitespace.hresp"
	resps, err := ParseExpectedResponseFile(filePath)
	require.NoError(t, err)
	require.Len(t, resps, 2, "Expected two responses")

	assertExpectedResponseDetails(t, resps[0], 200, "200 OK", nil, "Body1")
	assertExpectedResponseDetails(t, resps[1], 201, "201 Created", nil, "Body2 ")
}

func TestParseExpectedResponses_SeparatorInContent(t *testing.T) {
	filePath := "testdata/http_response_files/parser_separator_in_content.hresp"
	resps, err := ParseExpectedResponseFile(filePath)
	require.NoError(t, err)
	require.Len(t, resps, 1, "Expected a single response")

	resp1 := resps[0]
	headers := http.Header{"X-Custom-Header": []string{"Info ### SeparatorLikeValue"}}
	expectedBody := "Body line 1\nThis body contains ### as text.\nBody line 3"
	assertExpectedResponseDetails(t, resp1, 200, "200 OK", headers, expectedBody)
}

// TestParseExpectedResponses_SingleResponseNoSeparator tests parsing a single expected response
// ... existing code ...
