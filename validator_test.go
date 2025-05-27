package restclient

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to write expected response content to a temporary file
func writeExpectedResponseFile(t *testing.T, content string) string {
	t.Helper()
	tempFile, err := os.CreateTemp(t.TempDir(), "expected_*.hresp")
	require.NoError(t, err)
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	err = tempFile.Close()
	require.NoError(t, err)
	return tempFile.Name()
}

func TestValidateResponses_NilAndEmptyActuals(t *testing.T) {
	ctx := context.Background()
	testFilePath := writeExpectedResponseFile(t, "HTTP/1.1 200 OK\n\nBody")

	t.Run("nil actual response slice", func(t *testing.T) {
		var nilActuals []*Response // nil slice
		err := ValidateResponses(ctx, testFilePath, nilActuals...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mismatch in number of responses: got 0 actual, but expected 1")
	})

	t.Run("empty actual response slice", func(t *testing.T) {
		emptyActuals := []*Response{} // empty slice
		err := ValidateResponses(ctx, testFilePath, emptyActuals...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mismatch in number of responses: got 0 actual, but expected 1")
	})

	t.Run("slice with one nil actual response", func(t *testing.T) {
		oneNilActual := []*Response{nil}
		err := ValidateResponses(ctx, testFilePath, oneNilActual...)
		require.Error(t, err)
		merr, ok := err.(*multierror.Error)
		require.True(t, ok, "Expected a multierror.Error")
		require.Len(t, merr.Errors, 1)
		assert.Contains(t, merr.Errors[0].Error(), "mismatch in number of responses: got 0 actual, but expected 1")
	})
}

func TestValidateResponses_FileErrors(t *testing.T) {
	ctx := context.Background()
	actualResp := &Response{StatusCode: 200}

	t.Run("missing expected response file", func(t *testing.T) {
		err := ValidateResponses(ctx, "nonexistent.hresp", actualResp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse expected response file")
		assert.Contains(t, err.Error(), "nonexistent.hresp")
		// Note: os.IsNotExist might be useful for more specific check if not wrapped too much
	})

	t.Run("empty expected response file", func(t *testing.T) {
		emptyFilePath := writeExpectedResponseFile(t, "")
		err := ValidateResponses(ctx, emptyFilePath, actualResp)
		require.Error(t, err)
		merr, ok := err.(*multierror.Error)
		require.True(t, ok, "Expected a multierror.Error")
		// Error 1: failed to parse: no expected responses found in file
		// Error 2: mismatch in number of responses (1 actual vs 0 expected)
		assert.Len(t, merr.Errors, 2)
		assert.ErrorContains(t, merr.Errors[0], "failed to parse expected response file")
		assert.ErrorContains(t, merr.Errors[0], "no valid expected responses found in file") // Check underlying error
		assert.ErrorContains(t, merr.Errors[1], "mismatch in number of responses: got 1 actual, but expected 0")
	})

	t.Run("malformed expected response file", func(t *testing.T) {
		malformedFilePath := writeExpectedResponseFile(t, "HTTP/1.1 INVALID")
		err := ValidateResponses(ctx, malformedFilePath, actualResp)
		require.Error(t, err)
		merr, ok := err.(*multierror.Error)
		require.True(t, ok, "Expected a multierror.Error")
		// Error 1: failed to parse: invalid status line / invalid status code
		// Error 2: mismatch in number of responses (1 actual vs 0 expected)
		assert.Len(t, merr.Errors, 2)
		assert.ErrorContains(t, merr.Errors[0], "failed to parse expected response file")
		assert.ErrorContains(t, merr.Errors[0], "invalid status code") // Or "invalid status line" depending on parser detail
		assert.ErrorContains(t, merr.Errors[1], "mismatch in number of responses: got 1 actual, but expected 0")
	})
}

func TestValidateResponses_StatusString(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name                string
		actualResponse      *Response
		expectedFileContent string
		expectedErrCount    int
		expectedErrText     string
		expectedErrTexts    []string
	}{
		{
			name:                "matching status string",
			actualResponse:      &Response{StatusCode: 200, Status: "200 OK"},
			expectedFileContent: "HTTP/1.1 200 OK",
			expectedErrCount:    0,
		},
		{
			name:                "mismatching status string",
			actualResponse:      &Response{StatusCode: 200, Status: "200 Something Else"},
			expectedFileContent: "HTTP/1.1 200 OK",
			expectedErrCount:    1,
			expectedErrText:     "status string mismatch: expected '200 OK', got '200 Something Else'",
		},
		{
			name:                "actual status string is correct, expected file has only status code",
			actualResponse:      &Response{StatusCode: 200, Status: "200 OK"},
			expectedFileContent: "HTTP/1.1 200",
			expectedErrCount:    1,
			expectedErrText:     "status string mismatch: expected '200', got '200 OK'",
		},
		{
			name:                "mismatching status code, status strings also mismatch",
			actualResponse:      &Response{StatusCode: 404, Status: "404 Not Found"},
			expectedFileContent: "HTTP/1.1 200 OK",
			expectedErrCount:    2,
			expectedErrTexts:    []string{"status code mismatch: expected 200, got 404", "status string mismatch: expected '200 OK', got '404 Not Found'"},
		},
		{
			name:                "matching status code, expected file only code, actual also only code in status",
			actualResponse:      &Response{StatusCode: 200, Status: "200"},
			expectedFileContent: "HTTP/1.1 200",
			expectedErrCount:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := writeExpectedResponseFile(t, tt.expectedFileContent)
			err := ValidateResponses(ctx, filePath, tt.actualResponse)

			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				merr, ok := err.(*multierror.Error)
				require.True(t, ok, "Expected a multierror.Error")
				assert.Len(t, merr.Errors, tt.expectedErrCount)
				if tt.expectedErrText != "" {
					require.Len(t, merr.Errors, 1, "expectedErrText is for single error cases")
					assert.Contains(t, merr.Errors[0].Error(), tt.expectedErrText)
				}
				if len(tt.expectedErrTexts) > 0 {
					for _, expectedText := range tt.expectedErrTexts {
						found := false
						for _, e := range merr.Errors {
							if strings.Contains(e.Error(), expectedText) {
								found = true
								break
							}
						}
						assert.True(t, found, "Expected error text '%s' not found in %v", expectedText, merr.Errors)
					}
				}
			}
		})
	}
}

func TestValidateResponses_Headers(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name                string
		actualResponse      *Response
		expectedFileContent string // Will contain status line + headers
		expectedErrCount    int
		expectedErrTexts    []string
	}{
		{
			name:           "matching headers",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"123"}}},
			expectedFileContent: `HTTP/1.1 200 OK
Content-Type: application/json`,
			expectedErrCount: 0,
		},
		{
			name:           "mismatching header value",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"text/html"}}},
			expectedFileContent: `HTTP/1.1 200 OK
Content-Type: application/json`,
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'application/json' for header 'Content-Type' not found"},
		},
		{
			name:           "missing expected header",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"X-Other": {"value"}}},
			expectedFileContent: `HTTP/1.1 200 OK
Content-Type: application/json`,
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected header 'Content-Type' not found"},
		},
		{
			name:           "multiple expected values in file, one missing in actual",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Vary": {"Accept-Encoding"}}},
			expectedFileContent: `HTTP/1.1 200 OK
Vary: Accept-Encoding
Vary: User-Agent`,
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'User-Agent' for header 'Vary' not found"},
		},
		{
			name:                "no headers in expected file (ignore actual headers)",
			actualResponse:      &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"application/json"}}},
			expectedFileContent: `HTTP/1.1 200 OK`, // No headers here
			expectedErrCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := writeExpectedResponseFile(t, tt.expectedFileContent)
			err := ValidateResponses(ctx, filePath, tt.actualResponse)

			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				merr, ok := err.(*multierror.Error)
				require.True(t, ok, "Expected a multierror.Error")
				assert.Len(t, merr.Errors, tt.expectedErrCount)
				for _, errText := range tt.expectedErrTexts {
					found := false
					for _, e := range merr.Errors {
						if strings.Contains(e.Error(), errText) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error text not found: %s in %v", errText, merr.Errors)
				}
			}
		})
	}
}

func TestValidateResponses_Body_ExactMatch(t *testing.T) {
	ctx := context.Background()
	body1 := "Hello World"
	body2 := "Hello Go"
	tests := []struct {
		name                string
		actualResponse      *Response
		expectedFileContent string // Will contain status line + body
		expectedErrCount    int
		expectedErrTexts    []string // Changed from expectedErrText
	}{
		{
			name:                "matching body",
			actualResponse:      &Response{StatusCode: 200, Status: "200 OK", BodyString: body1},
			expectedFileContent: "HTTP/1.1 200 OK\n\n" + body1,
			expectedErrCount:    0,
		},
		{
			name:                "mismatching body",
			actualResponse:      &Response{StatusCode: 200, Status: "200 OK", BodyString: body2},
			expectedFileContent: "HTTP/1.1 200 OK\n\n" + body1,
			expectedErrCount:    1,
			expectedErrTexts:    []string{"body mismatch"}, // Diff will be part of the message
		},
		{
			name:                "empty body in file, actual has content",
			actualResponse:      &Response{StatusCode: 200, Status: "200 OK", BodyString: body1},
			expectedFileContent: "HTTP/1.1 200 OK\n\n", // Empty body
			expectedErrCount:    1,
			expectedErrTexts:    []string{"body mismatch"},
		},
		{
			name:                "empty body in file, actual also empty",
			actualResponse:      &Response{StatusCode: 200, Status: "200 OK", BodyString: ""},
			expectedFileContent: "HTTP/1.1 200 OK\n\n", // Empty body
			expectedErrCount:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := writeExpectedResponseFile(t, tt.expectedFileContent)
			err := ValidateResponses(ctx, filePath, tt.actualResponse)

			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				merr, ok := err.(*multierror.Error)
				require.True(t, ok, "Expected a multierror.Error")
				assert.Len(t, merr.Errors, tt.expectedErrCount)
				if len(tt.expectedErrTexts) > 0 {
					for _, expectedText := range tt.expectedErrTexts {
						found := false
						for _, e := range merr.Errors {
							if strings.Contains(e.Error(), expectedText) {
								found = true
								break
							}
						}
						assert.True(t, found, "Expected error text '%s' not found in %v", expectedText, merr.Errors)
					}
				}
			}
		})
	}
}

// TestValidateResponses_BodyContains tests the BodyContains logic.
// Since ParseExpectedResponseFile does not populate ExpectedResponse.BodyContains,
// this test verifies that the BodyContains logic in ValidateResponses is benign
// (doesn't cause errors) when the expected response comes from a file.
func TestValidateResponses_BodyContains(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name           string
		actualResponse *Response
		// expectedFileContent will be simple, as it cannot express BodyContains
		expectedFileContent string
		expectedErrCount    int // Should be 0 if actual matches file's explicit parts
		expectedErrTexts    []string
	}{
		{
			name:           "BodyContains logic is not triggered by file (positive case)",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello World Wide Web"},
			// Exact match for body
			expectedFileContent: "HTTP/1.1 200 OK\n\nHello World Wide Web",
			expectedErrCount:    0,
		},
		{
			name:           "BodyContains logic not triggered, body mismatch handled by exact check",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello World"},
			// File expects "Universe" - this will cause an exact body mismatch.
			// The BodyContains part of ValidateResponses will not run due to empty expected.BodyContains.
			expectedFileContent: "HTTP/1.1 200 OK\n\nHello Universe",
			expectedErrCount:    1,
			expectedErrTexts:    []string{"body mismatch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := writeExpectedResponseFile(t, tt.expectedFileContent)
			err := ValidateResponses(ctx, filePath, tt.actualResponse)

			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				merr, ok := err.(*multierror.Error)
				require.True(t, ok, "Expected a multierror.Error")
				assert.Len(t, merr.Errors, tt.expectedErrCount)
				for _, errText := range tt.expectedErrTexts {
					found := false
					for _, e := range merr.Errors {
						if strings.Contains(e.Error(), errText) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error text not found: %s in %v", errText, merr.Errors)
				}
			}
		})
	}
}

// TestValidateResponses_BodyNotContains is similar to BodyContains.
// It verifies that BodyNotContains logic in ValidateResponses is benign
// when the expected response comes from a file, as the file cannot specify BodyNotContains.
func TestValidateResponses_BodyNotContains(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name                string
		actualResponse      *Response
		expectedFileContent string
		expectedErrCount    int
		expectedErrTexts    []string
	}{
		{
			name:           "BodyNotContains logic is not triggered by file (positive case)",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello World"},
			// Exact match for body
			expectedFileContent: "HTTP/1.1 200 OK\n\nHello World",
			expectedErrCount:    0,
		},
		{
			name:           "BodyNotContains logic not triggered, actual contains something, file expects different exact body",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello Universe"},
			// Exact body mismatch
			expectedFileContent: "HTTP/1.1 200 OK\n\nHello World",
			expectedErrCount:    1,
			expectedErrTexts:    []string{"body mismatch"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := writeExpectedResponseFile(t, tt.expectedFileContent)
			err := ValidateResponses(ctx, filePath, tt.actualResponse)

			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				merr, ok := err.(*multierror.Error)
				require.True(t, ok, "Expected a multierror.Error")
				assert.Len(t, merr.Errors, tt.expectedErrCount)
				for _, errText := range tt.expectedErrTexts {
					found := false
					for _, e := range merr.Errors {
						if strings.Contains(e.Error(), errText) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error text not found: %s in %v", errText, merr.Errors)
				}
			}
		})
	}
}

func TestValidateResponses_WithSampleFile(t *testing.T) {
	ctx := context.Background()
	// Content of testdata/http_response_files/sample1.http
	// HTTP/1.1 200 OK
	// Content-Type: application/json; charset=utf-8
	// Date: Mon, 27 May 2024 12:00:00 GMT
	// Connection: keep-alive
	//
	// {
	//   "userId": 1,
	//   "id": 1,
	//   "title": "delectus aut autem",
	//   "completed": false
	// }
	sampleFilePath := "testdata/http_response_files/sample1.http" // Use the actual file

	// Parse sample1.http once to get the structure for baseActual
	// This is just to help create a baseActual; ValidateResponses will parse it again internally.
	parsedSampleExpected, err := ParseExpectedResponseFile(sampleFilePath)
	require.NoError(t, err)
	require.Len(t, parsedSampleExpected, 1)
	sampleExpectedStruct := parsedSampleExpected[0]

	baseActual := &Response{
		StatusCode: *sampleExpectedStruct.StatusCode,
		Status:     *sampleExpectedStruct.Status,
		Headers:    make(http.Header),
		BodyString: *sampleExpectedStruct.Body,
	}
	for k, v := range sampleExpectedStruct.Headers { // Deep copy headers
		baseActual.Headers[k] = append([]string{}, v...)
	}

	tests := []struct {
		name           string
		actualModifier func(actual *Response)
		// expectedFileContent will typically be sampleFilePath or a variation
		// if the test is about how ValidateResponses handles file content.
		// For features not supported by file (BodyContains etc.), this will be sampleFilePath,
		// and the test verifies no unexpected errors.
		expectedFileSource        string // path to file or "inline" to use expectedFileContentString
		expectedFileContentString string // used if expectedFileSource is "inline"
		expectedErrCount          int
		expectedErrTexts          []string
	}{
		{
			name:               "perfect match with sample1.http",
			actualModifier:     func(actual *Response) {}, // No change to baseActual
			expectedFileSource: sampleFilePath,
			expectedErrCount:   0,
		},
		{
			name: "status code mismatch",
			actualModifier: func(actual *Response) {
				actual.StatusCode = 500
			},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   1,
			expectedErrTexts:   []string{"status code mismatch: expected 200, got 500"},
		},
		{
			name: "status string mismatch",
			actualModifier: func(actual *Response) {
				actual.Status = "200 Everything is Fine"
			},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   1,
			// sample1.http has "HTTP/1.1 200 OK", parser gives status "200 OK"
			expectedErrTexts: []string{"status string mismatch: expected '200 OK', got '200 Everything is Fine'"},
		},
		{
			name: "header value mismatch for Content-Type",
			actualModifier: func(actual *Response) {
				actual.Headers.Set("Content-Type", "text/plain")
			},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   1,
			expectedErrTexts:   []string{"expected value 'application/json; charset=utf-8' for header 'Content-Type' not found"},
		},
		{
			name: "missing expected header Date",
			actualModifier: func(actual *Response) {
				actual.Headers.Del("Date")
			},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   1,
			expectedErrTexts:   []string{"expected header 'Date' not found"},
		},
		{
			name: "body mismatch",
			actualModifier: func(actual *Response) {
				actual.BodyString = "{\"message\": \"this is not the sample body\"}"
			},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   1,
			expectedErrTexts:   []string{"body mismatch"}, // Diff will follow
		},
		{
			// This test now verifies that BodyContains logic in ValidateResponses
			// is NOT triggered by an exact match expected file, and the overall validation passes.
			name:               "BodyContains logic not triggered by exact file match (positive case)",
			actualModifier:     func(actual *Response) { /* use baseActual body which contains "delectus aut autem" */ },
			expectedFileSource: sampleFilePath, // sample1.http matches baseActual perfectly
			expectedErrCount:   0,
		},
		{
			// This test now verifies that if the file specifies an exact body,
			// and it mismatches, BodyContains logic is not involved.
			name:                      "BodyContains logic not triggered, exact body mismatch from file",
			actualModifier:            func(actual *Response) { /* baseActual body */ },
			expectedFileSource:        "inline",
			expectedFileContentString: "HTTP/1.1 200 OK\n\n{\"message\": \"nonexistent substring in sample1\"}",
			expectedErrCount:          1,
			expectedErrTexts:          []string{"body mismatch"},
		},
		{
			name:               "BodyNotContains logic not triggered by exact file match (positive case)",
			actualModifier:     func(actual *Response) { /* use baseActual body */ },
			expectedFileSource: sampleFilePath, // sample1.http matches baseActual perfectly
			expectedErrCount:   0,
		},
		{
			name:                      "BodyNotContains logic not triggered, exact body mismatch from file (actual contains something unwanted by this hypothetical check)",
			actualModifier:            func(actual *Response) { actual.BodyString = "{\"title\": \"delectus aut autem\"}" }, // This IS in the baseActual body
			expectedFileSource:        "inline",                                                                             // File expects something different
			expectedFileContentString: "HTTP/1.1 200 OK\n\n{\"title\": \"completely different thing\"}",
			expectedErrCount:          1,
			expectedErrTexts:          []string{"body mismatch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTest := &Response{
				StatusCode: baseActual.StatusCode,
				Status:     baseActual.Status,
				Headers:    make(http.Header),
				BodyString: baseActual.BodyString,
			}
			for k, v := range baseActual.Headers {
				actualTest.Headers[k] = append([]string{}, v...)
			}
			tt.actualModifier(actualTest)

			currentExpectedFilePath := tt.expectedFileSource
			if tt.expectedFileSource == "inline" {
				currentExpectedFilePath = writeExpectedResponseFile(t, tt.expectedFileContentString)
			}

			err := ValidateResponses(ctx, currentExpectedFilePath, actualTest)

			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				merr, ok := err.(*multierror.Error)
				require.True(t, ok, "Expected a multierror.Error")
				assert.Len(t, merr.Errors, tt.expectedErrCount)
				for _, errText := range tt.expectedErrTexts {
					found := false
					for _, e := range merr.Errors {
						if strings.Contains(e.Error(), errText) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error text '%s' not found in errors: %v", errText, merr.Errors)
				}
			}
		})
	}
}

// TODO: Add comprehensive tests combining multiple validation types
// TODO: Add tests for JSONPath once that's part of ExpectedResponse and ValidateResponse
// func TestValidateResponse_JSONPathChecks(t *testing.T) { ... } // Commented out for now
// func TestValidateResponse_HeadersContain(t *testing.T) { ... } // Commented out for now

// TestValidateResponses_HeadersContain verifies that HeadersContain logic in ValidateResponses
// is benign when the expected response comes from a file, as the file cannot specify HeadersContain.
func TestValidateResponses_HeadersContain(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name                string
		actualResponse      *Response
		expectedFileContent string
		expectedErrCount    int
		expectedErrTexts    []string
	}{
		{
			name:           "HeadersContain logic not triggered (matching case for other fields)",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"application/json; charset=utf-8"}}},
			expectedFileContent: `HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8`,
			expectedErrCount: 0,
		},
		{
			name:           "HeadersContain logic not triggered (mismatch on standard header)",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"text/html"}}},
			expectedFileContent: `HTTP/1.1 200 OK
Content-Type: application/json`, // Standard header mismatch
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'application/json' for header 'Content-Type' not found"},
		},
		{
			name:           "HeadersContain logic not triggered (expected header key not found by standard check)",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"X-Other": {"value"}}},
			expectedFileContent: `HTTP/1.1 200 OK
Content-Type: application/json`, // Expected header 'Content-Type' missing from actual
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected header 'Content-Type' not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := writeExpectedResponseFile(t, tt.expectedFileContent)
			err := ValidateResponses(ctx, filePath, tt.actualResponse)

			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				merr, ok := err.(*multierror.Error)
				require.True(t, ok, "Expected a multierror.Error")
				assert.Len(t, merr.Errors, tt.expectedErrCount)
				for _, errText := range tt.expectedErrTexts {
					found := false
					for _, e := range merr.Errors {
						if strings.Contains(e.Error(), errText) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error text '%s' not found in errors: %v", errText, merr.Errors)
				}
			}
		})
	}
}

func TestValidateResponses_PartialExpected(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name           string
		actualResponse *Response
		// expectedFileContent will represent the "partial" expectation.
		expectedFileContent string
		expectedErrCount    int
		expectedErrTexts    []string
	}{
		{
			name: "SCENARIO-LIB-009-005 Equiv: Expected file has only status code - match",
			actualResponse: &Response{
				StatusCode: 200,
				Status:     "200", // This will now match the parsed "200" from the file
				Headers:    http.Header{"Content-Type": {"application/json"}},
				BodyString: "", // Actual body must be empty to match empty body from file
			},
			expectedFileContent: "HTTP/1.1 200\n\n", // Parses to StatusCode:200, Status:"200", Headers:empty, Body:""
			expectedErrCount:    0,
		},
		{
			name:           "SCENARIO-LIB-009-005 Corrected: File has status code and empty body - actual matches",
			actualResponse: &Response{StatusCode: 200, Status: "200", BodyString: ""},
			expectedFileContent: `HTTP/1.1 200

`, // Status:"200", Body:""
			expectedErrCount: 0,
		},
		{
			name:           "SCENARIO-LIB-009-005 Corrected: File has status code and empty body - actual body mismatch",
			actualResponse: &Response{StatusCode: 200, Status: "200", BodyString: "non-empty body"},
			expectedFileContent: `HTTP/1.1 200

`,
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"},
		},
		{
			name: "SCENARIO-LIB-009-005 Equiv: Expected file has only status code - status code mismatch",
			actualResponse: &Response{
				StatusCode: 404,   // Mismatch
				Status:     "404", // Mismatch vs "200"
				Headers:    http.Header{"Content-Type": {"application/json"}},
				BodyString: "", // Match empty body from file
			},
			expectedFileContent: `HTTP/1.1 200

`,
			expectedErrCount: 2, // Status code and Status string
			expectedErrTexts: []string{"status code mismatch: expected 200, got 404", "status string mismatch: expected '200', got '404'"},
		},
		{
			name: "SCENARIO-LIB-009-006 Equiv: Expected file has only specific headers (and status, empty body) - match",
			actualResponse: &Response{
				StatusCode: 200,
				Status:     "200",
				Headers:    http.Header{"Content-Type": {"application/json"}, "X-Custom": {"val"}},
				BodyString: "",
			},
			expectedFileContent: `HTTP/1.1 200
Content-Type: application/json

`,
			expectedErrCount: 0,
		},
		{
			name: "SCENARIO-LIB-009-006 Equiv: Expected file has only specific headers - header value mismatch",
			actualResponse: &Response{
				StatusCode: 200,
				Status:     "200",
				Headers:    http.Header{"Content-Type": {"text/plain"}, "X-Custom": {"val"}}, // Mismatch Content-Type
				BodyString: "",
			},
			expectedFileContent: `HTTP/1.1 200
Content-Type: application/json

`,
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'application/json' for header 'Content-Type' not found"},
		},
		{
			name: "SCENARIO-LIB-009-006 Equiv: Expected file has only specific headers - header missing in actual",
			actualResponse: &Response{
				StatusCode: 200,
				Status:     "200",
				Headers:    http.Header{"X-Other": {"value"}}, // Content-Type missing
				BodyString: "",
			},
			expectedFileContent: `HTTP/1.1 200
Content-Type: application/json

`,
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected header 'Content-Type' not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := writeExpectedResponseFile(t, tt.expectedFileContent)
			err := ValidateResponses(ctx, filePath, tt.actualResponse)

			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				merr, ok := err.(*multierror.Error)
				require.True(t, ok, "Expected a multierror.Error")
				assert.Len(t, merr.Errors, tt.expectedErrCount)
				if len(tt.expectedErrTexts) > 0 {
					for _, errText := range tt.expectedErrTexts {
						found := false
						for _, e := range merr.Errors {
							if strings.Contains(e.Error(), errText) {
								found = true
								break
							}
						}
						assert.True(t, found, "Expected error text '%s' not found in errors: %v", errText, merr.Errors)
					}
				}
			}
		})
	}
}
