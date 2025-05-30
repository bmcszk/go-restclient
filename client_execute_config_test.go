package restclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_WithBaseURL(t *testing.T) {
	// Given
	var interceptedReq *http.Request
	mockTransport := &mockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			interceptedReq = req.Clone(req.Context())
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("mock"))}, nil
		},
	}

	mockServerURL := "http://localhost:12345" // Dummy URL, won't be hit
	client, err := NewClient(
		WithBaseURL(mockServerURL+"/api"),
		WithHTTPClient(&http.Client{Transport: mockTransport}),
	)
	require.NoError(t, err)
	requestFilePath := "testdata/http_request_files/relative_path_get.http"

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	assert.NoError(t, responses[0].Error)

	require.NotNil(t, interceptedReq)
	assert.Equal(t, mockServerURL, interceptedReq.URL.Scheme+"://"+interceptedReq.URL.Host)
	assert.Equal(t, "/api/todos/1", interceptedReq.URL.Path)
}

func TestExecuteFile_WithDefaultHeaders(t *testing.T) {
	// Given
	var interceptedReq *http.Request
	mockTransport := &mockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			interceptedReq = req.Clone(req.Context())
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("mock"))}, nil
		},
	}

	client, err := NewClient(
		WithDefaultHeader("X-Default", "default-value"),
		WithDefaultHeader("X-Override", "default-should-be-overridden"),
		WithHTTPClient(&http.Client{Transport: mockTransport}),
		WithBaseURL("http://dummyserver.com"), // Base URL needed for relative path in .http file
	)
	require.NoError(t, err)
	requestFilePath := "testdata/http_request_files/get_with_override_header.http"

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	assert.NoError(t, responses[0].Error)

	require.NotNil(t, interceptedReq)
	assert.Equal(t, "default-value", interceptedReq.Header.Get("X-Default"))
	assert.Equal(t, "file-value", interceptedReq.Header.Get("X-Override"), "Header from file should override client default")
	assert.Equal(t, "present", interceptedReq.Header.Get("X-File-Only"))
}
