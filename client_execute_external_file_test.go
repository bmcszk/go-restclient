package restclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// Create a test HTTP file
	httpContent := `@userId = user123
@userName = John Doe
@env = testing

### External File with Variable Substitution
POST https://httpbin.org/post
Content-Type: application/json

<@ ./test_vars.json`

	httpFile := filepath.Join(tempDir, "test.http")
	err = os.WriteFile(httpFile, []byte(httpContent), 0644)
	require.NoError(t, err)

	// Create client with additional programmatic variables
	client, err := NewClient(WithVars(map[string]interface{}{
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
	assert.Regexp(t, `"timestamp": "\d+"`, bodyStr)
}

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

	// Create a test HTTP file using static file reference
	httpContent := `@userId = user123
@userName = John Doe

### External File without Variable Substitution
POST https://httpbin.org/post
Content-Type: application/json

< ./test_static.json`

	httpFile := filepath.Join(tempDir, "test.http")
	err = os.WriteFile(httpFile, []byte(httpContent), 0644)
	require.NoError(t, err)

	// Create client
	client, err := NewClient()
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

func TestExecuteFile_ExternalFileWithEncoding(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	
	// Create a test file with special characters
	textContent := "Café français: été, naïve, résumé"
	
	// Write as UTF-8 (Go's default)
	textFile := filepath.Join(tempDir, "test_encoding.txt")
	err := os.WriteFile(textFile, []byte(textContent), 0644)
	require.NoError(t, err)

	// Create a test HTTP file with encoding specification
	httpContent := `### External File with UTF-8 Encoding
POST https://httpbin.org/post
Content-Type: text/plain

<@utf-8 ./test_encoding.txt`

	httpFile := filepath.Join(tempDir, "test.http")
	err = os.WriteFile(httpFile, []byte(httpContent), 0644)
	require.NoError(t, err)

	// Create client
	client, err := NewClient()
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
	client, err := NewClient()
	require.NoError(t, err)

	// Execute the file - should return error
	responses, err := client.ExecuteFile(context.Background(), httpFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error processing external file")
	require.Contains(t, err.Error(), "nonexistent.json")
	
	// Should still get responses array but with error
	require.Len(t, responses, 0) // No responses should be returned on file processing error
}

func TestClient_ProcessExternalFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	
	// Create a test file with variables
	fileContent := "Hello {{name}}, your ID is {{userId}}"
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	// Create client with variables
	client, err := NewClient(WithVars(map[string]interface{}{
		"name": "Alice",
		"userId": "12345",
	}))
	require.NoError(t, err)

	// Create a mock request
	request := &Request{
		ExternalFilePath:          "./test.txt",
		ExternalFileWithVariables: true,
		FilePath:                  tempDir + "/test.http", // Set directory context
		ActiveVariables:           make(map[string]string),
	}

	// Create a mock parsed file
	parsedFile := &ParsedFile{
		EnvironmentVariables: make(map[string]string),
		GlobalVariables:      make(map[string]string),
	}

	// Process the external file
	result, err := client.processExternalFile(request, parsedFile, make(map[string]string), os.LookupEnv)
	require.NoError(t, err)

	// Check that variables were substituted
	expected := "Hello Alice, your ID is 12345"
	assert.Equal(t, expected, result)
}

func TestClient_ReadFileWithEncoding(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	
	// Create test content
	content := "Hello World"
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	client, err := NewClient()
	require.NoError(t, err)

	tests := []struct {
		name     string
		encoding string
		wantErr  bool
	}{
		{"UTF-8", "utf-8", false},
		{"UTF8", "utf8", false},
		{"Latin1", "latin1", false},
		{"ISO-8859-1", "iso-8859-1", false},
		{"ASCII", "ascii", false},
		{"CP1252", "cp1252", false},
		{"Windows-1252", "windows-1252", false},
		{"Invalid", "invalid-encoding", true},
		{"Empty", "", false}, // Should default to UTF-8
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.readFileWithEncoding(testFile, tt.encoding)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported encoding")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, content, result)
			}
		})
	}
}

func TestClient_GetEncodingDecoder(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)

	tests := []struct {
		encoding string
		wantErr  bool
	}{
		{"latin1", false},
		{"iso-8859-1", false},
		{"cp1252", false},
		{"windows-1252", false},
		{"ascii", false},
		{"LATIN1", false}, // Case insensitive
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.encoding, func(t *testing.T) {
			decoder, err := client.getEncodingDecoder(tt.encoding)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, decoder)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, decoder)
			}
		})
	}
}