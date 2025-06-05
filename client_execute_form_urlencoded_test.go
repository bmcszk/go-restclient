package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTempHTTPFileFromStringForFormTest creates a temporary .http file with the given content.
// It returns the path to the file and registers a cleanup function to remove the temp directory.
func createTempHTTPFileFromStringForFormTest(t *testing.T, content string) string {
	t.Helper()
	// Use a more specific prefix for temp dirs from this test file
	tempDir, err := os.MkdirTemp("", "test-form-urlencoded-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	filePath := filepath.Join(tempDir, "test.http")
	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

// runFormUrlencodedSubTest executes a single sub-test for TestExecuteFile_XWwwFormUrlencoded.
func runFormUrlencodedSubTest(t *testing.T, tc struct {
	name                    string
	httpFileContent         string
	varsToSet               map[string]string
	expectedRawBodyOnServer string
	expectExecuteError      bool
	requestAsserterFunc     func(t *testing.T, r *http.Request, expectedBody string)
}) {
	t.Helper()
	var capturedRequestBody string
	var capturedRequestContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err, "Failed to read request body on server")
		capturedRequestBody = string(bodyBytes)
		capturedRequestContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	hostVar := server.URL
	finalHttpFileContent := strings.ReplaceAll(tc.httpFileContent, "{{host}}", hostVar)
	// Substitute other vars if present in tc.varsToSet
	for key, val := range tc.varsToSet {
		placeholder := fmt.Sprintf("{{%s}}", key)
		finalHttpFileContent = strings.ReplaceAll(finalHttpFileContent, placeholder, val)
	}

	requestFilePath := createTempHTTPFileFromStringForFormTest(t, finalHttpFileContent)

	currentClient, clientErr := NewClient()
	require.NoError(t, clientErr)

	responses, execErr := currentClient.ExecuteFile(context.Background(), requestFilePath)

	if tc.expectExecuteError {
		require.Error(t, execErr)
		return
	}
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Should have one result")
	require.Nil(t, responses[0].Error, "Request execution error should be nil")

	assert.Equal(t, tc.expectedRawBodyOnServer, capturedRequestBody, "Captured request body does not match expected")

	// Assert Content-Type if the original request specified it and it's form-urlencoded
	if strings.Contains(tc.httpFileContent, "Content-Type: application/x-www-form-urlencoded") {
		assert.Equal(t, "application/x-www-form-urlencoded", capturedRequestContentType, "Content-Type header mismatch for form-urlencoded")
	} else if strings.Contains(tc.httpFileContent, "Content-Type: application/json") {
		assert.Equal(t, "application/json", capturedRequestContentType, "Content-Type header mismatch for json")
	}
	// If no Content-Type was in httpFileContent, Go's client might set one by default or leave it empty, specific assertion might be needed if strict about its absence.
}

// TestExecuteFile_XWwwFormUrlencoded tests the handling of application/x-www-form-urlencoded request bodies.
func TestExecuteFile_XWwwFormUrlencoded(t *testing.T) {
	tests := []struct {
		name                    string
		httpFileContent         string
		varsToSet               map[string]string // For {{variable}} substitution in HTTP file content
		expectedRawBodyOnServer string
		expectExecuteError      bool
		requestAsserterFunc     func(t *testing.T, r *http.Request, expectedBody string)
	}{
		{
			name: "correct_encoding_with_special_chars",
			httpFileContent: `
POST {{host}}/submit
Content-Type: application/x-www-form-urlencoded

key1=value with spaces&key2=value+plus&key3=value/slash&key4=value=equals&key5=value&ampersand&key6=value%percent&key7=你好世界
`,
			expectedRawBodyOnServer: "ampersand=&key1=value+with+spaces&key2=value%2Bplus&key3=value%2Fslash&key4=value%3Dequals&key5=value&key6=value%25percent&key7=%E4%BD%A0%E5%A5%BD%E4%B8%96%E7%95%8C",
		},
		{
			name: "variable_substitution_before_encoding",
			httpFileContent: `
@my_value = value with spaces & special chars like +/=%& and unicode 世界

POST {{host}}/submit
Content-Type: application/x-www-form-urlencoded

param1={{my_value}}&param2=static value
`,
			expectedRawBodyOnServer: "+and+unicode+%E4%B8%96%E7%95%8C=&+special+chars+like+%2B%2F=%25&param1=value+with+spaces+&param2=static+value", // Keys are sorted alphabetically by Encode
		},
		{
			name: "malformed_looking_body_is_correctly_reencoded",
			httpFileContent: `
POST {{host}}/submit
Content-Type: application/x-www-form-urlencoded

key1=value1=invalid&key2=value2
`,
			expectedRawBodyOnServer: "key1=value1%3Dinvalid&key2=value2",
		},
		{
			name: "other_content_type_not_affected_json",
			httpFileContent: `
POST {{host}}/submit
Content-Type: application/json

{"key1": "value with spaces & special chars like +/=%& and unicode 世界"}
`,
			expectedRawBodyOnServer: `{"key1": "value with spaces & special chars like +/=%& and unicode 世界"}`,
		},
		{
			name: "no_content_type_not_affected",
			httpFileContent: `
POST {{host}}/submit

key1=value with spaces&key2=value+plus
`,
			expectedRawBodyOnServer: "key1=value with spaces&key2=value+plus",
		},
		{
			name: "empty_body_with_form_urlencoded_type",
			httpFileContent: `
POST {{host}}/submit
Content-Type: application/x-www-form-urlencoded

`,
			expectedRawBodyOnServer: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runFormUrlencodedSubTest(t, tc)
		})
	}
}
