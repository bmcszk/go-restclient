package restclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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

	// Use different delimiters to avoid conflict with {{...}} used by the library itself
	tmpl, err := template.New("testfile").Delims("[[", "]]").Parse(string(tmplContent))
	require.NoError(t, err)

	tempFile, err := os.CreateTemp(t.TempDir(), "processed_*.http")
	require.NoError(t, err)

	err = tmpl.Execute(tempFile, data)
	require.NoError(t, err)

	err = tempFile.Close()
	require.NoError(t, err)

	return tempFile.Name()
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
