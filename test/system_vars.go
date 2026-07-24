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

func RunSystemVar_Base64Encode(t *testing.T) {
	t.Helper()
	// Given
	var authHeader string
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		"### auth\n"+
			"GET "+server.URL+"/api\n"+
			"Authorization: Basic {{$base64encode user:pass}}\n")

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	assert.Equal(t, "Basic dXNlcjpwYXNz", authHeader)
}

func RunSystemVar_Base64EncodeInBody(t *testing.T) {
	t.Helper()
	// Given
	var body string
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		body = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		"### send\n"+
			"POST "+server.URL+"/data\n"+
			"Content-Type: text/plain\n\n"+
			"{{$base64encode hello world}}\n")

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	assert.Equal(t, "aGVsbG8gd29ybGQ=", body)
}
