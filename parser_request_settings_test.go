package restclient

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to parse HTTP test files and make basic assertions
func parseSettingsTestFile(t *testing.T, filePath string) *ParsedFile {
	// When
	parsedFile, err := parseRequestFile(filePath, nil, make([]string, 0))

	// Then
	require.NoError(t, err, "Failed to parse request file")
	require.NotNil(t, parsedFile, "Parsed file should not be nil")
	require.NotEmpty(t, parsedFile.Requests, "Parsed file should contain at least one request")

	return parsedFile
}

// PRD-COMMENT: FR6.1 - Request Setting: @name Directive
// Corresponds to: http_syntax.md "Request Settings", "@name"
// This test verifies the parser's ability to correctly process the '@name' directive.
// It ensures that the specified name is assigned to the Request.Name field, allowing for easier
// identification and selection of requests, especially when multiple requests are defined in a single file.
// It uses 'testdata/request_settings/name_directive.http'.
func TestNameDirective(t *testing.T) {
	// Given
	const requestFilePath = "testdata/request_settings/name_directive.http"

	// When
	parsedFile := parseSettingsTestFile(t, requestFilePath)

	// Then
	require.Len(t, parsedFile.Requests, 2, "Should have 2 requests in file")

	// First request should have name from @name directive
	firstReq := parsedFile.Requests[0]
	assert.Equal(t, "getUsers", firstReq.Name, "Request name should be set from @name directive")
	assert.Equal(t, "GET", firstReq.Method, "Request method mismatch")
	assert.Equal(t, "https://api.example.com/users", firstReq.URL.String(), "Request URL mismatch")

	// Second request should have no explicit name (or default name based on implementation)
	secondReq := parsedFile.Requests[1]
	assert.NotEqual(t, "getUsers", secondReq.Name, "Second request should not have the same name as first request")
	assert.Equal(t, "GET", secondReq.Method, "Request method mismatch")
	assert.Equal(t, "https://api.example.com/products", secondReq.URL.String(), "Request URL mismatch")
}

// PRD-COMMENT: FR6.1 - Request Setting: @no-redirect Directive
// Corresponds to: http_syntax.md "Request Settings", "@no-redirect"
// This test verifies the parser's ability to correctly process the '@no-redirect' directive.
// It ensures that when this directive is present, the Request.NoRedirect field is set to true,
// signaling that HTTP redirects should not be followed automatically for this specific request.
// It uses 'testdata/request_settings/no_redirect_directive.http'.
func TestNoRedirectDirective(t *testing.T) {
	// Given
	const requestFilePath = "testdata/request_settings/no_redirect_directive.http"

	// When
	parsedFile := parseSettingsTestFile(t, requestFilePath)

	// Then
	require.Len(t, parsedFile.Requests, 2, "Should have 2 requests in file")

	// First request should have redirect enabled (by default)
	firstReq := parsedFile.Requests[0]
	assert.False(t, firstReq.NoRedirect, "First request should allow redirects by default")
	assert.Equal(t, "GET", firstReq.Method, "Request method mismatch")
	assert.Equal(t, "https://api.example.com/redirect-me", firstReq.URL.String(), "Request URL mismatch")

	// Second request should have redirect disabled due to @no-redirect directive
	secondReq := parsedFile.Requests[1]
	assert.True(t, secondReq.NoRedirect, "Second request should have redirects disabled via @no-redirect directive")
	assert.Equal(t, "GET", secondReq.Method, "Request method mismatch")
	assert.Equal(t, "https://api.example.com/redirect-me", secondReq.URL.String(), "Request URL mismatch")
}

// PRD-COMMENT: FR6.1 - Request Setting: @no-cookie-jar Directive
// Corresponds to: http_syntax.md "Request Settings", "@no-cookie-jar"
// This test verifies the parser's ability to correctly process the '@no-cookie-jar' directive.
// It ensures that when this directive is present, the Request.NoCookieJar field is set to true,
// indicating that the client's cookie jar should not be used for this specific request (neither for sending stored cookies nor for saving new ones).
// It uses 'testdata/request_settings/no_cookie_jar_directive.http'.
func TestNoCookieJarDirective(t *testing.T) {
	// Given
	const requestFilePath = "testdata/request_settings/no_cookie_jar_directive.http"

	// When
	parsedFile := parseSettingsTestFile(t, requestFilePath)

	// Then
	require.Len(t, parsedFile.Requests, 2, "Should have 2 requests in file")

	// First request should have cookie jar enabled (by default)
	firstReq := parsedFile.Requests[0]
	assert.False(t, firstReq.NoCookieJar, "First request should use cookie jar by default")
	assert.Equal(t, "GET", firstReq.Method, "Request method mismatch")
	assert.Equal(t, "https://api.example.com/with-cookies", firstReq.URL.String(), "Request URL mismatch")

	// Second request should have cookie jar disabled due to @no-cookie-jar directive
	secondReq := parsedFile.Requests[1]
	assert.True(t, secondReq.NoCookieJar, "Second request should have cookie jar disabled via @no-cookie-jar directive")
	assert.Equal(t, "GET", secondReq.Method, "Request method mismatch")
	assert.Equal(t, "https://api.example.com/no-cookies", secondReq.URL.String(), "Request URL mismatch")
}

// PRD-COMMENT: FR6.2 - Request Setting: @timeout Directive
// Corresponds to: http_syntax.md "Request Settings", "@timeout"
// This test verifies the parser's ability to correctly process the '@timeout' directive.
// It ensures that the specified timeout value (in milliseconds) is parsed and assigned to the Request.Timeout field.
// This allows for configuring request-specific timeouts.
// It uses 'testdata/request_settings/timeout_directive.http'.
func TestTimeoutDirective(t *testing.T) {
	// Given
	const requestFilePath = "testdata/request_settings/timeout_directive.http"

	// When
	parsedFile := parseSettingsTestFile(t, requestFilePath)

	// Then
	require.Len(t, parsedFile.Requests, 2, "Should have 2 requests in file")

	// First request should have default timeout (0 or implementation defined)
	firstReq := parsedFile.Requests[0]
	assert.Equal(t, time.Duration(0), firstReq.Timeout, "First request should have default timeout")
	assert.Equal(t, "GET", firstReq.Method, "Request method mismatch")
	assert.Equal(t, "https://api.example.com/standard", firstReq.URL.String(), "Request URL mismatch")

	// Second request should have custom timeout from @timeout directive
	secondReq := parsedFile.Requests[1]
	assert.Equal(t, 5000*time.Millisecond, secondReq.Timeout, "Second request should have timeout set to 5000ms via @timeout directive")
	assert.Equal(t, "GET", secondReq.Method, "Request method mismatch")
	assert.Equal(t, "https://api.example.com/slow-endpoint", secondReq.URL.String(), "Request URL mismatch")
}
