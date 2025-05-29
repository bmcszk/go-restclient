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

func TestParseRequests_SeparatorComments(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		expectedCount int
		req1Method    string
		req1URL       string
		req2Method    string
		req2URL       string
	}{
		{
			name: "SCENARIO-LIB-027-001 & SCENARIO-LIB-027-004 combined",
			fileContent: `
GET https://example.com/api/resource1

### This is a comment for the first request block

POST https://example.com/api/resource2

### This line is just a separator comment

PUT https://example.com/api/resource3
`, // SCENARIO-LIB-027-003 style is implicitly handled by line-by-line parsing.
			expectedCount: 3,
			req1Method:    "GET",
			req1URL:       "https://example.com/api/resource1",
			// req2 is POST to /resource2, req3 is PUT to /resource3
		},
		{
			name: "SCENARIO-LIB-027-003 style: Separator comment no newline before next request",
			fileContent: `
GET https://example.com/api/item1 ### Comment for item1
POST https://example.com/api/item2
`,
			expectedCount: 2,
			req1Method:    "GET",
			req1URL:       "https://example.com/api/item1",
			req2Method:    "POST",
			req2URL:       "https://example.com/api/item2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.fileContent)
			parsedFile, err := parseRequests(reader, "test.http")
			require.NoError(t, err)
			require.NotNil(t, parsedFile)
			assert.Len(t, parsedFile.Requests, tt.expectedCount)

			if tt.expectedCount > 0 && len(parsedFile.Requests) > 0 {
				assert.Equal(t, tt.req1Method, parsedFile.Requests[0].Method)
				assert.Equal(t, tt.req1URL, parsedFile.Requests[0].RawURLString)
			}
			if tt.expectedCount > 1 && len(parsedFile.Requests) > 1 {
				// For simplicity, checking specific requests based on scenario structure
				switch tt.name {
				case "SCENARIO-LIB-027-001 & SCENARIO-LIB-027-004 combined":
					assert.Equal(t, "POST", parsedFile.Requests[1].Method)
					assert.Equal(t, "https://example.com/api/resource2", parsedFile.Requests[1].RawURLString)
					assert.Equal(t, "PUT", parsedFile.Requests[2].Method)
					assert.Equal(t, "https://example.com/api/resource3", parsedFile.Requests[2].RawURLString)
				case "SCENARIO-LIB-027-003 style: Separator comment no newline before next request":
					assert.Equal(t, tt.req2Method, parsedFile.Requests[1].Method)
					assert.Equal(t, tt.req2URL, parsedFile.Requests[1].RawURLString)
				}
			}
		})
	}
}

func TestParseExpectedResponses_SeparatorComments(t *testing.T) {
	tests := []struct {
		name           string
		fileContent    string
		expectedCount  int
		expResp1Status string
		expResp2Status string
	}{
		{
			name: "SCENARIO-LIB-027-002: Separator comment in response file",
			fileContent: `
HTTP/1.1 200 OK
Content-Type: application/json

{"status": "success"}

### This is a comment for the first response block

HTTP/1.1 404 Not Found
`,
			expectedCount:  2,
			expResp1Status: "200 OK",
			expResp2Status: "404 Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.fileContent)
			parsedResponses, err := parseExpectedResponses(reader, "test.hresp")
			require.NoError(t, err)
			require.NotNil(t, parsedResponses)
			assert.Len(t, parsedResponses, tt.expectedCount)

			if tt.expectedCount > 0 && len(parsedResponses) > 0 {
				require.NotNil(t, parsedResponses[0].Status)
				assert.Equal(t, tt.expResp1Status, *parsedResponses[0].Status)
			}
			if tt.expectedCount > 1 && len(parsedResponses) > 1 {
				require.NotNil(t, parsedResponses[1].Status)
				assert.Equal(t, tt.expResp2Status, *parsedResponses[1].Status)
			}
		})
	}
}
