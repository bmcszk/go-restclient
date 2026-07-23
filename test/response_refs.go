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

func RunResponseRef_StatusCode(t *testing.T) {
	t.Helper()
	// Given
	server := startMockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"id": 42}`)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		"### create\n"+
			"POST "+server.URL+"/items\n"+
			"Content-Type: application/json\n\n"+
			`{"name":"test"}`+"\n\n"+
			"###\n"+
			"### check status\n"+
			"GET "+server.URL+"/status/{{create.response.status}}\n")

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Equal(t, 201, responses[0].StatusCode)
	assert.Contains(t, responses[1].Request.URL.String(), "/status/201")
}

func RunResponseRef_BodyField(t *testing.T) {
	t.Helper()
	// Given
	var paths []string
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/login" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"access_token":"abc123","token_type":"Bearer"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "protected data")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		"### login\n"+
			"POST "+server.URL+"/login\n"+
			"Content-Type: application/json\n\n"+
			`{"user":"admin","pass":"secret"}`+"\n\n"+
			"###\n"+
			"### access protected\n"+
			"GET "+server.URL+"/protected\n"+
			"Authorization: Bearer {{login.response.body.access_token}}\n")

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Equal(t, []string{"/login", "/protected"}, paths)
	assert.Equal(t, "Bearer abc123", responses[1].Request.Headers.Get("Authorization"))
}

func RunResponseRef_HeaderValue(t *testing.T) {
	t.Helper()
	// Given
	server := startMockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Session-Id", "sess-999")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		"### first\n"+
			"GET "+server.URL+"/init\n\n"+
			"###\n"+
			"### second\n"+
			"GET "+server.URL+"/use\n"+
			"X-Session: {{first.response.headers.X-Session-Id}}\n")

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Equal(t, "sess-999", responses[1].Request.Headers.Get("X-Session"))
}

func RunResponseRef_NestedBodyField(t *testing.T) {
	t.Helper()
	// Given
	server := startMockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"user":{"name":"Alice","id":123}}}`)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		"### get user\n"+
			"GET "+server.URL+"/user\n\n"+
			"###\n"+
			"### use name\n"+
			"GET "+server.URL+"/greet/{{get user.response.body.data.user.name}}\n")

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Contains(t, responses[1].Request.URL.String(), "/greet/Alice")
}

func RunResponseRef_ArrayIndex(t *testing.T) {
	t.Helper()
	// Given
	server := startMockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"items":["alpha","beta","gamma"]}`)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		"### list\n"+
			"GET "+server.URL+"/list\n\n"+
			"###\n"+
			"### get second\n"+
			"GET "+server.URL+"/item/{{list.response.body.items[1]}}\n")

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Contains(t, responses[1].Request.URL.String(), "/item/beta")
}

func RunResponseRef_MissingName(t *testing.T) {
	t.Helper()
	// Given
	server := startMockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		"### actual\n"+
			"GET "+server.URL+"/ok\n\n"+
			"###\n"+
			"### ref non-existent\n"+
			"GET "+server.URL+"/ref/{{nonexistent.response.status}}\n")

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then — should not error, just resolve to empty
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Contains(t, responses[1].Request.URL.String(), "/ref/")
}

func RunResponseRef_BodyWhole(t *testing.T) {
	t.Helper()
	// Given
	server := startMockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "raw-body-content")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	filePath := createTestFileFromString(t,
		"### get\n"+
			"GET "+server.URL+"/data\n\n"+
			"###\n"+
			"### echo\n"+
			"POST "+server.URL+"/echo\n"+
			"Content-Type: text/plain\n\n"+
			"{{get.response.body}}\n")

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Equal(t, "raw-body-content", string(responses[1].Request.RawBody))
}
