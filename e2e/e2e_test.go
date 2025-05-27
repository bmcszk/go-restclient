//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmcszk/go-restclient" // Import the library itself
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a temporary .rest file for testing
func createTempRestFile(t *testing.T, content string) string {
	t.Helper()
	tempDir := t.TempDir() // Creates a temporary directory that is cleaned up automatically
	tempFile, err := os.Create(filepath.Join(tempDir, "testcase.rest"))
	require.NoError(t, err, "Failed to create temp .rest file")

	_, err = tempFile.WriteString(content)
	require.NoError(t, err, "Failed to write to temp .rest file")
	err = tempFile.Close()
	require.NoError(t, err, "Failed to close temp .rest file")

	return tempFile.Name()
}

func TestE2E_SimpleGetRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/test/get" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-E2E-Test", "active")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"message": "E2E GET success"}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	restFileContent := fmt.Sprintf("GET %s/test/get\nAccept: application/json", server.URL)
	restFilePath := createTempRestFile(t, restFileContent)

	client, err := restclient.NewClient()
	require.NoError(t, err, "Failed to create restclient.Client")

	responses, err := client.ExecuteFile(restFilePath)
	require.NoError(t, err, "client.ExecuteFile returned an error")
	require.Len(t, responses, 1, "Expected one response from ExecuteFile")

	resp := responses[0]
	require.NotNil(t, resp, "Response object should not be nil")
	require.NoError(t, resp.Error, "Response.Error should be nil for a successful request")

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")
	assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"), "Content-Type header mismatch")
	assert.Equal(t, "active", resp.Headers.Get("X-E2E-Test"), "X-E2E-Test header mismatch")
	assert.Contains(t, resp.BodyString, "E2E GET success", "Response body mismatch")

	// Test request details are populated
	require.NotNil(t, resp.Request, "Response.Request should be populated")
	assert.Equal(t, http.MethodGet, resp.Request.Method)
	assert.Equal(t, server.URL+"/test/get", resp.Request.URL.String())
}

func TestE2E_MultipleRequestsInFile(t *testing.T) {
	var getCounter, postCounter int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/multi/get" {
			getCounter++
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "GET response")
		} else if r.Method == http.MethodPost && r.URL.Path == "/multi/post" {
			postCounter++
			bodyBytes, _ := io.ReadAll(r.Body)
			assert.Equal(t, "post_body_content", string(bodyBytes))
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, "POST response")
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	restFileContent := fmt.Sprintf(
		"GET %s/multi/get\n\n### POST Request\nPOST %s/multi/post\nContent-Type: text/plain\n\npost_body_content",
		server.URL, server.URL,
	)
	restFilePath := createTempRestFile(t, restFileContent)

	client, _ := restclient.NewClient()
	responses, err := client.ExecuteFile(restFilePath)

	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Equal(t, 1, getCounter, "GET request not handled or handled multiple times")
	assert.Equal(t, 1, postCounter, "POST request not handled or handled multiple times")

	// Check GET response
	respGet := responses[0]
	require.NoError(t, respGet.Error)
	assert.Equal(t, http.StatusOK, respGet.StatusCode)
	assert.Equal(t, "GET response", respGet.BodyString)
	assert.Equal(t, "GET", respGet.Request.Method)

	// Check POST response
	respPost := responses[1]
	require.NoError(t, respPost.Error)
	assert.Equal(t, http.StatusCreated, respPost.StatusCode)
	assert.Equal(t, "POST response", respPost.BodyString)
	assert.Equal(t, "POST", respPost.Request.Method)
	assert.Equal(t, "post_body_content", respPost.Request.RawBody)
}

// TODO: Add E2E test for request resulting in error (e.g., connection refused)
// TODO: Add E2E test for TLS connection (using httptest.NewTLSServer) and verify resp.IsTLS, resp.TLSVersion etc.
// TODO: Add E2E test that uses an expected response file and restclient.ValidateResponse
