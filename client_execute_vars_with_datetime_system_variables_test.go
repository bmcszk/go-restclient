package restclient

import (
	"context"
	"encoding/json" // Kept as not flagged by linter for this file
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_WithDateTimeSystemVariables(t *testing.T) {
	// Given
	var interceptedRequest struct {
		Headers map[string]string
		Body    string
	}
	interceptedRequest.Headers = make(map[string]string)

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		for name, values := range r.Header {
			if len(values) > 0 {
				interceptedRequest.Headers[name] = values[0]
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t,
		"testdata/http_request_files/system_var_datetime.http",
		struct{ ServerURL string }{ServerURL: server.URL},
	)

	// Log the content of the generated temporary file for debugging
	tempFileContent, errRead := os.ReadFile(requestFilePath)
	require.NoError(t, errRead, "Failed to read temporary file for debugging: %s", requestFilePath)
	t.Logf("[DEBUG_TEST] Content of temporary file '%s':\n%s", requestFilePath, string(tempFileContent))

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for datetime processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	now := time.Now()
	threshold := 5 * time.Second // Allow 5s difference for timestamp checks

	// Helper to check datetime strings
	checkDateTimeStr := func(t *testing.T, valueStr string, formatKeyword string, isUTC bool, headerName string) {
		t.Helper()
		if formatKeyword == "timestamp" {
			ts, err := strconv.ParseInt(valueStr, 10, 64)
			require.NoError(t, err, "Failed to parse timestamp from %s: %s", headerName, valueStr)
			parsedTime := time.Unix(ts, 0)
			assert.WithinDuration(t, now, parsedTime, threshold, "%s timestamp %s not within threshold of current time %s", headerName, parsedTime, now)
		} else {
			var layout string
			switch formatKeyword {
			case "rfc1123":
				layout = time.RFC1123
			case "iso8601":
				layout = time.RFC3339 // Go's RFC3339 is ISO8601 compliant
			default:
				t.Fatalf("Unhandled format keyword: %s for %s", formatKeyword, headerName)
			}
			parsedTime, err := time.Parse(layout, valueStr)
			require.NoError(t, err, "Failed to parse datetime string from %s ('%s') with layout '%s'", headerName, valueStr, layout)
			assert.WithinDuration(t, now, parsedTime, threshold, "%s datetime %s not within threshold of current time %s", headerName, parsedTime, now)
			if isUTC {
				assert.Equal(t, time.UTC, parsedTime.Location(), "%s expected to be UTC", headerName)
			} else {
				// Get offset for time.Local
				_, localOffset := now.In(time.Local).Zone()
				// Get offset for parsedTime
				_, parsedOffset := parsedTime.Zone()
				assert.Equal(t, localOffset, parsedOffset, "%s expected to have local time offset, got %d, want %d", headerName, parsedOffset, localOffset)
			}
		}
	}

	// Check Headers
	checkDateTimeStr(t, interceptedRequest.Headers["X-Datetime-Rfc1123"], "rfc1123", true, "X-Datetime-RFC1123")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Datetime-Iso8601"], "iso8601", true, "X-Datetime-ISO8601")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Datetime-Timestamp"], "timestamp", true, "X-Datetime-Timestamp")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Datetime-Default"], "iso8601", true, "X-Datetime-Default (ISO8601)")

	checkDateTimeStr(t, interceptedRequest.Headers["X-Localdatetime-Rfc1123"], "rfc1123", false, "X-LocalDatetime-RFC1123")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Localdatetime-Iso8601"], "iso8601", false, "X-LocalDatetime-ISO8601")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Localdatetime-Timestamp"], "timestamp", false, "X-LocalDatetime-Timestamp")
	checkDateTimeStr(t, interceptedRequest.Headers["X-Localdatetime-Default"], "iso8601", false, "X-LocalDatetime-Default (ISO8601)")

	assert.Equal(t, "{{$datetime \"invalidFormat\"}}", interceptedRequest.Headers["X-Datetime-Invalid"], "X-Datetime-Invalid should remain unresolved")

	// Check Body
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal request body JSON")

	checkDateTimeStr(t, bodyJSON["utc_rfc1123"], "rfc1123", true, "body.utc_rfc1123")
	checkDateTimeStr(t, bodyJSON["utc_iso8601"], "iso8601", true, "body.utc_iso8601")
	checkDateTimeStr(t, bodyJSON["utc_timestamp"], "timestamp", true, "body.utc_timestamp")
	checkDateTimeStr(t, bodyJSON["utc_default_iso"], "iso8601", true, "body.utc_default_iso (ISO8601)")
	checkDateTimeStr(t, bodyJSON["local_rfc1123"], "rfc1123", false, "body.local_rfc1123")
	checkDateTimeStr(t, bodyJSON["local_iso8601"], "iso8601", false, "body.local_iso8601")
	checkDateTimeStr(t, bodyJSON["local_timestamp"], "timestamp", false, "body.local_timestamp")
	checkDateTimeStr(t, bodyJSON["local_default_iso"], "iso8601", false, "body.local_default_iso (ISO8601)")

	assert.Equal(t, "{{$datetime \"invalidFormat\"}}", bodyJSON["invalid_format"], "body.invalid_format should remain unresolved")
}
