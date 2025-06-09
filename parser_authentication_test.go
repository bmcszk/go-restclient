package restclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to parse HTTP test files and make basic assertions
func parseAuthTestFile(t *testing.T, filePath string) *ParsedFile {
	// When
	parsedFile, err := parseRequestFile(filePath, nil, make([]string, 0))

	// Then
	require.NoError(t, err, "Failed to parse request file")
	require.NotNil(t, parsedFile, "Parsed file should not be nil")
	require.NotEmpty(t, parsedFile.Requests, "Parsed file should contain at least one request")

	return parsedFile
}

// PRD-COMMENT: FR5.1 - Basic Authentication via Authorization Header
// Corresponds to: http_syntax.md "Authentication" (Basic Auth section)
// This test verifies the parser's ability to correctly interpret the 'Authorization: Basic <credentials>' header.
// It ensures that the header is parsed and stored appropriately for later use by the client.
// The test uses 'testdata/authentication/basic_auth_header.http'.
func TestBasicAuthHeader(t *testing.T) {
	// Given
	const requestFilePath = "testdata/authentication/basic_auth_header.http"

	// When
	parsedFile := parseAuthTestFile(t, requestFilePath)
	req := parsedFile.Requests[0]

	// Then
	require.NotNil(t, req, "Request should not be nil")
	require.NotNil(t, req.Headers, "Headers should not be nil")

	authHeader, exists := req.Headers["Authorization"]
	require.True(t, exists, "Authorization header should exist")
	require.Len(t, authHeader, 1, "Authorization header should have one value")
	assert.Equal(t, "Basic dXNlcm5hbWU6cGFzc3dvcmQ=", authHeader[0], "Basic auth header value mismatch")
}

// PRD-COMMENT: FR5.1 - Basic Authentication via URL
// Corresponds to: http_syntax.md "Authentication" (Basic Auth section)
// This test verifies the parser's ability to extract username and password credentials embedded directly in the request URL (e.g., 'user:pass@domain.com').
// It ensures these credentials are correctly parsed and associated with the request.
// The test uses 'testdata/authentication/basic_auth_url.http'.
func TestBasicAuthURL(t *testing.T) {
	// Given
	const requestFilePath = "testdata/authentication/basic_auth_url.http"

	// When
	parsedFile := parseAuthTestFile(t, requestFilePath)
	req := parsedFile.Requests[0]

	// Then
	require.NotNil(t, req, "Request should not be nil")
	require.NotNil(t, req.URL, "URL should not be nil")

	// Check that URL has user info
	userInfo := req.URL.User
	require.NotNil(t, userInfo, "URL should contain user info")

	username := userInfo.Username()
	password, passwordSet := userInfo.Password()

	assert.Equal(t, "username", username, "Username in URL mismatch")
	assert.True(t, passwordSet, "Password should be set in URL")
	assert.Equal(t, "password", password, "Password in URL mismatch")
	assert.Equal(t, "api.example.com", req.URL.Host, "Host in URL mismatch")
	assert.Equal(t, "/secured", req.URL.Path, "Path in URL mismatch")
}

// PRD-COMMENT: FR5.2 - Bearer Token Authentication
// Corresponds to: http_syntax.md "Authentication" (Bearer Token section)
// This test verifies the parser's ability to correctly interpret the 'Authorization: Bearer <token>' header.
// It ensures that the bearer token is extracted and stored for the request.
// The test uses 'testdata/authentication/bearer_token.http'.
func TestBearerTokenAuth(t *testing.T) {
	// Given
	const requestFilePath = "testdata/authentication/bearer_token.http"
	const expectedToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ"

	// When
	parsedFile := parseAuthTestFile(t, requestFilePath)
	req := parsedFile.Requests[0]

	// Then
	require.NotNil(t, req, "Request should not be nil")
	require.NotNil(t, req.Headers, "Headers should not be nil")

	authHeader, exists := req.Headers["Authorization"]
	require.True(t, exists, "Authorization header should exist")
	require.Len(t, authHeader, 1, "Authorization header should have one value")
	assert.Equal(t, "Bearer "+expectedToken, authHeader[0], "Bearer token header value mismatch")
}

// PRD-COMMENT: FR5.3 - Authentication with Response References & FR7.2 - Response Reference Variables
// Corresponds to:
// - FR5.3: http_syntax.md "Authentication" (implicitly, via OAuth example), "Response Reference Variables" (general syntax)
// - FR7.2: http_syntax.md "Response Reference Variables", "Response Handler > Response reference"
// This test, TestOAuthFlowWithRequestReferences, validates the parser's ability to handle a common OAuth 2.0 client credentials flow
// where a token is fetched in one request and its value (e.g., `{{getToken.response.body.access_token}}`) is referenced
// in the Authorization header of a subsequent request.
// Key aspects tested:
// 1. Correct parsing of multiple requests within a single file.
// 2. Identification and parsing of response reference variables (e.g., `{{reqName.response.body.field}}`) in request headers.
// 3. Ensuring the structure of an OAuth token request (POST, form-urlencoded body) is correctly parsed.
// 4. Ensuring the structure of a subsequent API request using the referenced token is correctly parsed.
// The test uses 'testdata/authentication/oauth_flow.http'.
func TestOAuthFlowWithRequestReferences(t *testing.T) {
	// Given
	const requestFilePath = "testdata/authentication/oauth_flow.http"

	// When
	parsedFile := parseAuthTestFile(t, requestFilePath)

	// Then
	require.NotNil(t, parsedFile, "Parsed file should not be nil")
	require.Len(t, parsedFile.Requests, 2, "Should have 2 requests for OAuth flow")

	// First request should be the token request
	tokenRequest := parsedFile.Requests[0]
	require.NotNil(t, tokenRequest, "Token request should not be nil")
	assert.Equal(t, "getToken", tokenRequest.Name, "Token request name mismatch")
	assert.Equal(t, "POST", tokenRequest.Method, "Token request method mismatch")
	assert.Equal(t, "https://oauth.example.com/token", tokenRequest.URL.String(), "Token request URL mismatch")

	// Check content type is form-urlencoded
	contentTypeHeaders, exists := tokenRequest.Headers["Content-Type"]
	require.True(t, exists, "Content-Type header should exist in token request")
	require.Len(t, contentTypeHeaders, 1, "Content-Type header should have one value")
	assert.Equal(t, "application/x-www-form-urlencoded", contentTypeHeaders[0], "Content-Type header mismatch")

	// Check body has OAuth parameters
	assert.Contains(t, tokenRequest.RawBody, "grant_type=client_credentials", "Token request body should contain grant_type")
	assert.Contains(t, tokenRequest.RawBody, "client_id=my-client", "Token request body should contain client_id")
	assert.Contains(t, tokenRequest.RawBody, "client_secret=my-secret", "Token request body should contain client_secret")

	// Second request should reference the token
	apiRequest := parsedFile.Requests[1]
	require.NotNil(t, apiRequest, "API request should not be nil")
	assert.Equal(t, "useToken", apiRequest.Name, "API request name mismatch")
	assert.Equal(t, "GET", apiRequest.Method, "API request method mismatch")
	assert.Equal(t, "https://api.example.com/secured", apiRequest.URL.String(), "API request URL mismatch")

	// Check authorization header contains reference to previous request
	authHeader, exists := apiRequest.Headers["Authorization"]
	require.True(t, exists, "Authorization header should exist in API request")
	require.Len(t, authHeader, 1, "Authorization header should have one value")
	assert.Equal(t, "Bearer {{getToken.response.body.access_token}}", authHeader[0], "Authorization header should reference token from previous request")
}
