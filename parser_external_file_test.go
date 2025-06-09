package restclient

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRequestFile_ExternalFileReferences(t *testing.T) {
	tests := []struct {
		name                         string
		fileContent                  string
		expectedExternalFilePath     string
		expectedExternalFileEncoding string
		expectedWithVariables        bool
		expectedRawBody              string
	}{
		{
			name: "Basic external file reference",
			fileContent: `POST https://example.com/api/test
Content-Type: application/json

< ./path/to/file.json`,
			expectedExternalFilePath:     "./path/to/file.json",
			expectedExternalFileEncoding: "",
			expectedWithVariables:        false,
			expectedRawBody:              "< ./path/to/file.json",
		},
		{
			name: "External file with variable substitution",
			fileContent: `POST https://example.com/api/test
Content-Type: application/json

<@ ./path/to/file.json`,
			expectedExternalFilePath:     "./path/to/file.json",
			expectedExternalFileEncoding: "",
			expectedWithVariables:        true,
			expectedRawBody:              "<@ ./path/to/file.json",
		},
		{
			name: "External file with encoding specification",
			fileContent: `POST https://example.com/api/test
Content-Type: text/plain

<@latin1 ./path/to/file.txt`,
			expectedExternalFilePath:     "./path/to/file.txt",
			expectedExternalFileEncoding: "latin1",
			expectedWithVariables:        true,
			expectedRawBody:              "<@latin1 ./path/to/file.txt",
		},
		{
			name: "External file with path containing spaces",
			fileContent: `POST https://example.com/api/test
Content-Type: application/json

<@ ./path with spaces/file.json`,
			expectedExternalFilePath:     "./path with spaces/file.json",
			expectedExternalFileEncoding: "",
			expectedWithVariables:        true,
			expectedRawBody:              "<@ ./path with spaces/file.json",
		},
		{
			name: "External file with encoding and path with spaces",
			fileContent: `POST https://example.com/api/test
Content-Type: text/plain

<@cp1252 ./path with spaces/file.txt`,
			expectedExternalFilePath:     "./path with spaces/file.txt",
			expectedExternalFileEncoding: "cp1252",
			expectedWithVariables:        true,
			expectedRawBody:              "<@cp1252 ./path with spaces/file.txt",
		},
		{
			name: "External file with non-encoding first part",
			fileContent: `POST https://example.com/api/test
Content-Type: application/json

<@ ./some/path.json`,
			expectedExternalFilePath:     "./some/path.json",
			expectedExternalFileEncoding: "",
			expectedWithVariables:        true,
			expectedRawBody:              "<@ ./some/path.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file with the test content
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.http")
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			client, err := NewClient()
			require.NoError(t, err)

			parsedFile, err := parseRequestFile(testFile, client, make([]string, 0))
			require.NoError(t, err)
			require.Len(t, parsedFile.Requests, 1)

			req := parsedFile.Requests[0]
			assert.Equal(t, tt.expectedExternalFilePath, req.ExternalFilePath, "ExternalFilePath mismatch")
			assert.Equal(t, tt.expectedExternalFileEncoding, req.ExternalFileEncoding, "ExternalFileEncoding mismatch")
			assert.Equal(t, tt.expectedWithVariables, req.ExternalFileWithVariables, "ExternalFileWithVariables mismatch")
			assert.Equal(t, tt.expectedRawBody, req.RawBody, "RawBody mismatch")
		})
	}
}

func TestIsValidEncoding(t *testing.T) {
	tests := []struct {
		encoding string
		valid    bool
	}{
		{"utf-8", true},
		{"utf8", true},
		{"latin1", true},
		{"iso-8859-1", true},
		{"ascii", true},
		{"cp1252", true},
		{"windows-1252", true},
		{"UTF-8", true},  // Case insensitive
		{"LATIN1", true}, // Case insensitive
		{"invalid", false},
		{"random", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.encoding, func(t *testing.T) {
			result := isValidEncoding(tt.encoding)
			assert.Equal(t, tt.valid, result, "isValidEncoding result mismatch for %s", tt.encoding)
		})
	}
}
