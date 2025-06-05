package restclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_WithGuidSystemVariable(t *testing.T) {
	// Given
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Request-ID")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_guid.http", struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for GUID processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// SCENARIO-LIB-014-001: {{$guid}} in URL
	urlParts := strings.Split(interceptedRequest.URL, "/")
	require.True(t, len(urlParts) >= 2, "URL should have at least two parts after splitting by /")
	guidFromURL := urlParts[len(urlParts)-1]
	_, err = uuid.Parse(guidFromURL)
	assert.NoError(t, err, "GUID from URL should be a valid UUID: %s", guidFromURL)

	// SCENARIO-LIB-014-002: {{$guid}} in header
	guidFromHeader := interceptedRequest.Header
	_, err = uuid.Parse(guidFromHeader)
	assert.NoError(t, err, "GUID from X-Request-ID header should be a valid UUID: %s", guidFromHeader)

	// SCENARIO-LIB-014-003: {{$guid}} in body
	// SCENARIO-LIB-014-004: Multiple {{$guid}} in one request yield different GUIDs
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal response body JSON")

	guidFromBody1, ok1 := bodyJSON["transactionId"]
	require.True(t, ok1, "transactionId not found in body")
	_, err = uuid.Parse(guidFromBody1)
	assert.NoError(t, err, "GUID from body (transactionId) should be a valid UUID: %s", guidFromBody1)

	guidFromBody2, ok2 := bodyJSON["correlationId"]
	require.True(t, ok2, "correlationId not found in body")
	_, err = uuid.Parse(guidFromBody2)
	assert.NoError(t, err, "GUID from body (correlationId) should be a valid UUID: %s", guidFromBody2)

	guidFromRandomUuidAlias, ok3 := bodyJSON["randomUuidAlias"]
	require.True(t, ok3, "randomUuidAlias not found in body")
	_, err = uuid.Parse(guidFromRandomUuidAlias)
	assert.NoError(t, err, "GUID from body (randomUuidAlias) should be a valid UUID: %s", guidFromRandomUuidAlias)
	assert.Equal(t, guidFromURL, guidFromRandomUuidAlias, "GUID from URL and randomUuidAlias should be the same")

	// With request-scoped system variables, all {{$guid}} ({{$uuid}}) instances should resolve to the SAME value.
	assert.Equal(t, guidFromURL, guidFromHeader, "GUID from URL and header should be the same")
	assert.Equal(t, guidFromURL, guidFromBody1, "GUID from URL and body1 should be the same")
	// For this test, the .http file uses {{$guid}} twice in the body for different fields.
	// These should now resolve to the same request-scoped GUID.
	assert.Equal(t, guidFromBody1, guidFromBody2, "GUIDs from body (transactionId and correlationId) should be the same")
}
