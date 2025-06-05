package restclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_WithIsoTimestampSystemVariable(t *testing.T) {
	// Given
	var interceptedRequest struct {
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Timestamp-Header")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_iso_timestamp.http", struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for $isoTimestamp processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check header
	_, err = time.Parse(time.RFC3339Nano, interceptedRequest.Header)
	assert.NoError(t, err, "X-Timestamp-Header should be a valid ISO8601 timestamp: %s", interceptedRequest.Header)

	// Check body
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal response body JSON")

	isoTimeFromBody, ok := bodyJSON["requestTime"]
	require.True(t, ok, "requestTime not found in body")
	_, err = time.Parse(time.RFC3339Nano, isoTimeFromBody)
	assert.NoError(t, err, "Body requestTime should be a valid ISO8601 timestamp: %s", isoTimeFromBody)

	assert.Equal(t, interceptedRequest.Header, isoTimeFromBody, "ISO Timestamp from header and body should be the same")
}
