//go:build unit

package restclient

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRequests_SimpleGET(t *testing.T) {
	content := `
# This is a comment
GET https://example.com/api/users
Accept: application/json
User-Agent: test-client

`
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_simple_get.rest")

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
	assert.Equal(t, "test_simple_get.rest", req.FilePath)
	assert.Equal(t, 3, req.LineNumber) // Line number where GET is (after initial newline and comment)
}

func TestParseRequests_POSTWithBody(t *testing.T) {
	content := `
POST https://example.com/api/resource HTTP/1.1
Content-Type: application/json

{
  "name": "test",
  "value": 123
}
`
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_post_body.rest")

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
	assert.Equal(t, 2, req.LineNumber) // POST line is L2 after initial newline
}

func TestParseRequests_MultipleRequests(t *testing.T) {
	content := `
### First Request: Get Users
GET https://example.com/users

### Second Request: Create User
# A comment for the second request
POST https://example.com/users
Content-Type: application/json

{
  "username": "newuser"
}

### Third Request with Custom HTTP Version
PUT https://example.com/users/1 HTTP/2.0

{
  "status": "active"
}
`
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_multi.rest")

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
	assert.Equal(t, 2, req1.LineNumber) // ### First Request is L2

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
	assert.Equal(t, 5, req2.LineNumber) // ### Second Request is L5

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
	assert.Equal(t, 14, req3.LineNumber) // ### Third Request is L14
}

func TestParseRequests_InvalidRequestLine(t *testing.T) {
	content := `GET`
	reader := strings.NewReader(content)
	_, err := parseRequests(reader, "test_invalid.rest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid request line")
	assert.Contains(t, err.Error(), "line 1")
}

func TestParseRequests_InvalidHeader(t *testing.T) {
	content := `GET https://example.com
Accept application/json`
	reader := strings.NewReader(content)
	_, err := parseRequests(reader, "test_invalid_header.rest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid header line")
	assert.Contains(t, err.Error(), "line 2") // Header is on L2
}

func TestParseRequests_InvalidURL(t *testing.T) {
	content := `GET ://invalid-url`
	reader := strings.NewReader(content)
	_, err := parseRequests(reader, "test_invalid_url.rest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URL")
}

func TestParseRequests_EmptyFile(t *testing.T) {
	content := ``
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_empty.rest") // provide filename to trigger error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid requests found in file")
	assert.Nil(t, parsedFile)

	// Test with empty reader and no filename (should not error, used in some internal cases)
	parsedFile, err = parseRequests(strings.NewReader(""), "")
	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	assert.Empty(t, parsedFile.Requests)
}

func TestParseRequests_CommentOnlyFile(t *testing.T) {
	content := `# comment1
# comment2`
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_comment_only.rest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid requests found in file")
	assert.Nil(t, parsedFile)
}

func TestParseRequests_RequestWithoutSeparator(t *testing.T) {
	content := `
GET https://example.com/no-separator
Accept: text/plain
`
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_no_sep.rest")
	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	assert.Empty(t, req.Name) // No name if no separator
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "https://example.com/no-separator", req.URL.String())
	assert.Equal(t, "text/plain", req.Headers.Get("Accept"))
	assert.Equal(t, 2, req.LineNumber) // GET is L2 after initial newline
}

func TestParseRequests_HeadersWithDifferentSpacing(t *testing.T) {
	content := `
GET https://example.com
Header1:Value1
Header2: Value2
Header3 : Value3
Header4 :Value4
`
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_header_spacing.rest")
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	assert.Equal(t, "Value1", req.Headers.Get("Header1"))
	assert.Equal(t, "Value2", req.Headers.Get("Header2"))
	assert.Equal(t, "Value3", req.Headers.Get("Header3"))
	assert.Equal(t, "Value4", req.Headers.Get("Header4"))
}

func TestParseRequests_BodyEndsWithNewline(t *testing.T) {
	content := `
POST https://example.com/submit
Content-Type: text/plain

This is the body.

`
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_body_newline.rest")
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	expectedBody := "This is the body.\n"
	bodyBytes, _ := io.ReadAll(req.Body)
	assert.Equal(t, expectedBody, string(bodyBytes))
	assert.Equal(t, expectedBody, req.RawBody)
}

func TestParseRequests_BodyWithoutTrailingNewline(t *testing.T) {
	content := `
POST https://example.com/submit
Content-Type: text/plain

This is the body.`
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_body_no_newline.rest")
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	expectedBody := "This is the body."
	bodyBytes, _ := io.ReadAll(req.Body)
	assert.Equal(t, expectedBody, string(bodyBytes))
	assert.Equal(t, expectedBody, req.RawBody)
}

func TestParseRequests_MultipleHeadersSameKey(t *testing.T) {
	content := `
GET https://example.com
Accept: application/json
Accept: text/xml
`
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_multi_header.rest")
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 1)

	req := parsedFile.Requests[0]
	// http.Header.Get only returns the first. Use req.Headers["Accept"] for all.
	assert.Equal(t, "application/json", req.Headers.Get("Accept"))
	assert.Equal(t, []string{"application/json", "text/xml"}, req.Headers["Accept"])
}

func TestParseRequests_SeparatorWithoutName(t *testing.T) {
	content := `
###
GET https://example.com/one

### 
POST https://example.com/two

`
	reader := strings.NewReader(content)
	parsedFile, err := parseRequests(reader, "test_sep_no_name.rest")
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 2)

	assert.Empty(t, parsedFile.Requests[0].Name)
	assert.Equal(t, "https://example.com/one", parsedFile.Requests[0].URL.String())
	assert.Equal(t, 2, parsedFile.Requests[0].LineNumber) // First ### is L2

	assert.Empty(t, parsedFile.Requests[1].Name)
	assert.Equal(t, "https://example.com/two", parsedFile.Requests[1].URL.String())
	assert.Equal(t, 5, parsedFile.Requests[1].LineNumber) // Second ### is L5
}
