package test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	rc "github.com/bmcszk/go-restclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RunParseFile_SingleRequest(t *testing.T) {
	t.Helper()
	// Given
	server := startMockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		fmt.Sprintf("GET %s/health\n", server.URL))

	// When
	parsedFile, err := client.ParseFile(filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 1)
	assert.Equal(t, http.MethodGet, parsedFile.Requests[0].Method)
	assert.Equal(t, server.URL+"/health", parsedFile.Requests[0].RawURLString)
}

func RunParseFile_MultipleRequests(t *testing.T) {
	t.Helper()
	// Given
	filePath := createTestFileFromString(t,
		"GET http://example.com/a\n###\nPOST http://example.com/b\nContent-Type: application/json\n\n{\"x\":1}\n")

	client, _ := rc.NewClient()

	// When
	parsedFile, err := client.ParseFile(filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 2)
	assert.Equal(t, http.MethodGet, parsedFile.Requests[0].Method)
	assert.Equal(t, http.MethodPost, parsedFile.Requests[1].Method)
}

func RunParseFile_NamedRequests(t *testing.T) {
	t.Helper()
	// Given
	filePath := createTestFileFromString(t,
		"### get users\nGET http://example.com/users\n###\n### create user\nPOST http://example.com/users\n")

	client, _ := rc.NewClient()

	// When
	parsedFile, err := client.ParseFile(filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, parsedFile.Requests, 2)
	assert.Equal(t, "get users", parsedFile.Requests[0].Name)
	assert.Equal(t, "create user", parsedFile.Requests[1].Name)
}

func RunParseFile_NoRequests(t *testing.T) {
	t.Helper()
	// Given
	filePath := createTestFileFromString(t, "# just a comment\n")
	client, _ := rc.NewClient()

	// When
	_, err := client.ParseFile(filePath)

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no requests found")
}

func RunParseFile_FileNotFound(t *testing.T) {
	t.Helper()
	// Given
	client, _ := rc.NewClient()

	// When
	_, err := client.ParseFile("/nonexistent/file.http")

	// Then
	require.Error(t, err)
}

func RunExecuteRequest_ByIndex(t *testing.T) {
	t.Helper()
	// Given
	var gotPath string
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "response1")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		fmt.Sprintf("GET %s/a\n###\nGET %s/b\n", server.URL, server.URL))

	parsedFile, err := client.ParseFile(filePath)
	require.NoError(t, err)

	// When — execute first request only
	resp, execErr := client.ExecuteRequest(context.Background(), parsedFile, 0)

	// Then
	require.NoError(t, execErr)
	assert.NoError(t, resp.Error)
	assert.Equal(t, "/a", gotPath)
	assert.Equal(t, "response1", resp.BodyString)
}

func RunExecuteRequest_OutOfRange(t *testing.T) {
	t.Helper()
	// Given
	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t, "GET http://example.com/a\n")
	parsedFile, err := client.ParseFile(filePath)
	require.NoError(t, err)

	// When
	_, execErr := client.ExecuteRequest(context.Background(), parsedFile, 5)

	// Then
	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "out of range")
}

func RunExecuteRequest_NegativeIndex(t *testing.T) {
	t.Helper()
	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t, "GET http://example.com/a\n")
	parsedFile, err := client.ParseFile(filePath)
	require.NoError(t, err)

	_, execErr := client.ExecuteRequest(context.Background(), parsedFile, -1)
	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "out of range")
}

func RunExecuteRequest_SecondOfThree(t *testing.T) {
	t.Helper()
	// Given
	var paths []string
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		fmt.Sprintf("GET %s/1\n###\nGET %s/2\n###\nGET %s/3\n", server.URL, server.URL, server.URL))

	parsedFile, err := client.ParseFile(filePath)
	require.NoError(t, err)

	// When — execute only the second request
	resp, execErr := client.ExecuteRequest(context.Background(), parsedFile, 1)

	// Then
	require.NoError(t, execErr)
	assert.NoError(t, resp.Error)
	assert.Equal(t, []string{"/2"}, paths, "only second request should execute")
}
