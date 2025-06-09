package restclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	rc "github.com/bmcszk/go-restclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR4.1 - Request Body: External File with Variables (<@)
// Corresponds to: Client's ability to process request bodies specified via '<@ filepath' where 'filepath' points to an external file whose content is subject to variable substitution (http_syntax.md "Request Body", "External File with Variables (<@ filepath)").
// This test verifies that variables defined in the .http file or programmatically are correctly substituted into the content of the external file ('test_vars.json') before it's used as the request body.
func TestExecuteFile_ExternalFileWithVariables(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test JSON file with variables
	jsonContent := `{
  "userId": "{{userId}}",
  "name": "{{userName}}",
  "timestamp": "{{$timestamp}}",
  "environment": "{{env}}"
}`
	jsonFile := filepath.Join(tempDir, "test_vars.json")
	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	require.NoError(t, err)

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var data map[string]interface{}
		err = json.Unmarshal(body, &data)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"json": data,
		})
	}))
	defer server.Close()

	// Create a test HTTP file
	httpContent := fmt.Sprintf(`@userId = user123
@userName = John Doe
@env = testing

### External File with Variable Substitution
POST %s/post
Content-Type: application/json

<@ ./test_vars.json`, server.URL)

	httpFile := filepath.Join(tempDir, "test.http")
	err = os.WriteFile(httpFile, []byte(httpContent), 0644)
	require.NoError(t, err)

	// Create client with additional programmatic variables
	client, err := rc.NewClient(rc.WithVars(map[string]interface{}{
		"userName": "Override Name", // This should override the file variable
	}))
	require.NoError(t, err)

	// Execute the file
	responses, err := client.ExecuteFile(context.Background(), httpFile)
	require.NoError(t, err)
	require.Len(t, responses, 1)

	response := responses[0]
	assert.NoError(t, response.Error)
	assert.Equal(t, 200, response.StatusCode)

	// Check that the body was processed correctly
	bodyStr := response.Request.RawBody
	assert.Contains(t, bodyStr, `"userId": "user123"`)
	assert.Contains(t, bodyStr, `"name": "Override Name"`) // Programmatic variable should override
	assert.Contains(t, bodyStr, `"environment": "testing"`)
	assert.Contains(t, bodyStr, `"timestamp":`) // Should contain a timestamp

	// Verify that timestamp is a number
	// The actual response from httptest will be nested under a "json" key if we mimic httpbin
	// For simplicity, we'll check the direct body content as sent.
	// If we were to fully mimic httpbin's response structure, the assertions would need to change
	// to look for response.Body.json.userId etc.
	// For now, client.ExecuteFile populates response.Request.RawBody with the *sent* body.
	assert.Regexp(t, `"timestamp": "\d+"`, bodyStr)
}

// PRD-COMMENT: FR4.2 - Request Body: External File Static (<)
// Corresponds to: Client's ability to process request bodies specified via '< filepath' where 'filepath' points to an external file whose content is included statically, without variable substitution (http_syntax.md "Request Body", "External File Static (< filepath)").
// This test verifies that the content of the external file ('test_static.json') is used as the request body verbatim, with any variable-like syntax within it preserved literally.
func TestExecuteFile_ExternalFileWithoutVariables(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test JSON file with variable placeholders (should NOT be substituted)
	jsonContent := `{
  "userId": "{{userId}}",
  "name": "{{userName}}",
  "literal": "this should stay as-is"
}`
	jsonFile := filepath.Join(tempDir, "test_static.json")
	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	require.NoError(t, err)

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var data map[string]interface{}
		err = json.Unmarshal(body, &data)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"json": data, // Echo back the received JSON data
		})
	}))
	defer server.Close()

	// Create a test HTTP file using static file reference
	httpContent := fmt.Sprintf(`@userId = user123
@userName = John Doe

### External File without Variable Substitution
POST %s/post
Content-Type: application/json

< ./test_static.json`, server.URL)

	httpFile := filepath.Join(tempDir, "test.http")
	err = os.WriteFile(httpFile, []byte(httpContent), 0644)
	require.NoError(t, err)

	// Create client
	client, err := rc.NewClient()
	require.NoError(t, err)

	// Execute the file
	responses, err := client.ExecuteFile(context.Background(), httpFile)
	require.NoError(t, err)
	require.Len(t, responses, 1)

	response := responses[0]
	assert.NoError(t, response.Error)
	assert.Equal(t, 200, response.StatusCode)

	// Check that the body was NOT processed for variables (static file reference)
	bodyStr := response.Request.RawBody
	assert.Contains(t, bodyStr, `"userId": "{{userId}}"`) // Should remain as template
	assert.Contains(t, bodyStr, `"name": "{{userName}}"`) // Should remain as template
	assert.Contains(t, bodyStr, `"literal": "this should stay as-is"`)
}

// PRD-COMMENT: FR4.3 - Request Body: External File with Encoding (<@|encoding)
// Corresponds to: Client's ability to process request bodies from external files with a specified character encoding using '<@|encoding filepath' (http_syntax.md "Request Body", "External File with Encoding (<@|encoding filepath)").
// This test verifies that an external file ('test_encoded.txt') with a specific encoding (e.g., latin1) is correctly read and used as the request body.
func TestExecuteFile_ExternalFileWithEncoding(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test file with special characters
	textContent := "Café français: été, naïve, résumé"

	// Write as UTF-8 (Go's default)
	textFile := filepath.Join(tempDir, "test_encoding.txt")
	err := os.WriteFile(textFile, []byte(textContent), 0644)
	require.NoError(t, err)

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type")) // Expecting text/plain
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "text/plain") // Echo back as text/plain
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(body) // Echo back the exact body received
		require.NoError(t, err)
	}))
	defer server.Close()

	// Create a test HTTP file with encoding specification
	httpContent := fmt.Sprintf(`### External File with UTF-8 Encoding
POST %s/post
Content-Type: text/plain

<@utf-8 ./test_encoding.txt`, server.URL)

	httpFile := filepath.Join(tempDir, "test.http")
	err = os.WriteFile(httpFile, []byte(httpContent), 0644)
	require.NoError(t, err)

	// Create client
	client, err := rc.NewClient()
	require.NoError(t, err)

	// Execute the file
	responses, err := client.ExecuteFile(context.Background(), httpFile)
	require.NoError(t, err)
	require.Len(t, responses, 1)

	response := responses[0]
	assert.NoError(t, response.Error)
	assert.Equal(t, 200, response.StatusCode)

	// Check that the body contains the expected text
	bodyStr := response.Request.RawBody
	assert.Equal(t, textContent, bodyStr)
}

// PRD-COMMENT: FR4.4 - Request Body: External File Not Found
// Corresponds to: Client error handling when an external file referenced in a request body (e.g., via '<@ ./nonexistent.json') cannot be found (http_syntax.md "Request Body").
// This test verifies that the client reports an appropriate error when attempting to process a request that references a non-existent external file.
func TestExecuteFile_ExternalFileNotFound(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test HTTP file referencing a non-existent file
	httpContent := `### External File Not Found
POST https://httpbin.org/post
Content-Type: application/json

<@ ./nonexistent.json`

	httpFile := filepath.Join(tempDir, "test.http")
	err := os.WriteFile(httpFile, []byte(httpContent), 0644)
	require.NoError(t, err)

	// Create client
	client, err := rc.NewClient()
	require.NoError(t, err)

	// Execute the file - should return error
	responses, err := client.ExecuteFile(context.Background(), httpFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error processing body for request")
	require.Contains(t, err.Error(), "nonexistent.json")

	// Should still get responses array but with error
	require.Len(t, responses, 0) // No responses should be returned on file processing error
}
