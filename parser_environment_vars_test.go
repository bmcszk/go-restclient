package restclient

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseRequestFile_EnvironmentVariables tests parsing of environment variables in requests (FR2.1, FR2.2)
func TestParseRequestFile_EnvironmentVariables(t *testing.T) {
	// Given
	const requestFilePath = "testdata/variables/environment_variables.http"

	// Setup environment variables for testing
	os.Setenv("base_url", "https://api.example.com")
	os.Setenv("protocol", "https")
	os.Setenv("host", "api.example.com")
	os.Setenv("port", "8080")
	os.Setenv("api_token", "xyz123")
	os.Setenv("api_version", "v2")
	os.Setenv("auth_token", "Bearer_Token_123")
	os.Setenv("custom_header_value", "custom-value")
	os.Setenv("username", "testuser")
	os.Setenv("email", "test@example.com")
	os.Setenv("api_key", "api_key_123")
	os.Setenv("theme", "dark")
	os.Setenv("notifications_enabled", "true")
	defer func() {
		// Clean up environment variables after test
		vars := []string{
			"base_url", "protocol", "host", "port", "api_token", "api_version",
			"auth_token", "custom_header_value", "username", "email", "api_key",
			"theme", "notifications_enabled", "api_url",
		}
		for _, v := range vars {
			os.Unsetenv(v)
		}
	}()

	// When
	parsedFile, err := parseRequestFile(requestFilePath, nil, make([]string, 0))

	// Then
	require.NoError(t, err, "Failed to parse request file")
	require.NotNil(t, parsedFile, "Parsed file should not be nil")
	require.Len(t, parsedFile.Requests, 6, "Expected 6 requests")

	// Verify environment variable placeholders are preserved in raw URL string
	assert.Equal(t, "{{base_url}}/api/users", parsedFile.Requests[0].RawURLString, "Basic environment variable in URL mismatch")

	// Verify multiple environment variables in URL
	assert.Equal(t, "{{protocol}}://{{host}}:{{port}}/api/users", parsedFile.Requests[1].RawURLString, "Multiple environment variables in URL mismatch")

	// Verify environment variables in query parameters
	assert.Equal(t, "https://example.com/api/users?token={{api_token}}&version={{api_version}}", parsedFile.Requests[2].RawURLString, "Environment variables in query parameters mismatch")

	// Verify environment variables in headers
	headerReq := parsedFile.Requests[3]
	assert.Equal(t, "Bearer {{auth_token}}", headerReq.Headers.Get("Authorization"), "Authorization header with environment variable mismatch")
	assert.Equal(t, "{{api_version}}", headerReq.Headers.Get("X-API-Version"), "X-API-Version header with environment variable mismatch")
	assert.Equal(t, "{{custom_header_value}}", headerReq.Headers.Get("Custom-Header"), "Custom-Header with environment variable mismatch")

	// Verify environment variables in request body
	bodyReq := parsedFile.Requests[4]
	assert.Contains(t, bodyReq.RawBody, "\"username\": \"{{username}}\"", "Username placeholder in JSON body not preserved")
	assert.Contains(t, bodyReq.RawBody, "\"email\": \"{{email}}\"", "Email placeholder in JSON body not preserved")
	assert.Contains(t, bodyReq.RawBody, "\"apiKey\": \"{{api_key}}\"", "API key placeholder in JSON body not preserved")
	assert.Contains(t, bodyReq.RawBody, "\"theme\": \"{{theme}}\"", "Theme placeholder in JSON body not preserved")
	assert.Contains(t, bodyReq.RawBody, "\"notifications\": {{notifications_enabled}}", "Notifications placeholder in JSON body not preserved")

	// Verify environment variable with default value
	assert.Equal(t, "{{api_url:https://default-api.example.com}}/v1/resources", parsedFile.Requests[5].RawURLString, "Environment variable with default value mismatch")
}

// TestParseRequestFile_VariableDefinitions tests parsing of variable definitions in request files (FR2.3)
func TestParseRequestFile_VariableDefinitions(t *testing.T) {
	// Given
	const requestFilePath = "testdata/variables/variable_definitions.http"

	// When
	parsedFile, err := parseRequestFile(requestFilePath, nil, make([]string, 0))

	// Then
	require.NoError(t, err, "Failed to parse request file")
	require.NotNil(t, parsedFile, "Parsed file should not be nil")
	require.Len(t, parsedFile.Requests, 3, "Expected 3 requests")

	// Verify file variables are defined and used
	assert.Equal(t, "{{base_url}}/{{api_version}}/users", parsedFile.Requests[0].RawURLString, "Request using file variables mismatch")
	assert.Equal(t, "Bearer {{auth_token}}", parsedFile.Requests[0].Headers.Get("Authorization"), "Authorization header mismatch")

	// Verify new variable defined mid-file
	assert.Equal(t, "{{base_url}}/{{api_version}}/users/{{user_id}}", parsedFile.Requests[1].RawURLString, "Request using new variable mismatch")

	// Verify multiple variables defined in sequence
	assert.Equal(t, "{{protocol}}://{{host}}:{{port}}/status", parsedFile.Requests[2].RawURLString, "Request using sequence-defined variables mismatch")
}

// TestParseRequestFile_VariableScoping tests variable scoping and references (FR2.4)
func TestParseRequestFile_VariableScoping(t *testing.T) {
	// Given
	const requestFilePath = "testdata/variables/variable_references.http"

	// When
	parsedFile, err := parseRequestFile(requestFilePath, nil, make([]string, 0))

	// Then
	require.NoError(t, err, "Failed to parse request file")
	require.NotNil(t, parsedFile, "Parsed file should not be nil")
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
