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
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	rc "github.com/bmcszk/go-restclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR4.1 - Request Body: External File with Variables (<@)
// Corresponds to: Client's ability to process request bodies specified via '<@ filepath' where 
// 'filepath' points to an external file whose content is subject to variable substitution 
// (http_syntax.md "Request Body", "External File with Variables (<@ filepath)").
// This test verifies that variables defined in the .http file or programmatically are correctly 
// substituted into the content of the external file ('test_vars.json') before it's used as the 
// request body.
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
		var data map[string]any
		err = json.Unmarshal(body, &data)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
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
	client, err := rc.NewClient(rc.WithVars(map[string]any{
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
// Corresponds to: Client's ability to process request bodies specified via '< filepath' where 
// 'filepath' points to an external file whose content is included statically, without variable 
// substitution (http_syntax.md "Request Body", "External File Static (< filepath)").
// This test verifies that the content of the external file ('test_static.json') is used as the 
// request body verbatim, with any variable-like syntax within it preserved literally.
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
		var data map[string]any
		err = json.Unmarshal(body, &data)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
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
// Corresponds to: Client's ability to process request bodies from external files with a 
// specified character encoding using '<@|encoding filepath' (http_syntax.md "Request Body", 
// "External File with Encoding (<@|encoding filepath)")

// TODO: Add tests for variable substitution within external files (<@ syntax).

func TestClientExecuteFileWithEncoding(t *testing.T) {
	testCases := getEncodingTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runEncodingTestCase(t, tc)
		})
	}
}

// encodingTestCase represents a test case for encoding
type encodingTestCase struct {
	name             string
	encodingName     string // e.g., "latin1", "cp1252"
	contentToWrite   string // Raw string content to be encoded
	expectedUTF8Body string // Expected body received by server (should be UTF-8)
	encoder          transform.Transformer
}

// getEncodingTestCases returns the test cases for encoding tests
func getEncodingTestCases() []encodingTestCase {
	return []encodingTestCase{
		{
			name:             "Latin-1 encoded file",
			encodingName:     "latin1",
			contentToWrite:   "H\u00e4llo W\u00f6rld! \u00d1ice to meet you. ?", 
			// Simulating content where € was replaced by ? as it's not in Latin-1.
			expectedUTF8Body: "Hällo Wörld! Ñice to meet you. ?", // How charmap.ISO8859_1 handles €
			encoder:          charmap.ISO8859_1.NewEncoder(),
		},
		{
			name:             "CP1252 (Windows-1252) encoded file",
			encodingName:     "cp1252",
			contentToWrite:   "H\u00e4llo W\u00f6rld! \u00d1ice to meet you. \u20ac\u2122", // € and ™ are in CP1252
			expectedUTF8Body: "Hällo Wörld! Ñice to meet you. €™",
			encoder:          charmap.Windows1252.NewEncoder(),
		},
		{
			name:             "ASCII encoded file (as subset of UTF-8)",
			encodingName:     "ascii",
			contentToWrite:   "Hello World! Nice to meet you.",
			expectedUTF8Body: "Hello World! Nice to meet you.",
			encoder:          nil, // Will be handled as UTF-8 by client
		},
		{
			name:             "UTF-8 encoded file (explicit)",
			encodingName:     "utf-8",
			contentToWrite:   "Hällo Wörld! Ñice to meet you. €😊",
			expectedUTF8Body: "Hällo Wörld! Ñice to meet you. €😊",
			encoder:          nil, // Will be handled as UTF-8 by client
		},
	}
}

// runEncodingTestCase executes a single encoding test case
func runEncodingTestCase(t *testing.T, tc encodingTestCase) {
	t.Helper()
	tempDir := setupTempDir(t)
	defer os.RemoveAll(tempDir)

	_ = createEncodedDataFile(t, tempDir, tc)
	bodyReceived := make(chan []byte, 1)
	mockServer := setupMockServer(t, bodyReceived)
	defer mockServer.Close()

	httpFilePath := createHTTPFile(t, tempDir, mockServer.URL, tc.encodingName)
	executeAndVerify(t, httpFilePath, tc.expectedUTF8Body, bodyReceived)
}

// setupTempDir creates a temporary directory for test files
func setupTempDir(t *testing.T) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "client_enc_test_*")
	require.NoError(t, err, "Failed to create temp dir")
	return tempDir
}

// createEncodedDataFile creates the encoded data file for testing
func createEncodedDataFile(t *testing.T, tempDir string, tc encodingTestCase) string {
	t.Helper()
	encodedDataFilePath := filepath.Join(tempDir, "encoded_body.txt")
	var fileBytes []byte
	var err error

	if tc.encoder != nil {
		fileBytes, _, err = transform.Bytes(tc.encoder, []byte(tc.contentToWrite))
		require.NoError(t, err, "Failed to encode contentToWrite")
	} else {
		fileBytes = []byte(tc.contentToWrite) // For UTF-8 or ASCII
	}

	require.NoError(t, os.WriteFile(encodedDataFilePath, fileBytes, 0644),
		"Failed to write encoded data file")
	return encodedDataFilePath
}

// setupMockServer creates a mock server for testing
func setupMockServer(t *testing.T, bodyReceived chan []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Logf("Mock server failed to read body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		bodyReceived <- body
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock response"))
	}))
}

// createHTTPFile creates the .http file for testing
func createHTTPFile(t *testing.T, tempDir, serverURL, encodingName string) string {
	t.Helper()
	httpFilePath := filepath.Join(tempDir, "request.http")
	httpFileContent := fmt.Sprintf(
		"POST %s\nContent-Type: text/plain\n\n<@%s encoded_body.txt", 
		serverURL, encodingName)
	require.NoError(t, os.WriteFile(httpFilePath, []byte(httpFileContent), 0644),
		"Failed to write .http file")
	return httpFilePath
}

// executeAndVerify executes the test and verifies the results
func executeAndVerify(t *testing.T, httpFilePath, expectedUTF8Body string, bodyReceived chan []byte) {
	t.Helper()
	client, err := rc.NewClient()
	require.NoError(t, err)
	responses, err := client.ExecuteFile(context.Background(), httpFilePath)
	require.NoError(t, err, "ExecuteFile failed")
	require.Len(t, responses, 1, "Expected one response")
	assert.Equal(t, http.StatusOK, responses[0].StatusCode, "Expected status OK")

	select {
	case received := <-bodyReceived:
		assert.Equal(t, expectedUTF8Body, string(received), "Mismatch in body received by server")
	case <-time.After(2 * time.Second): // Timeout for receiving body
		t.Fatal("Timeout waiting for mock server to receive body")
	}
}

// PRD-COMMENT: FR4.3 - Request Body: External File with Encoding (<@|encoding)
// Corresponds to: Client's ability to process request bodies from external files with a 
// specified character encoding using '<@|encoding filepath' (http_syntax.md "Request Body", 
// "External File with Encoding (<@|encoding filepath)").
// This test verifies that an external file ('test_encoded.txt') with a specific encoding 
// (e.g., latin1) is correctly read and used as the request body.
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

// PRD-COMMENT: FR4.5 / FR4.6 - Request Body: External File with Variables and Encoding (<@encoding)
// Corresponds to: Client's ability to process request bodies from external files with 
// specified character encoding and variable substitution.
// This test verifies that variables are substituted into an encoded external file, and 
// the resulting content is correctly sent to the server.
func TestExecuteFile_ExternalFileWithVariablesAndEncoding(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Define content with variables
	contentWithVars := "name={{name}}, city={{city}}, id={{id}}"
	expectedSubstitutedUTF8 := "name=TestName, city=ProgrammaticCity, id=12345"

	// Encode content to latin1
	latin1Encoder := charmap.ISO8859_1.NewEncoder()
	encodedBytes, _, err := transform.Bytes(latin1Encoder, []byte(contentWithVars))
	require.NoError(t, err, "Failed to encode content to latin1")

	varsFile := filepath.Join(tempDir, "vars_latin1.txt")
	err = os.WriteFile(varsFile, encodedBytes, 0644)
	require.NoError(t, err)

	// Channel to receive the body read by the server
	bodyReceived := make(chan []byte, 1)

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "text/plain; charset=iso-8859-1", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		bodyReceived <- body // Send raw bytes for later decoding

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Create a test HTTP file
	httpContent := fmt.Sprintf(`@name = TestName
@id = 12345

### External File with Variable Substitution and Encoding
POST %s/post
Content-Type: text/plain; charset=iso-8859-1

<@latin1 ./vars_latin1.txt`, server.URL)

	httpFile := filepath.Join(tempDir, "test_vars_encoded.http")
	err = os.WriteFile(httpFile, []byte(httpContent), 0644)
	require.NoError(t, err)

	// Create client with additional programmatic variables
	client, err := rc.NewClient(rc.WithVars(map[string]any{
		"city": "ProgrammaticCity", // This should be used
	}))
	require.NoError(t, err)

	// Execute the file
	responses, err := client.ExecuteFile(context.Background(), httpFile)
	require.NoError(t, err)
	require.Len(t, responses, 1)

	response := responses[0]
	assert.NoError(t, response.Error)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	// Check that the client-side RawBody (which should be UTF-8 after substitution) is correct
	assert.Equal(t, expectedSubstitutedUTF8, response.Request.RawBody, "Client RawBody mismatch")

	// Check body received by server
	select {
	case receivedBytes := <-bodyReceived:
		// Decode received bytes from latin1 to UTF-8
		latin1Decoder := charmap.ISO8859_1.NewDecoder()
		decodedBytes, _, decErr := transform.Bytes(latin1Decoder, receivedBytes)
		require.NoError(t, decErr, "Failed to decode server received body from latin1")
		assert.Equal(t, expectedSubstitutedUTF8, string(decodedBytes), "Server received body mismatch after decoding")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for mock server to receive body")
	}
}

// PRD-COMMENT: FR1.1 - File Type: .rest extension support
// Corresponds to: Client's ability to parse and execute request files with the .rest 
// extension, as an alternative to .http (http_syntax.md "File Structure", "File Extension").
// This test verifies that a simple GET request defined in a .rest file is correctly executed.
func TestExecuteFile_WithRestExtension(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Channel to confirm server received the request
	requestReceived := make(chan bool, 1)

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/get_test_rest_extension", r.URL.Path)
		assert.Equal(t, "rest-extension-test-value", r.Header.Get("X-Test-Header-Rest"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok from .rest"}`))
		requestReceived <- true
	}))
	defer server.Close()

	// Create a test .rest file
	restContent := fmt.Sprintf(`### Test Request with .rest extension
GET %s/get_test_rest_extension
X-Test-Header-Rest: rest-extension-test-value
`, server.URL)

	restFile := filepath.Join(tempDir, "test_request.rest")
	err := os.WriteFile(restFile, []byte(restContent), 0644)
	require.NoError(t, err)

	// Create client
	client, err := rc.NewClient()
	require.NoError(t, err)

	// Execute the file
	responses, err := client.ExecuteFile(context.Background(), restFile)
	require.NoError(t, err)
	require.Len(t, responses, 1)

	response := responses[0]
	assert.NoError(t, response.Error)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Contains(t, string(response.Body), `{"status": "ok from .rest"}`)

	// Verify server received the request
	select {
	case <-requestReceived:
		// All good
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for mock server to receive request")
	}
}

// PRD-COMMENT: FR4.4 - Request Body: External File Not Found
// Corresponds to: Client error handling when an external file referenced in a request 
// body (e.g., via '<@ ./nonexistent.json') cannot be found (http_syntax.md "Request Body").
// This test verifies that the client reports an appropriate error when attempting to 
// process a request that references a non-existent external file.
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
