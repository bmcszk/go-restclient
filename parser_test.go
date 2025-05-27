package restclient

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRequests_SimpleGET(t *testing.T) {
	filePath := "testdata/http_request_files/simple_get_with_headers_and_comment.http"
	parsedFile, err := ParseRequestFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "https://example.com/api/users", req.URL.String())
	assert.Equal(t, "application/json", req.Headers.Get("Accept"))
	assert.Equal(t, "test-client", req.Headers.Get("User-Agent"))
	assert.Equal(t, "HTTP/1.1", req.HTTPVersion)
	assert.Empty(t, req.Name)
	bodyBytes, _ := io.ReadAll(req.Body)
	assert.Empty(t, string(bodyBytes))
	assert.Equal(t, filePath, req.FilePath)
	assert.Equal(t, 2, req.LineNumber) // Line number where GET is (after comment)
}

func TestParseRequests_POSTWithBody(t *testing.T) {
	filePath := "testdata/http_request_files/post_with_body_and_version.http"
	parsedFile, err := ParseRequestFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "https://example.com/api/resource", req.URL.String())
	assert.Equal(t, "application/json", req.Headers.Get("Content-Type"))
	assert.Equal(t, "HTTP/1.1", req.HTTPVersion)

	expectedBody := `{
  "name": "test",
  "value": 123
}`
	bodyBytes, _ := io.ReadAll(req.Body)
	assert.Equal(t, expectedBody, string(bodyBytes))
	assert.Equal(t, expectedBody, req.RawBody)
	assert.Equal(t, 1, req.LineNumber) // POST line is L1
}

func TestParseRequests_MultipleRequests(t *testing.T) {
	filePath := "testdata/http_request_files/multiple_requests_varied.http"
	parsedFile, err := ParseRequestFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	require.Len(t, parsedFile.Requests, 3)

	// Check first request
	req1 := parsedFile.Requests[0]
	assert.Equal(t, "First Request: Get Users", req1.Name)
	assert.Equal(t, "GET", req1.Method)
	assert.Equal(t, "https://example.com/users", req1.URL.String())
	assert.Equal(t, "HTTP/1.1", req1.HTTPVersion)
	bodyBytes1, _ := io.ReadAll(req1.Body)
	assert.Empty(t, string(bodyBytes1))
	assert.Equal(t, 1, req1.LineNumber) // ### First Request is L1

	// Check second request
	req2 := parsedFile.Requests[1]
	assert.Equal(t, "Second Request: Create User", req2.Name)
	assert.Equal(t, "POST", req2.Method)
	assert.Equal(t, "https://example.com/users", req2.URL.String())
	assert.Equal(t, "application/json", req2.Headers.Get("Content-Type"))
	assert.Equal(t, "HTTP/1.1", req2.HTTPVersion)
	expectedBody2 := `{
  "username": "newuser"
}
`
	bodyBytes2, _ := io.ReadAll(req2.Body)
	assert.Equal(t, expectedBody2, string(bodyBytes2))
	assert.Equal(t, 4, req2.LineNumber) // ### Second Request is L4

	// Check third request
	req3 := parsedFile.Requests[2]
	assert.Equal(t, "Third Request with Custom HTTP Version", req3.Name)
	assert.Equal(t, "PUT", req3.Method)
	assert.Equal(t, "https://example.com/users/1", req3.URL.String())
	assert.Equal(t, "HTTP/2.0", req3.HTTPVersion)
	expectedBody3 := `{
  "status": "active"
}`
	bodyBytes3, _ := io.ReadAll(req3.Body)
	assert.Equal(t, expectedBody3, string(bodyBytes3))
	assert.Equal(t, 13, req3.LineNumber) // ### Third Request is L13
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
	// Note: The file testdata/http_request_files/no_requests.http is used here as it's an empty file.
	// It is also used by TestExecuteFile_NoRequestsInFile in client_test.go
	parsedFile, err := ParseRequestFile("testdata/http_request_files/no_requests.http")
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
	assert.Empty(t, req.Name) // No name if no separator
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "https://example.com/no-separator", req.URL.String())
	assert.Equal(t, "text/plain", req.Headers.Get("Accept"))
	assert.Equal(t, 1, req.LineNumber) // GET is L1
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
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "https://jsonplaceholder.typicode.com/todos/1", req.URL.String())
	assert.Empty(t, req.Headers)
	assert.Equal(t, "HTTP/1.1", req.HTTPVersion) // Assuming default or parser sets it
	assert.Empty(t, req.Name)
	bodyBytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Empty(t, string(bodyBytes))
	assert.Equal(t, filePath, req.FilePath)
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
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "https://jsonplaceholder.typicode.com/todos/1", req.URL.String())
	assert.Equal(t, "HTTP/1.1", req.HTTPVersion)
	assert.Empty(t, req.Name)

	require.NotNil(t, req.Headers)
	assert.Equal(t, "application/json", req.Headers.Get("Accept"))
	assert.Equal(t, "go-restclient-test", req.Headers.Get("User-Agent"))

	bodyBytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Empty(t, string(bodyBytes))
	assert.Equal(t, filePath, req.FilePath)
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
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "https://jsonplaceholder.typicode.com/posts", req.URL.String())
	assert.Equal(t, "HTTP/1.1", req.HTTPVersion) // Assuming default
	assert.Empty(t, req.Name)

	require.NotNil(t, req.Headers)
	assert.Equal(t, "application/json", req.Headers.Get("Content-Type"))

	expectedBody := `{
  "title": "foo",
  "body": "bar",
  "userId": 1
}`
	bodyBytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.JSONEq(t, expectedBody, string(bodyBytes), "Parsed body does not match expected JSON")
	assert.Equal(t, filePath, req.FilePath)
	assert.True(t, req.LineNumber > 0, "Line number should be set and greater than 0")

	// Also check RawBody if it's populated and matches (stripping potential trailing newline from file)
	// The parser might or might not include the final newline of the file in RawBody.
	// For this test, we are primarily concerned with the io.Reader Body's content.
	// However, if RawBody is a feature, it's good to be aware.
	// For now, let's ensure it's not empty if a body was provided.
	if len(strings.TrimSpace(expectedBody)) > 0 {
		assert.NotEmpty(t, req.RawBody, "RawBody should not be empty when a body is provided")
		// A more specific assertion for RawBody might be needed if its exact content
		// (including trailing newlines) is strictly defined by the parser.
		// For instance, if the parser is expected to store exactly what's in the file:
		assert.JSONEq(t, expectedBody, strings.TrimSuffix(req.RawBody, "\n"), "RawBody does not match expected JSON (ignoring potential trailing newline)")
	}
}

func TestParseExpectedResponses_SimpleOK(t *testing.T) {
	filePath := "testdata/http_response_files/parser_simple_ok.hresp"
	expectedResponses, err := ParseExpectedResponseFile(filePath)

	require.NoError(t, err)
	require.NotNil(t, expectedResponses)
	require.Len(t, expectedResponses, 1)

	expResp := expectedResponses[0]
	require.NotNil(t, expResp.StatusCode)
	assert.Equal(t, 200, *expResp.StatusCode)
	require.NotNil(t, expResp.Status)
	assert.Equal(t, "200 OK", *expResp.Status)
	assert.Equal(t, "application/json", expResp.Headers.Get("Content-Type"))
	require.NotNil(t, expResp.Body)
	expectedBody := `{
  "status": "success"
}`
	assert.Equal(t, expectedBody, *expResp.Body)
}

func TestParseExpectedResponses_MultipleResponses(t *testing.T) {
	filePath := "testdata/http_response_files/parser_multiple_responses.hresp"
	expectedResponses, err := ParseExpectedResponseFile(filePath)
	require.NoError(t, err)
	require.Len(t, expectedResponses, 2)

	// Check first expected response
	exp1 := expectedResponses[0]
	require.NotNil(t, exp1.StatusCode)
	assert.Equal(t, 200, *exp1.StatusCode)
	require.NotNil(t, exp1.Status)
	assert.Equal(t, "200 OK", *exp1.Status)
	assert.Equal(t, "application/json", exp1.Headers.Get("Content-Type"))
	require.NotNil(t, exp1.Body)
	assert.JSONEq(t, `{"message": "First response"}`, *exp1.Body)

	// Check second expected response
	exp2 := expectedResponses[1]
	require.NotNil(t, exp2.StatusCode)
	assert.Equal(t, 201, *exp2.StatusCode)
	require.NotNil(t, exp2.Status)
	assert.Equal(t, "201 Created", *exp2.Status)
	assert.Equal(t, "application/json", exp2.Headers.Get("Content-Type"))
	assert.Equal(t, "value", exp2.Headers.Get("X-Custom-Header"))
	require.NotNil(t, exp2.Body)
	assert.JSONEq(t, `{"id": 123}`, *exp2.Body)
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
	_, err := ParseExpectedResponseFile("testdata/http_response_files/parser_empty_file.hresp")
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

	assert.Equal(t, 200, *resps[0].StatusCode)
	// With simplified file content, no blank line after Body1, so no trailing \n
	assert.Equal(t, "Body1", *resps[0].Body)

	assert.Equal(t, 201, *resps[1].StatusCode)
	assert.Equal(t, "Body2 ", *resps[1].Body)
}

func TestParseExpectedResponses_SeparatorInContent(t *testing.T) {
	filePath := "testdata/http_response_files/parser_separator_in_content.hresp"
	resps, err := ParseExpectedResponseFile(filePath)
	require.NoError(t, err)
	require.Len(t, resps, 1, "Expected a single response")

	resp1 := resps[0]
	assert.Equal(t, 200, *resp1.StatusCode)
	assert.Equal(t, "Info ### SeparatorLikeValue", resp1.Headers.Get("X-Custom-Header"))
	expectedBody := "Body line 1\nThis body contains ### as text.\nBody line 3"
	assert.Equal(t, expectedBody, *resp1.Body)
}

// TestParseExpectedResponses_SingleResponseNoSeparator tests parsing a single expected response
// ... existing code ...
