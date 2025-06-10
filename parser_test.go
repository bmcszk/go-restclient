//go:build unit

package restclient_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmcszk/go-restclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(h))
	os.Exit(m.Run())
}

// TODO: Refactor tests for request naming (@name directive), expected response parsing
// (separator comments, simple cases), and variable scoping.
// These tests previously used unexported parser functions (parseRequestFile, parseExpectedResponses).
// They need to be rewritten to use the public Client.ExecuteFile API,
// mock HTTP servers, and assertions on the returned Response or errors.
// Ensure coverage for:
// - FR1.3: Request Naming (# @name directive, ### Name, precedence, whitespace handling, mixed usage)
// - Expected response parsing:
//   - Separator comments affecting response block association.
//   - Basic structure: status line, headers, body.
//   - Different body types (JSON, text).
//   - Header parsing (single, multiple values).
// - FR2.4: Variable Scoping and Templating:
//   - Nested variable references.
//   - File-level vs. request-specific variable overrides.
//   - Restoration of file-level variables.
//   - Variable expansion in request bodies (JSON).

func TestParserExternalFileDirectives(t *testing.T) {
	type externalFileTestCase struct {
		name             string
		httpFileContent  string // Raw content for the .http file, {{mock_server_url}} will be replaced
		targetFileName   string // Name of the dummy file to create (e.g., "body.txt")
		expectedEncoding string
		expectedFilePath string
		expectedVars     bool
	}

	// Setup a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock response"))
	}))
	defer mockServer.Close()

	testCases := []externalFileTestCase{
		{
			name:             "Valid encoding specified",
			httpFileContent:  "GET {{mock_server_url}}/\n<@latin1 body.txt",
			targetFileName:   "body.txt",
			expectedEncoding: "latin1",
			expectedFilePath: "body.txt",
			expectedVars:     true,
		},
		{
			name:             "Valid encoding with spaces in path",
			httpFileContent:  "GET {{mock_server_url}}/\n<@utf-8 \"my body data.txt\"",
			targetFileName:   "my body data.txt",
			expectedEncoding: "utf-8",
			expectedFilePath: "my body data.txt",
			expectedVars:     true,
		},
		{
			name:             "No encoding, with variables",
			httpFileContent:  "GET {{mock_server_url}}/\n<@ body.txt",
			targetFileName:   "body.txt",
			expectedEncoding: "", // Defaults to empty, client will use UTF-8
			expectedFilePath: "body.txt",
			expectedVars:     true,
		},
		{
			name:             "No encoding, static file",
			httpFileContent:  "GET {{mock_server_url}}/\n< body.txt",
			targetFileName:   "body.txt",
			expectedEncoding: "",
			expectedFilePath: "body.txt",
			expectedVars:     false,
		},
		{
			name:             "Invalid encoding name treated as part of path",
			httpFileContent:  "GET {{mock_server_url}}/\n<@invalid-enc body.txt",
			targetFileName:   "body.txt", // Target file name for creation, not for assertion
			expectedEncoding: "",
			expectedFilePath: "invalid-enc body.txt", // Parser behavior based on isValidEncoding
			expectedVars:     true,
		},
		{
			name:             "Encoding specified but no path",
			httpFileContent:  "GET {{mock_server_url}}/\n<@utf-8",
			targetFileName:   "", // No target file to create
			expectedEncoding: "utf-8",
			expectedFilePath: "",
			expectedVars:     true,
		},
		{
			name:             "Encoding and path with extra spaces",
			httpFileContent:  "GET {{mock_server_url}}/\n<@  utf-8   spaced path.txt  ",
			targetFileName:   "spaced path.txt",
			expectedEncoding: "utf-8",
			expectedFilePath: "spaced path.txt",
			expectedVars:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "parser_test_*")
			require.NoError(t, err, "Failed to create temp dir")
			defer os.RemoveAll(tempDir)

			// Create dummy target file if one is specified for the test case
			if tc.targetFileName != "" {
				dummyFilePath := filepath.Join(tempDir, tc.targetFileName)
				require.NoError(t, os.WriteFile(dummyFilePath, []byte("dummy content"), 0644),
					"Failed to write dummy target file")
			}

			httpFilePath := filepath.Join(tempDir, "test_request.http")
			contentToWrite := strings.ReplaceAll(tc.httpFileContent, "{{mock_server_url}}", mockServer.URL)
			
			// Adjust expected file path to be relative to the temp .http file if it's not empty
			// The parser resolves it relative to the .http file's directory.
			// However, ExternalFilePath stores the path *as written* in the .http file.
			// The client then resolves it. For parser test, we check what's stored.

			require.NoError(t, os.WriteFile(httpFilePath, []byte(contentToWrite), 0644),
				"Failed to write .http file")

			client := restclient.NewClient()
			responses, err := client.ExecuteFile(httpFilePath) // Use ExecuteFile

			require.NoError(t, err, "ExecuteFile failed")
			require.NotNil(t, responses, "ExecuteFile returned nil responses")
			require.Len(t, responses, 1, "Expected one response from ExecuteFile")
			
			req := responses[0].Request
			require.NotNil(t, req, "Request object is nil")

			assert.Equal(t, tc.expectedEncoding, req.ExternalFileEncoding, "ExternalFileEncoding mismatch")
			assert.Equal(t, tc.expectedFilePath, req.ExternalFilePath, "ExternalFilePath mismatch")
			assert.Equal(t, tc.expectedVars, req.ExternalFileWithVariables, "ExternalFileWithVariables mismatch")
		})
	}
}
