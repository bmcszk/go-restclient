package restclient

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T {
	return &v
}

func TestValidateResponse_NilInputs(t *testing.T) {
	assert.NotNil(t, ValidateResponse(nil, &ExpectedResponse{}))
	assert.NotNil(t, ValidateResponse(&Response{}, nil))
	assert.Len(t, ValidateResponse(nil, &ExpectedResponse{}), 1)
	assert.Contains(t, ValidateResponse(nil, &ExpectedResponse{})[0].Error(), "actual response is nil")
	assert.Len(t, ValidateResponse(&Response{}, nil), 1)
	assert.Contains(t, ValidateResponse(&Response{}, nil)[0].Error(), "expected response is nil")
}

func TestValidateResponse_StatusCode(t *testing.T) {
	tests := []struct {
		name             string
		actual           *Response
		expected         *ExpectedResponse
		expectedErrCount int
		expectedErrText  string // if count is 1, check this text
	}{
		{
			name:             "matching status code",
			actual:           &Response{StatusCode: 200},
			expected:         &ExpectedResponse{StatusCode: ptr(200)},
			expectedErrCount: 0,
		},
		{
			name:             "mismatching status code",
			actual:           &Response{StatusCode: 500},
			expected:         &ExpectedResponse{StatusCode: ptr(200)},
			expectedErrCount: 1,
			expectedErrText:  "status code mismatch: expected 200, got 500",
		},
		{
			name:             "nil expected status code (ignore)",
			actual:           &Response{StatusCode: 200, BodyString: "actual body"}, // Provide an actual body
			expected:         &ExpectedResponse{StatusCode: nil, Body: nil},         // StatusCode is nil, and Body is nil to not trigger body validation here
			expectedErrCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateResponse(tt.actual, tt.expected)
			assert.Len(t, errs, tt.expectedErrCount)
			if tt.expectedErrCount == 1 && len(errs) == 1 {
				assert.Contains(t, errs[0].Error(), tt.expectedErrText)
			}
		})
	}
}

func TestValidateResponse_StatusString(t *testing.T) {
	tests := []struct {
		name             string
		actual           *Response
		expected         *ExpectedResponse
		expectedErrCount int
		expectedErrText  string
	}{
		{
			name:             "matching status string",
			actual:           &Response{Status: "200 OK"},
			expected:         &ExpectedResponse{Status: ptr("200 OK")}, // Use ptr for *string
			expectedErrCount: 0,
		},
		{
			name:             "mismatching status string",
			actual:           &Response{Status: "500 Internal Server Error"},
			expected:         &ExpectedResponse{Status: ptr("200 OK")}, // Use ptr for *string
			expectedErrCount: 1,
			expectedErrText:  "status string mismatch: expected '200 OK', got '500 Internal Server Error'",
		},
		{
			name:             "nil expected status string (ignore)",
			actual:           &Response{Status: "200 OK"},
			expected:         &ExpectedResponse{Status: nil}, // Nil means ignore
			expectedErrCount: 0,
		},
		{
			name:             "empty expected status string (ignore)",
			actual:           &Response{Status: "200 OK"},
			expected:         &ExpectedResponse{Status: ptr("")}, // Empty string in ptr means ignore
			expectedErrCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateResponse(tt.actual, tt.expected)
			assert.Len(t, errs, tt.expectedErrCount)
			if tt.expectedErrCount == 1 && len(errs) == 1 {
				assert.Contains(t, errs[0].Error(), tt.expectedErrText)
			}
		})
	}
}

func TestValidateResponse_Headers(t *testing.T) {
	tests := []struct {
		name             string
		actual           *Response
		expected         *ExpectedResponse
		expectedErrCount int
		expectedErrTexts []string // All these substrings must be in the errors
	}{
		{
			name:             "matching headers",
			actual:           &Response{Headers: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"123"}}},
			expected:         &ExpectedResponse{Headers: http.Header{"Content-Type": {"application/json"}}},
			expectedErrCount: 0,
		},
		{
			name:             "mismatching header value",
			actual:           &Response{Headers: http.Header{"Content-Type": {"text/html"}}},
			expected:         &ExpectedResponse{Headers: http.Header{"Content-Type": {"application/json"}}},
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'application/json' for header 'Content-Type' not found"},
		},
		{
			name:             "missing expected header",
			actual:           &Response{Headers: http.Header{"X-Other": {"value"}}},
			expected:         &ExpectedResponse{Headers: http.Header{"Content-Type": {"application/json"}}},
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected header 'Content-Type' not found"},
		},
		{
			name:             "multiple expected values, one missing",
			actual:           &Response{Headers: http.Header{"Vary": {"Accept-Encoding"}}},
			expected:         &ExpectedResponse{Headers: http.Header{"Vary": {"Accept-Encoding", "User-Agent"}}},
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'User-Agent' for header 'Vary' not found"},
		},
		{
			name:             "nil expected headers (ignore)",
			actual:           &Response{Headers: http.Header{"Content-Type": {"application/json"}}},
			expected:         &ExpectedResponse{Headers: nil},
			expectedErrCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateResponse(tt.actual, tt.expected)
			assert.Len(t, errs, tt.expectedErrCount)
			for _, errText := range tt.expectedErrTexts {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Error(), errText) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected error text not found: %s", errText)
			}
		})
	}
}

func TestValidateResponse_Body_ExactMatch(t *testing.T) {
	body1 := "Hello World"
	body2 := "Hello Go"
	tests := []struct {
		name             string
		actual           *Response
		expected         *ExpectedResponse
		expectedErrCount int
		expectedErrText  string // for diff check, this can be a substring of the diff
	}{
		{
			name:             "matching body",
			actual:           &Response{BodyString: body1},
			expected:         &ExpectedResponse{Body: ptr(body1)},
			expectedErrCount: 0,
		},
		{
			name:             "mismatching body",
			actual:           &Response{BodyString: body2},
			expected:         &ExpectedResponse{Body: ptr(body1)},
			expectedErrCount: 1,
			expectedErrText:  "--- Expected Body\n+++ Actual Body", // Start of diff
		},
		{
			name:             "nil expected body (ignore)",
			actual:           &Response{BodyString: body1},
			expected:         &ExpectedResponse{Body: nil},
			expectedErrCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateResponse(tt.actual, tt.expected)
			assert.Len(t, errs, tt.expectedErrCount)
			if tt.expectedErrCount == 1 && len(errs) == 1 {
				assert.Contains(t, errs[0].Error(), tt.expectedErrText)
			}
		})
	}
}

func TestValidateResponse_BodyContains(t *testing.T) {
	tests := []struct {
		name             string
		actual           *Response
		expected         *ExpectedResponse
		expectedErrCount int
		expectedErrTexts []string
	}{
		{
			name:             "body contains expected substring",
			actual:           &Response{BodyString: "Hello World Wide Web"},
			expected:         &ExpectedResponse{BodyContains: []string{"World Wide"}},
			expectedErrCount: 0,
		},
		{
			name:             "body does not contain expected substring",
			actual:           &Response{BodyString: "Hello World"},
			expected:         &ExpectedResponse{BodyContains: []string{"Universe"}},
			expectedErrCount: 1,
			expectedErrTexts: []string{"actual body does not contain expected substring: 'Universe'"},
		},
		{
			name:             "body contains one but not another",
			actual:           &Response{BodyString: "Hello World"},
			expected:         &ExpectedResponse{BodyContains: []string{"Hello", "Universe"}},
			expectedErrCount: 1,
			expectedErrTexts: []string{"actual body does not contain expected substring: 'Universe'"},
		},
		{
			name:             "empty BodyContains (ignore)",
			actual:           &Response{BodyString: "Hello World"},
			expected:         &ExpectedResponse{BodyContains: []string{}}, // Empty means no checks
			expectedErrCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateResponse(tt.actual, tt.expected)
			assert.Len(t, errs, tt.expectedErrCount)
			for _, errText := range tt.expectedErrTexts {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Error(), errText) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected error text not found: %s", errText)
			}
		})
	}
}

func TestValidateResponse_BodyNotContains(t *testing.T) {
	tests := []struct {
		name             string
		actual           *Response
		expected         *ExpectedResponse
		expectedErrCount int
		expectedErrTexts []string
	}{
		{
			name:             "body does not contain unexpected substring",
			actual:           &Response{BodyString: "Hello World"},
			expected:         &ExpectedResponse{BodyNotContains: []string{"Universe"}},
			expectedErrCount: 0,
		},
		{
			name:             "body contains unexpected substring",
			actual:           &Response{BodyString: "Hello Universe"},
			expected:         &ExpectedResponse{BodyNotContains: []string{"Universe"}},
			expectedErrCount: 1,
			expectedErrTexts: []string{"actual body contains unexpected substring: 'Universe'"},
		},
		{
			name:             "empty BodyNotContains (ignore)",
			actual:           &Response{BodyString: "Hello Universe"},
			expected:         &ExpectedResponse{BodyNotContains: []string{}}, // Empty means no checks
			expectedErrCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateResponse(tt.actual, tt.expected)
			assert.Len(t, errs, tt.expectedErrCount)
			for _, errText := range tt.expectedErrTexts {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Error(), errText) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected error text not found: %s", errText)
			}
		})
	}
}

// TODO: Add comprehensive tests combining multiple validation types
// TODO: Add tests for JSONPath once that's part of ExpectedResponse and ValidateResponse
// func TestValidateResponse_JSONPathChecks(t *testing.T) { ... } // Commented out for now
// func TestValidateResponse_HeadersContain(t *testing.T) { ... } // Commented out for now
