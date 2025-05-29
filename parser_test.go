package restclient

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRequests_IgnoreEmptyBlocks(t *testing.T) {
	tests := []struct {
		name           string
		fileContent    string
		expectedCount  int
		expectedError  bool
		firstReqMethod string // if expectedCount > 0
		firstReqURL    string // if expectedCount > 0
		lastReqMethod  string // if expectedCount > 1
		lastReqURL     string // if expectedCount > 1
	}{
		{
			name: "SCENARIO-LIB-028-001: File with only comments",
			fileContent: `
# This is a comment
# Another comment
`, // Extra newline for clarity
			expectedCount: 0,
			expectedError: false,
		},
		{
			name: "SCENARIO-LIB-028-002: File with only ### separators",
			fileContent: `
###
###
###
`, // Extra newline
			expectedCount: 0,
			expectedError: false,
		},
		{
			name: "SCENARIO-LIB-028-003: File with comments and ### separators only",
			fileContent: `
# Request 1 (will be skipped)
###
# Request 2 (will be skipped)
###
`, // Extra newline
			expectedCount: 0,
			expectedError: false,
		},
		{
			name: "SCENARIO-LIB-028-004: Valid request, then separator, then only comments",
			fileContent: `
GET https://example.com/first

###
# This block is empty
`, // Extra newline
			expectedCount:  1,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "https://example.com/first",
		},
		{
			name: "SCENARIO-LIB-028-005: Only comments, then separator, then valid request",
			fileContent: `
# This block is empty
###
GET https://example.com/second
`, // Extra newline
			expectedCount:  1,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "https://example.com/second",
		},
		{
			name: "SCENARIO-LIB-028-006: Valid request, separator with comments, then another valid request",
			fileContent: `
GET https://example.com/req1

### Comment for empty block
# More comments

###
POST https://example.com/req2
Content-Type: application/json

{
  "key": "value"
}
`, // Extra newline
			expectedCount:  2,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "https://example.com/req1",
			lastReqMethod:  "POST",
			lastReqURL:     "https://example.com/req2",
		},
		{
			name:          "Empty file content",
			fileContent:   "",
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:           "Single valid request no trailing newline",
			fileContent:    "GET http://localhost/api/test",
			expectedCount:  1,
			expectedError:  false,
			firstReqMethod: "GET",
			firstReqURL:    "http://localhost/api/test",
		},
		{
			name: "File with only variable definitions",
			fileContent: `@host=localhost
@port=8080`,
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.fileContent)
			parsedFile, err := parseRequests(reader, "test.http")

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, parsedFile, "parsedFile should not be nil on no error")
				assert.Len(t, parsedFile.Requests, tt.expectedCount, "Number of parsed requests mismatch")

				if tt.expectedCount > 0 && len(parsedFile.Requests) > 0 {
					assert.Equal(t, tt.firstReqMethod, parsedFile.Requests[0].Method)
					assert.Equal(t, tt.firstReqURL, parsedFile.Requests[0].RawURLString)
				}
				if tt.expectedCount > 1 && len(parsedFile.Requests) > 1 {
					assert.Equal(t, tt.lastReqMethod, parsedFile.Requests[tt.expectedCount-1].Method)
					assert.Equal(t, tt.lastReqURL, parsedFile.Requests[tt.expectedCount-1].RawURLString)
				}
			}
		})
	}
}
