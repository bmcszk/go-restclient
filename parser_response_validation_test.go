package restclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR7.1.1 - Response Validation: $any Placeholder
// Corresponds to: http_syntax.md "Response Validation Placeholders", "$any"
// This test verifies the parser's ability to correctly process the '$any' placeholder
// within expected response (.hresp) files. It ensures that the placeholder is recognized
// and the response body containing it is parsed correctly.
// It uses 'testdata/response_validation/expected_response_any.hresp'.
func TestParseResponseValidationPlaceholder_Any(t *testing.T) {
	// Given: An expected response with $any placeholders
	expectedResponseFile := "testdata/response_validation/expected_response_any.hresp"

	// When: We parse the expected response file
	expectedResponses, err := parseExpectedResponseFile(expectedResponseFile)

	// Then: The parsing should succeed and contain $any placeholders
	require.NoError(t, err, "Should parse expected response file without error")
	require.Len(t, expectedResponses, 1, "Should have one expected response")

	resp := expectedResponses[0]
	require.NotNil(t, resp.StatusCode, "StatusCode should not be nil")
	assert.Equal(t, 200, *resp.StatusCode, "Status code should be 200")
	assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"), "Content-Type should be application/json")

	// Verify the body was parsed correctly and contains placeholders
	require.NotNil(t, resp.Body, "Body should not be nil")
	assert.Contains(t, *resp.Body, "{{$any}}", "Response body should contain $any placeholder")
	assert.Contains(t, *resp.Body, "prefix_match", "Response body should contain prefix_match field")
	assert.Contains(t, *resp.Body, "hello{{$any}}", "Response should contain prefix with $any placeholder")
	assert.Contains(t, *resp.Body, "{{$any}}world", "Response should contain suffix with $any placeholder")
}

// PRD-COMMENT: FR7.1.2 - Response Validation: $regexp Placeholder
// Corresponds to: http_syntax.md "Response Validation Placeholders", "$regexp"
// This test verifies the parser's ability to correctly process the '$regexp' placeholder
// within expected response (.hresp) files. It checks that various regular expressions,
// including those with special characters requiring escaping, are parsed correctly.
// It uses 'testdata/response_validation/expected_response_regexp.hresp'.
func TestParseResponseValidationPlaceholder_Regexp(t *testing.T) {
	// Given: An expected response with $regexp placeholders
	expectedResponseFile := "testdata/response_validation/expected_response_regexp.hresp"

	// When: We parse the expected response file
	expectedResponses, err := parseExpectedResponseFile(expectedResponseFile)

	// Then: The parsing should succeed and contain $regexp placeholders
	require.NoError(t, err, "Should parse expected response file without error")
	require.Len(t, expectedResponses, 1, "Should have one expected response")

	resp := expectedResponses[0]
	require.NotNil(t, resp.StatusCode, "StatusCode should not be nil")
	assert.Equal(t, 200, *resp.StatusCode, "Status code should be 200")

	// Verify the body was parsed correctly and contains regexp placeholders
	require.NotNil(t, resp.Body, "Body should not be nil")

	// Instead of trying to construct the exact string with the correct escaping,
	// we'll verify each pattern individually with proper contains checks

	// Check id regex pattern
	assert.Contains(t, *resp.Body, "\"id\": \"{{$regexp '[A-Z0-9]+'}}\"", "Response body should contain id regex pattern")

	// Check email regex pattern - the parser adds extra escaping for backslashes
	assert.Contains(t, *resp.Body, "\"email\": \"{{$regexp", "Response body should contain email regex pattern start")
	assert.Contains(t, *resp.Body, "@[a-zA-Z0-9.-]+", "Response body should contain email domain pattern")

	// Check phone regex pattern
	assert.Contains(t, *resp.Body, "\"phone\": \"{{$regexp", "Response body should contain phone regex pattern")
	assert.Contains(t, *resp.Body, "\\\\d{3}-\\\\d{3}-\\\\d{4}", "Response body should contain phone number format")
}

// PRD-COMMENT: FR7.1.3 - Response Validation: $anyGuid Placeholder
// Corresponds to: http_syntax.md "Response Validation Placeholders", "$anyGuid"
// This test verifies the parser's ability to correctly process the '$anyGuid' (or '$anyUuid')
// placeholder within expected response (.hresp) files. It ensures that these placeholders,
// used for matching any valid UUID/GUID, are parsed correctly.
// It uses 'testdata/response_validation/expected_response_guid.hresp'.
func TestParseResponseValidationPlaceholder_AnyGuid(t *testing.T) {
	// Given: An expected response with $anyGuid placeholders
	expectedResponseFile := "testdata/response_validation/expected_response_guid.hresp"

	// When: We parse the expected response file
	expectedResponses, err := parseExpectedResponseFile(expectedResponseFile)

	// Then: The parsing should succeed and contain $anyGuid placeholders
	require.NoError(t, err, "Should parse expected response file without error")
	require.Len(t, expectedResponses, 1, "Should have one expected response")

	resp := expectedResponses[0]
	require.NotNil(t, resp.StatusCode, "StatusCode should not be nil")
	assert.Equal(t, 200, *resp.StatusCode, "Status code should be 200")

	// Verify the body was parsed correctly and contains UUID placeholders
	require.NotNil(t, resp.Body, "Body should not be nil")
	assert.Contains(t, *resp.Body, "{{$anyGuid}}", "Response body should contain $anyGuid placeholder")
	assert.Contains(t, *resp.Body, "REF-{{$anyGuid}}-2025",
		"Response should contain $anyGuid placeholder within a string")
}

// PRD-COMMENT: FR7.1.4 - Response Validation: $anyTimestamp Placeholder
// Corresponds to: http_syntax.md "Response Validation Placeholders", "$anyTimestamp"
// This test verifies the parser's ability to correctly process the '$anyTimestamp'
// placeholder within expected response (.hresp) files. This placeholder is used for matching
// any valid Unix timestamp (integer).
// It uses 'testdata/response_validation/expected_response_timestamp.hresp'.
func TestParseResponseValidationPlaceholder_AnyTimestamp(t *testing.T) {
	// Given: An expected response with $anyTimestamp placeholders
	expectedResponseFile := "testdata/response_validation/expected_response_timestamp.hresp"

	// When: We parse the expected response file
	expectedResponses, err := parseExpectedResponseFile(expectedResponseFile)

	// Then: The parsing should succeed and contain $anyTimestamp placeholders
	require.NoError(t, err, "Should parse expected response file without error")
	require.Len(t, expectedResponses, 1, "Should have one expected response")

	resp := expectedResponses[0]
	require.NotNil(t, resp.StatusCode, "StatusCode should not be nil")
	assert.Equal(t, 200, *resp.StatusCode, "Status code should be 200")

	// Verify the body was parsed correctly and contains timestamp placeholders
	require.NotNil(t, resp.Body, "Body should not be nil")
	assert.Contains(t, *resp.Body, "{{$anyTimestamp}}", "Response body should contain $anyTimestamp placeholder")
	// Check for nested timestamp placeholder
	assert.Contains(t, *resp.Body, "\"nested\": {", "Response body should contain nested structure")
	assert.Contains(t, *resp.Body, "\"timestamp\": \"{{$anyTimestamp}}",
		"Response body should contain nested timestamp placeholder")
}

// PRD-COMMENT: FR7.1.5 - Response Validation: $anyDatetime Placeholder
// Corresponds to: http_syntax.md "Response Validation Placeholders", "$anyDatetime"
// This test verifies the parser's ability to correctly process the '$anyDatetime'
// placeholder with various format specifiers (RFC3339, RFC1123, custom Go format string)
// within expected response (.hresp) files.
// It uses 'testdata/response_validation/expected_response_datetime.hresp'.
func TestParseResponseValidationPlaceholder_AnyDatetime(t *testing.T) {
	// Given: An expected response with $anyDatetime placeholders
	expectedResponseFile := "testdata/response_validation/expected_response_datetime.hresp"

	// When: We parse the expected response file
	expectedResponses, err := parseExpectedResponseFile(expectedResponseFile)

	// Then: The parsing should succeed and contain $anyDatetime placeholders
	require.NoError(t, err, "Should parse expected response file without error")
	require.Len(t, expectedResponses, 1, "Should have one expected response")

	resp := expectedResponses[0]
	require.NotNil(t, resp.StatusCode, "StatusCode should not be nil")
	assert.Equal(t, 200, *resp.StatusCode, "Status code should be 200")

	// Verify the body was parsed correctly and contains datetime placeholders with formats
	require.NotNil(t, resp.Body, "Body should not be nil")
	assert.Contains(t, *resp.Body, "{{$anyDatetime 'RFC3339'}}",
		"Response should contain $anyDatetime placeholder with RFC3339 format")
	assert.Contains(t, *resp.Body, "{{$anyDatetime 'RFC1123'}}",
		"Response should contain $anyDatetime placeholder with RFC1123 format")
	assert.Contains(t, *resp.Body, "{{$anyDatetime '2006-01-02'}}",
		"Response should contain $anyDatetime placeholder with custom format")
}

// PRD-COMMENT: FR8.1 - Chained Requests: Response Reference Parsing
// Corresponds to: http_syntax.md "Chained Requests and Response References"
// This test verifies the parser's ability to correctly parse .http files containing chained requests,
// where subsequent requests reference values from the responses of previous requests (e.g., {{requestName.response.body.field}}).
// It ensures that request names are captured and that RawURLString and RawBody containing these references are preserved.
// It uses 'testdata/response_validation/chained_requests.http'.
func TestParseChainedRequests(t *testing.T) {
	// Given: HTTP file with chained requests using response references
	requestFile := "testdata/response_validation/chained_requests.http"
	// Create a test client
	client, err := NewClient()
	require.NoError(t, err, "Should create client without error")

	// When: We parse the file containing chained requests
	parsedFile, parseErr := parseRequestFile(requestFile, client, []string{})

	// Then: The parsing should succeed and all requests should be properly linked
	require.NoError(t, parseErr, "Should parse chained requests file without error")
	require.Len(t, parsedFile.Requests, 3, "Should have three requests")

	// Verify first request
	firstReq := parsedFile.Requests[0]
	require.Equal(t, "firstRequest", firstReq.Name, "First request should be named 'firstRequest'")
	require.Equal(t, "https://api.example.com/user", firstReq.URL.String(), "First URL should be correct")

	// Verify second request has references to first request's response
	secondRequest := parsedFile.Requests[1]
	assert.Equal(t, "GET", secondRequest.Method, "Second request should be GET")
	// For requests with variables in RawURLString, URL parsing is deferred.
	assert.Nil(t, secondRequest.URL, "Second request URL should be nil due to deferred parsing of variables")

	// Check URL path components and template variables in the RawURLString
	assert.Contains(t, secondRequest.RawURLString, "/user/", "Second request RawURLString should contain user path component")
	assert.Contains(t, secondRequest.RawURLString, "/profile", "Second request RawURLString should contain profile path component")
	assert.Contains(t, secondRequest.RawURLString, "{{firstRequest.response.body.id}}", "Second request RawURLString should contain reference to first request's response id")

	authHeader := secondRequest.Headers.Get("Authorization")
	require.Contains(t, authHeader, "{{firstRequest.response.body.token}}",
		"Second request should have authorization header with token from first response")

	// Verify third request has references to both first and second request responses
	thirdReq := parsedFile.Requests[2]
	require.Equal(t, "thirdRequest", thirdReq.Name, "Third request should be named 'thirdRequest'")
	require.NotNil(t, thirdReq.RawBody, "Third request should have a body")
	require.Contains(t, string(thirdReq.RawBody), "{{firstRequest.response.body.id}}",
		"Third request body should reference first request's id")
	require.Contains(t, string(thirdReq.RawBody), "{{secondRequest.response.body.name}}",
		"Third request body should reference second request's name field")
	require.Contains(t, string(thirdReq.RawBody), "{{secondRequest.response.body.email}}",
		"Third request body should reference second request's email field")
}
