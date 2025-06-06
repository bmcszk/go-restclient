package restclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require" // require is used in createTestFileFromTemplate
)

// Helper to create a mock server
func startMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// createTestFileFromTemplate processes a template file and returns the path to the processed file.
func createTestFileFromTemplate(t *testing.T, templatePath string, data interface{}) string {
	t.Helper()
	tmplContent, err := os.ReadFile(templatePath)
	require.NoError(t, err)

	t.Logf("[DEBUG_CREATE_TEST_FILE] Original template content for %s:\n%s", templatePath, string(tmplContent))

	// Use different delimiters to avoid conflict with {{...}} used by the library itself
	tmpl, err := template.New("testfile").Delims("[[", "]]").Parse(string(tmplContent))
	require.NoError(t, err)

	tempFile, err := os.CreateTemp(t.TempDir(), "processed_*.http")
	require.NoError(t, err)
	tempFileName := tempFile.Name() // Get name before closing for reliable access

	err = tmpl.Execute(tempFile, data)
	require.NoError(t, err)

	err = tempFile.Close()
	require.NoError(t, err)

	// Read back the content of the temporary file to log what was actually written
	writtenContent, readErr := os.ReadFile(tempFileName)
	require.NoError(t, readErr, "Failed to read back temp file for debugging: %s", tempFileName)
	t.Logf("[DEBUG_CREATE_TEST_FILE] Content written to temp file %s:\n%s", tempFileName, string(writtenContent))

	return tempFileName
}

// mockRoundTripper is a helper for mocking http.RoundTripper
type mockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.RoundTripFunc != nil {
		return m.RoundTripFunc(req)
	}
	return nil, fmt.Errorf("RoundTripFunc not set")
}

// ptr is a helper function to get a pointer to a string.
func ptr(s string) *string {
	return &s
}

// parseHrespBody reads an .hresp file and parses its content to separate
// headers and body. It returns the parsed headers as http.Header and the body as a string.
// The .hresp format expects headers first, then a blank line, then the body.
func parseHrespBody(filePath string) (http.Header, string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read .hresp file %s: %w", filePath, err)
	}

	parts := strings.SplitN(string(content), "\n\n", 2)
	headers := make(http.Header)
	bodyStr := ""

	headerLines := strings.Split(strings.TrimSpace(parts[0]), "\n")
	for _, line := range headerLines {
		if strings.TrimSpace(line) == "" || !strings.Contains(line, ":") {
			// Skip empty lines or lines not containing a colon (likely the HTTP version line or status)
			continue
		}
		headerParts := strings.SplitN(line, ":", 2)
		if len(headerParts) == 2 {
			headers.Add(strings.TrimSpace(headerParts[0]), strings.TrimSpace(headerParts[1]))
		}
	}

	if len(parts) == 2 {
		bodyStr = strings.TrimSpace(parts[1])
	}

	return headers, bodyStr, nil
}
