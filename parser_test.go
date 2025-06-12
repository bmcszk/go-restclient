package restclient

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper to create a mock client for parser testing
func newTestClient() *Client {
	client, _ := NewClient()
	return client
}

// Test helper to parse HTTP content from string
func parseHTTPContent(t *testing.T, content string) (*ParsedFile, error) {
	t.Helper()
	
	reader := bufio.NewReader(strings.NewReader(content))
	client := newTestClient()
	
	requestScopedSystemVars := make(map[string]string)
	osEnvGetter := func(key string) (string, bool) { return "", false }
	dotEnvVars := make(map[string]string)
	importStack := make([]string, 0)
	
	return parseRequests(reader, "test.http", client, requestScopedSystemVars, osEnvGetter, dotEnvVars, importStack)
}

// Test helper to parse HTTP content and expect success
func mustParseHTTP(t *testing.T, content string) *ParsedFile {
	t.Helper()
	
	parsed, err := parseHTTPContent(t, content)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	
	return parsed
}

// Test helper to assert request count
func assertRequestCount(t *testing.T, parsed *ParsedFile, expectedCount int) {
	t.Helper()
	assert.Len(t, parsed.Requests, expectedCount, "unexpected number of requests parsed")
}

// Test helper to get a specific request by index
func getRequest(t *testing.T, parsed *ParsedFile, index int) *Request {
	t.Helper()
	require.Less(t, index, len(parsed.Requests), "request index out of bounds")
	return parsed.Requests[index]
}

// TestParseRequestStructureBasics tests basic request structure parsing (T22)
func TestParseRequestStructureBasics(t *testing.T) {
	t.Run("single_GET_request", func(t *testing.T) {
		content := `GET https://api.example.com/users`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 1)
		
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, "https://api.example.com/users", req.URL.String())
	})
	
	t.Run("multiple_requests_separated_by_triple_hash", func(t *testing.T) {
		content := `GET https://api.example.com/users

###

POST https://api.example.com/users
Content-Type: application/json

{
  "name": "John Doe"
}`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 2)
		
		req1 := getRequest(t, parsed, 0)
		assert.Equal(t, "GET", req1.Method)
		assert.Equal(t, "https://api.example.com/users", req1.URL.String())
		
		req2 := getRequest(t, parsed, 1)
		assert.Equal(t, "POST", req2.Method)
		assert.Equal(t, "https://api.example.com/users", req2.URL.String())
		assert.Equal(t, "application/json", req2.Headers.Get("Content-Type"))
		assert.Contains(t, req2.RawBody, "John Doe")
	})
	
	t.Run("file_extensions_http_and_rest", func(t *testing.T) {
		// This test verifies that the parser doesn't care about file extension
		// when parsing content (file extension handling is in the client)
		content := `GET https://api.example.com/test`
		
		reader := bufio.NewReader(strings.NewReader(content))
		client := newTestClient()
		
		// Test with .http extension
		parsed1, err1 := parseRequests(reader, "test.http", client, make(map[string]string), 
			func(string) (string, bool) { return "", false }, make(map[string]string), []string{})
		require.NoError(t, err1)
		assertRequestCount(t, parsed1, 1)
		
		// Test with .rest extension
		reader = bufio.NewReader(strings.NewReader(content))
		parsed2, err2 := parseRequests(reader, "test.rest", client, make(map[string]string), 
			func(string) (string, bool) { return "", false }, make(map[string]string), []string{})
		require.NoError(t, err2)
		assertRequestCount(t, parsed2, 1)
		
		// Both should parse identically
		assert.Equal(t, parsed1.Requests[0].Method, parsed2.Requests[0].Method)
		assert.Equal(t, parsed1.Requests[0].URL.String(), parsed2.Requests[0].URL.String())
	})
	
	t.Run("empty_file", func(t *testing.T) {
		content := ``
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 0)
	})
	
	t.Run("file_with_only_comments", func(t *testing.T) {
		content := `# This is a comment
// This is another comment
# Another hash comment`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 0)
	})
	
	t.Run("file_with_only_separators", func(t *testing.T) {
		content := `###
###
###`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 0)
	})
}

// TestParseRequestNaming tests request naming functionality (T23)
func TestParseRequestNaming(t *testing.T) {
	t.Run("request_naming_with_triple_hash_separator", func(t *testing.T) {
		content := `### Get Users
GET https://api.example.com/users

### Create User
POST https://api.example.com/users
Content-Type: application/json

{"name": "test"}`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 2)
		
		req1 := getRequest(t, parsed, 0)
		assert.Equal(t, "Get Users", req1.Name)
		assert.Equal(t, "GET", req1.Method)
		
		req2 := getRequest(t, parsed, 1)
		assert.Equal(t, "Create User", req2.Name)
		assert.Equal(t, "POST", req2.Method)
	})
	
	t.Run("request_naming_with_at_name_directive", func(t *testing.T) {
		content := `# @name getUserRequest
GET https://api.example.com/users

###

// @name createUserRequest  
POST https://api.example.com/users
Content-Type: application/json

{"name": "test"}`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 2)
		
		req1 := getRequest(t, parsed, 0)
		assert.Equal(t, "getUserRequest", req1.Name)
		assert.Equal(t, "GET", req1.Method)
		
		req2 := getRequest(t, parsed, 1)
		assert.Equal(t, "createUserRequest", req2.Name)
		assert.Equal(t, "POST", req2.Method)
	})
	
	t.Run("at_name_overrides_separator_name", func(t *testing.T) {
		content := `### Default Name
# @name OverrideName
GET https://api.example.com/users`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 1)
		
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "OverrideName", req.Name)
	})
	
	t.Run("whitespace_in_request_names", func(t *testing.T) {
		content := `### Request With Spaces
GET https://api.example.com/users

###

# @name    nameWithSpaces    
POST https://api.example.com/users`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 2)
		
		req1 := getRequest(t, parsed, 0)
		assert.Equal(t, "Request With Spaces", req1.Name)
		
		req2 := getRequest(t, parsed, 1)
		assert.Equal(t, "nameWithSpaces", req2.Name)
	})
	
	t.Run("unnamed_requests", func(t *testing.T) {
		content := `GET https://api.example.com/users

###

POST https://api.example.com/users`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 2)
		
		req1 := getRequest(t, parsed, 0)
		assert.Empty(t, req1.Name)
		
		req2 := getRequest(t, parsed, 1)
		assert.Empty(t, req2.Name)
	})
}

// TestParseComments tests comment parsing functionality (T24)
func TestParseComments(t *testing.T) {
	t.Run("hash_comments_are_ignored", func(t *testing.T) {
		content := `# This is a comment
GET https://api.example.com/users
# Another comment
Content-Type: application/json
# Comment in body section

{
  "name": "test"
}`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 1)
		
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, "https://api.example.com/users", req.URL.String())
		assert.Equal(t, "application/json", req.Headers.Get("Content-Type"))
		// Body should contain the JSON content
		assert.Contains(t, req.RawBody, `"name": "test"`)
		// Comments should not appear in the body
		assert.NotContains(t, req.RawBody, "# Comment in body section")
	})
	
	t.Run("slash_comments_are_ignored", func(t *testing.T) {
		content := `// This is a slash comment
GET https://api.example.com/users
// Another slash comment
Content-Type: application/json

// Comment before body
{
  "name": "test"
}`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 1)
		
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, "https://api.example.com/users", req.URL.String())
		assert.Equal(t, "application/json", req.Headers.Get("Content-Type"))
	})
	
	t.Run("mixed_comment_styles", func(t *testing.T) {
		content := `# Hash comment
// Slash comment
GET https://api.example.com/users
# Another hash comment
// Another slash comment
Content-Type: application/json

{
  "name": "test"
}`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 1)
		
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, "application/json", req.Headers.Get("Content-Type"))
	})
	
	t.Run("comments_with_directives", func(t *testing.T) {
		content := `# This is a regular comment
# @name testRequest
// This is also a regular comment
GET https://api.example.com/users`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 1)
		
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "testRequest", req.Name)
		assert.Equal(t, "GET", req.Method)
	})
	
	t.Run("comments_around_separators", func(t *testing.T) {
		content := `GET https://api.example.com/users

# Comment before separator
###
// Comment after separator

POST https://api.example.com/users`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 2)
		
		req1 := getRequest(t, parsed, 0)
		assert.Equal(t, "GET", req1.Method)
		
		req2 := getRequest(t, parsed, 1)
		assert.Equal(t, "POST", req2.Method)
	})
	
	t.Run("comments_with_special_characters", func(t *testing.T) {
		content := `# Comment with special chars: !@#$%^&*()
// Comment with URLs: https://example.com/path?param=value
GET https://api.example.com/users`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 1)
		
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "GET", req.Method)
	})
}

// TestParseHTTPMethods tests parsing of various HTTP methods (T25)
func TestParseHTTPMethods(t *testing.T) {
	testCases := []struct {
		name           string
		method         string
		expectedMethod string
		shouldParse    bool
	}{
		{"GET_method", "GET", "GET", true},
		{"POST_method", "POST", "POST", true},
		{"PUT_method", "PUT", "PUT", true},
		{"DELETE_method", "DELETE", "DELETE", true},
		{"PATCH_method", "PATCH", "PATCH", true},
		{"HEAD_method", "HEAD", "HEAD", true},
		{"OPTIONS_method", "OPTIONS", "OPTIONS", true},
		{"TRACE_method", "TRACE", "TRACE", true},
		{"CONNECT_method", "CONNECT", "CONNECT", true},
		{"lowercase_get", "get", "get", true}, // Parser should preserve case
		{"mixed_case", "Get", "Get", true},
		{"custom_method", "CUSTOM", "CUSTOM", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := fmt.Sprintf("%s https://api.example.com/test", tc.method)
			
			parsed := mustParseHTTP(t, content)
			
			if tc.shouldParse {
				assertRequestCount(t, parsed, 1)
				req := getRequest(t, parsed, 0)
				assert.Equal(t, tc.expectedMethod, req.Method)
				assert.Equal(t, "https://api.example.com/test", req.URL.String())
			} else {
				assertRequestCount(t, parsed, 0)
			}
		})
	}
}

// TestParseInvalidMethods tests handling of edge cases in HTTP method parsing
func TestParseMethodEdgeCases(t *testing.T) {
	t.Run("leading_space_gets_trimmed", func(t *testing.T) {
		content := ` https://api.example.com/test`
		
		parsed := mustParseHTTP(t, content)
		// The parser interprets this as a short-form GET request
		assertRequestCount(t, parsed, 1)
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "GET", req.Method) // Parser defaults to GET for URL-only lines
	})
	
	t.Run("numeric_method_accepted", func(t *testing.T) {
		content := `123 https://api.example.com/test`
		
		parsed := mustParseHTTP(t, content)
		// Parser accepts numeric methods
		assertRequestCount(t, parsed, 1)
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "123", req.Method)
	})
	
	t.Run("method_with_spaces_treats_as_method_and_url", func(t *testing.T) {
		content := `GET PUT https://api.example.com/test`
		
		parsed := mustParseHTTP(t, content)
		// Parser treats first token as method, rest as URL
		assertRequestCount(t, parsed, 1)
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "GET", req.Method)
		// URL contains the "PUT" as part of the URL
		assert.Contains(t, req.RawURLString, "PUT")
	})
	
	t.Run("method_only_no_url_ignored", func(t *testing.T) {
		content := `GET`
		
		parsed := mustParseHTTP(t, content)
		// Should not parse as a valid request without URL
		assertRequestCount(t, parsed, 0)
	})
	
	t.Run("special_characters_in_method", func(t *testing.T) {
		content := `GET-TEST https://api.example.com/test`
		
		parsed := mustParseHTTP(t, content)
		assertRequestCount(t, parsed, 1)
		req := getRequest(t, parsed, 0)
		assert.Equal(t, "GET-TEST", req.Method)
	})
}