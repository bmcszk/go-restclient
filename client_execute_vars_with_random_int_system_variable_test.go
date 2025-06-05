package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runRandomIntSystemVariableSubTest executes a single sub-test for TestExecuteFile_WithRandomIntSystemVariable.
func runRandomIntSystemVariableSubTest(t *testing.T, client *Client, serverURL string, interceptedRequest *struct {
	URL    string
	Header string
	Body   string
}, tc struct {
	name               string
	httpFilePath       string
	validate           func(t *testing.T, url, header, body string)
	expectErrorInParse bool
}) {
	t.Helper()
	// Given specific setup for this subtest
	requestFilePath := createTestFileFromTemplate(t, tc.httpFilePath, struct{ ServerURL string }{ServerURL: serverURL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	if tc.expectErrorInParse {
		require.Error(t, err, "Expected an error during ExecuteFile for %s", tc.name)
		return
	}
	require.NoError(t, err, "ExecuteFile should not return an error for %s", tc.name)
	require.Len(t, responses, 1, "Expected 1 response for %s", tc.name)
	resp := responses[0]
	assert.NoError(t, resp.Error, "Response error should be nil for %s", tc.name)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status OK for %s", tc.name)

	actualURL := interceptedRequest.URL
	if strings.Contains(actualURL, "%") {
		decodedURL, decodeErr := url.PathUnescape(actualURL)
		if decodeErr == nil {
			actualURL = decodedURL
		}
	}
	tc.validate(t, actualURL, interceptedRequest.Header, interceptedRequest.Body)
}

func TestExecuteFile_WithRandomIntSystemVariable(t *testing.T) {
	// Given common setup for all subtests
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Random-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()
	client, _ := NewClient()

	tests := []struct {
		name               string
		httpFilePath       string
		validate           func(t *testing.T, url, header, body string)
		expectErrorInParse bool
	}{
		{ // SCENARIO-LIB-015-001
			name:         "valid min max args",
			httpFilePath: "testdata/http_request_files/system_var_randomint_valid_args.http",
			validate: func(t *testing.T, url, header, body string) {
				urlParts := strings.Split(url, "/")
				valURL, err := strconv.Atoi(urlParts[len(urlParts)-2])
				require.NoError(t, err, "Random int from URL should be valid int")
				assert.True(t, valURL >= 10 && valURL <= 20, "URL random int %d out of range [10,20]", valURL)

				valHeader, err := strconv.Atoi(header)
				require.NoError(t, err, "Random int from Header should be valid int")
				assert.True(t, valHeader >= 1 && valHeader <= 5, "Header random int %d out of range [1,5]", valHeader)

				var bodyJSON map[string]int
				err = json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body")
				assert.True(t, bodyJSON["value"] >= 100 && bodyJSON["value"] <= 105, "Body random int %d out of range [100,105]", bodyJSON["value"])
			},
		},
		{ // SCENARIO-LIB-015-002
			name:         "no args",
			httpFilePath: "testdata/http_request_files/system_var_randomint_no_args.http",
			validate: func(t *testing.T, url, header, body string) {
				urlParts := strings.Split(url, "/")
				valURL, err := strconv.Atoi(urlParts[len(urlParts)-2])
				require.NoError(t, err, "Random int from URL (no args) should be valid int")
				assert.True(t, valURL >= 0 && valURL <= 1000, "URL random int (no args) %d out of range [0,1000]", valURL)

				valHeader, err := strconv.Atoi(header)
				require.NoError(t, err, "Random int from Header (no args) should be valid int")
				assert.True(t, valHeader >= 0 && valHeader <= 1000, "Header random int (no args) %d out of range [0,1000]", valHeader)

				var bodyJSON map[string]int
				err = json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body (no args)")
				assert.True(t, bodyJSON["value"] >= 0 && bodyJSON["value"] <= 1000, "Body random int (no args) %d out of range [0,1000]", bodyJSON["value"])
			},
		},
		{ // SCENARIO-LIB-015-003
			name:         "swapped min max args",
			httpFilePath: "testdata/http_request_files/system_var_randomint_swapped_args.http",
			validate: func(t *testing.T, url, header, body string) {
				urlParts := strings.Split(url, "/")
				require.Len(t, urlParts, 4, "URL path should have 4 parts for swapped args test")
				assert.Equal(t, "{{$randomInt 30 25}}", urlParts[2], "URL part1 for swapped_min_max_args should be the unresolved placeholder")
				assert.Equal(t, "{{$randomInt 30 25}}", urlParts[3], "URL part2 for swapped_min_max_args should be the unresolved placeholder")
				var bodyJSON map[string]string
				err := json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body (swapped)")
				assert.Equal(t, "{{$randomInt 30 25}}", bodyJSON["value"], "Body for swapped_min_max_args should be the unresolved placeholder")
			},
		},
		{ // SCENARIO-LIB-015-004
			name:         "malformed args",
			httpFilePath: "testdata/http_request_files/system_var_randomint_malformed_args.http",
			validate: func(t *testing.T, urlStr, header, body string) {
				expectedLiteralPlaceholder := "{{$randomInt abc def}}"
				assert.Contains(t, urlStr, expectedLiteralPlaceholder, "URL should contain literal malformed $randomInt")
				assert.Equal(t, "{{$randomInt 1 xyz}}", header, "Header should retain malformed $randomInt")
				var bodyJSON map[string]string
				err := json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body (malformed)")
				assert.Equal(t, "{{$randomInt foo bar}}", bodyJSON["value"], "Body should retain malformed $randomInt")
			},
			expectErrorInParse: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runRandomIntSystemVariableSubTest(t, client, server.URL, &interceptedRequest, tc)
		})
	}
}
