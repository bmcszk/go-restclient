package restclient

import (
	"net/http"
	"testing"

	// Imported to ensure assertMultierrorContains compiles
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// No fmt needed
	// No strings needed
)

func TestValidateResponses_WithSampleFile(t *testing.T) {
	// Given: Setup with a sample response file and base actual response
	sampleFilePath := "testdata/http_response_files/sample1.http" // Use the actual file

	parsedSampleExpected, err := parseExpectedResponseFile(sampleFilePath)
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
		name                      string
		actualModifier            func(actual *Response)
		expectedFileSource        string
		expectedFileContentString string // No longer used by the logic, but kept for structure if needed later
		expectedErrCount          int
		expectedErrTexts          []string
	}{
		{
			name:               "perfect match with sample1.http",
			actualModifier:     func(actual *Response) {},
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
			expectedErrTexts:   []string{"status string mismatch: expected '200 OK', got '200 Everything is Fine'"},
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
			expectedErrTexts:   []string{"body mismatch"},
		},
		{
			name:               "BodyContains logic not triggered by exact file match (positive case)",
			actualModifier:     func(actual *Response) {},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   0,
		},
		{
			name:               "BodyContains logic not triggered, exact body mismatch from file",
			actualModifier:     func(actual *Response) {},
			expectedFileSource: "testdata/http_response_files/validator_withsample_bodycontains_exactmismatch.hresp",
			expectedErrCount:   1,
			expectedErrTexts:   []string{"body mismatch"},
		},
		{
			name:               "BodyNotContains logic not triggered by exact file match (positive case)",
			actualModifier:     func(actual *Response) {},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   0,
		},
		{
			name:               "BodyNotContains logic not triggered, exact body mismatch from file (actual contains something unwanted by this hypothetical check)",
			actualModifier:     func(actual *Response) { actual.BodyString = "{\"title\": \"delectus aut autem\"}" },
			expectedFileSource: "testdata/http_response_files/validator_withsample_bodynotcontains_exactmismatch.hresp",
			expectedErrCount:   1,
			expectedErrTexts:   []string{"body mismatch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: A modified actual response based on baseActual and tt.actualModifier
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
			client, _ := NewClient()

			// When
			err := client.ValidateResponses(currentExpectedFilePath, actualTest)

			// Then
			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
			}
		})
	}
}

func TestValidateResponses_PartialExpected(t *testing.T) {
	// Given: Test cases defined in 'tests' slice
	tests := []struct {
		name             string
		actualResponse   *Response
		expectedFilePath string // Was expectedFileContent
		expectedErrCount int
		expectedErrTexts []string
	}{
		{
			name: "SCENARIO-LIB-009-005 Equiv: Expected file has only status code - match",
			actualResponse: &Response{
				StatusCode: 200,
				Status:     "200",
				Headers:    http.Header{"Content-Type": {"application/json"}},
				BodyString: "",
			},
			expectedFilePath: "testdata/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-009-005 Corrected: File has status code and empty body - actual matches",
			actualResponse:   &Response{StatusCode: 200, Status: "200", BodyString: ""},
			expectedFilePath: "testdata/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-009-005-003 Corrected: File has status code and empty body - actual body mismatch",
			actualResponse:   &Response{StatusCode: 200, Status: "200", BodyString: "non-empty body"},
			expectedFilePath: "testdata/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"},
		},
		{
			name: "SCENARIO-LIB-009-005-004 Equiv: Expected file has only status code - status code mismatch",
			actualResponse: &Response{
				StatusCode: 404,
				Status:     "404",
				Headers:    http.Header{"Content-Type": {"application/json"}},
				BodyString: "",
			},
			expectedFilePath: "testdata/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount: 2,
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
			expectedFilePath: "testdata/http_response_files/validator_partial_headers_key_missing.hresp",
			expectedErrCount: 0,
		},
		{
			name: "SCENARIO-LIB-009-006-002 Equiv: Expected file has only specific headers - header value mismatch",
			actualResponse: &Response{
				StatusCode: 200,
				Status:     "200",
				Headers:    http.Header{"Content-Type": {"text/plain"}, "X-Custom": {"val"}},
				BodyString: "",
			},
			expectedFilePath: "testdata/http_response_files/validator_partial_headers_key_missing.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'application/json' for header 'Content-Type' not found"},
		},
		{
			name: "SCENARIO-LIB-009-006-003 Equiv: Expected file has only specific headers - header missing in actual",
			actualResponse: &Response{
				StatusCode: 200,
				Status:     "200",
				Headers:    http.Header{"X-Other": {"value"}},
				BodyString: "",
			},
			expectedFilePath: "testdata/http_response_files/validator_partial_headers_key_missing.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected header 'Content-Type' not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: actualResponse and expectedFilePath from the test case tt
			client, _ := NewClient()

			// When
			err := client.ValidateResponses(tt.expectedFilePath, tt.actualResponse)

			// Then
			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
			}
		})
	}
}
