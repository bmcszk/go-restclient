package restclient

import (
	"testing"

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

// TestNameDirective tests parsing of @name directive (FR6.1)
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

// TestNoRedirectDirective tests parsing of @no-redirect directive (FR6.1)
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

// TestNoCookieJarDirective tests parsing of @no-cookie-jar directive (FR6.1)
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

// TestTimeoutDirective tests parsing of @timeout directive (FR6.2)
func TestTimeoutDirective(t *testing.T) {
	// Given
	const requestFilePath = "testdata/request_settings/timeout_directive.http"

	// When
	parsedFile := parseSettingsTestFile(t, requestFilePath)

	// Then
	require.Len(t, parsedFile.Requests, 2, "Should have 2 requests in file")

	// First request should have default timeout (0 or implementation defined)
	firstReq := parsedFile.Requests[0]
	assert.Equal(t, 0, firstReq.Timeout, "First request should have default timeout")
	assert.Equal(t, "GET", firstReq.Method, "Request method mismatch")
	assert.Equal(t, "https://api.example.com/standard", firstReq.URL.String(), "Request URL mismatch")

	// Second request should have custom timeout from @timeout directive
	secondReq := parsedFile.Requests[1]
	assert.Equal(t, 5000, secondReq.Timeout, "Second request should have timeout set to 5000ms via @timeout directive")
	assert.Equal(t, "GET", secondReq.Method, "Request method mismatch")
	assert.Equal(t, "https://api.example.com/slow-endpoint", secondReq.URL.String(), "Request URL mismatch")
}
