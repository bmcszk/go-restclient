package restclient_test

import (
	"net/http"
	"testing"

	rc "github.com/bmcszk/go-restclient"

	// Imported to ensure assertMultierrorContains compiles
	"github.com/stretchr/testify/assert"
	// No fmt needed
	// No strings needed
)

type validateResponsesWithSampleFileTestCase struct {
	name               string
	actualModifier     func(actual *rc.Response)
	expectedFileSource string
	expectedErrCount   int
	expectedErrTexts   []string
}

func runValidateResponsesWithSampleFileSubtest(t *testing.T, tc validateResponsesWithSampleFileTestCase, baseActual *rc.Response) {
	t.Helper()
	// Given: A modified actual response based on baseActual and tc.actualModifier
	actualTest := &rc.Response{
		StatusCode: baseActual.StatusCode,
		Status:     baseActual.Status,
		Headers:    make(http.Header),
		BodyString: baseActual.BodyString,
	}
	for k, v := range baseActual.Headers { // Deep copy headers
		actualTest.Headers[k] = append([]string{}, v...)
	}
	tc.actualModifier(actualTest)

	currentExpectedFilePath := tc.expectedFileSource
	client, _ := rc.NewClient()

	// When
	err := client.ValidateResponses(currentExpectedFilePath, actualTest)

	// Then
	if tc.expectedErrCount == 0 {
		assert.NoError(t, err)
	} else {
		assertMultierrorContains(t, err, tc.expectedErrCount, tc.expectedErrTexts)
	}
}

func setupBaseActualResponseForSampleFileTests() (string, *rc.Response) {
	sampleFilePath := "testdata/http_response_files/sample1.http"

	expectedStatusCode := 200
	expectedStatus := "200 OK"
	expectedBody := `{
  "userId": 1,
  "id": 1,
  "title": "delectus aut autem",
  "completed": false
}`
	expectedHeaders := http.Header{
		"Content-Type": {"application/json; charset=utf-8"},
		"Date":         {"Tue, 27 May 2025 20:05:38 GMT"}, // Example date, actual can vary
	}

	sampleExpectedStruct := &rc.ExpectedResponse{
		StatusCode: &expectedStatusCode,
		Status:     &expectedStatus,
		Headers:    expectedHeaders,
		Body:       &expectedBody,
	}

	baseActual := &rc.Response{
		StatusCode: *sampleExpectedStruct.StatusCode,
		Status:     *sampleExpectedStruct.Status,
		Headers:    make(http.Header),
		BodyString: *sampleExpectedStruct.Body,
	}
	for k, v := range sampleExpectedStruct.Headers { // Deep copy headers
		baseActual.Headers[k] = append([]string{}, v...)
	}
	return sampleFilePath, baseActual
}

func getValidateResponsesWithSampleFileTestCases(sampleFilePath string) []validateResponsesWithSampleFileTestCase {
	return []validateResponsesWithSampleFileTestCase{
		{
			name:               "perfect match with sample1.http",
			actualModifier:     func(actual *rc.Response) {},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   0,
		},
		{
			name: "status code mismatch",
			actualModifier: func(actual *rc.Response) {
				actual.StatusCode = 500
			},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   1,
			expectedErrTexts:   []string{"status code mismatch: expected 200, got 500"},
		},
		{
			name: "status string mismatch",
			actualModifier: func(actual *rc.Response) {
				actual.Status = "200 Everything is Fine"
			},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   1,
			expectedErrTexts:   []string{"status string mismatch: expected '200 OK', got '200 Everything is Fine'"},
		},
		{
			name: "header value mismatch for Content-Type",
			actualModifier: func(actual *rc.Response) {
				actual.Headers.Set("Content-Type", "text/plain")
			},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   1,
			expectedErrTexts:   []string{"expected value 'application/json; charset=utf-8' for header 'Content-Type' not found"},
		},
		{
			name: "missing expected header Date",
			actualModifier: func(actual *rc.Response) {
				actual.Headers.Del("Date")
			},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   1,
			expectedErrTexts:   []string{"expected header 'Date' not found"},
		},
		{
			name: "body mismatch",
			actualModifier: func(actual *rc.Response) {
				actual.BodyString = "{\"message\": \"this is not the sample body\"}"
			},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   1,
			expectedErrTexts:   []string{"body mismatch"},
		},
		{
			name:               "BodyContains logic not triggered by exact file match (positive case)",
			actualModifier:     func(actual *rc.Response) {},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   0,
		},
		{
			name:               "BodyContains logic not triggered, exact body mismatch from file",
			actualModifier:     func(actual *rc.Response) {},
			expectedFileSource: "testdata/http_response_files/validator_withsample_bodycontains_exactmismatch.hresp",
			expectedErrCount:   1,
			expectedErrTexts:   []string{"body mismatch"},
		},
		{
			name:               "BodyNotContains logic not triggered by exact file match (positive case)",
			actualModifier:     func(actual *rc.Response) {},
			expectedFileSource: sampleFilePath,
			expectedErrCount:   0,
		},
		{
			name:               "BodyNotContains logic not triggered, exact body mismatch from file (actual contains something unwanted by this hypothetical check)",
			actualModifier:     func(actual *rc.Response) { actual.BodyString = "{\"title\": \"delectus aut autem\"}" },
			expectedFileSource: "testdata/http_response_files/validator_withsample_bodynotcontains_exactmismatch.hresp",
			expectedErrCount:   1,
			expectedErrTexts:   []string{"body mismatch"},
		},
	}
}

func TestValidateResponses_WithSampleFile(t *testing.T) {
	sampleFilePath, baseActual := setupBaseActualResponseForSampleFileTests()
	tests := getValidateResponsesWithSampleFileTestCases(sampleFilePath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runValidateResponsesWithSampleFileSubtest(t, tt, baseActual)
		})
	}
}

func TestValidateResponses_PartialExpected(t *testing.T) {
	// Given: Test cases defined in 'tests' slice
	tests := []struct {
		name             string
		actualResponse   *rc.Response
		expectedFilePath string // Was expectedFileContent
		expectedErrCount int
		expectedErrTexts []string
	}{
		{
			name: "SCENARIO-LIB-009-005 Equiv: Expected file has only status code - match",
			actualResponse: &rc.Response{
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
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200", BodyString: ""},
			expectedFilePath: "testdata/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-009-005-003 Corrected: File has status code and empty body - actual body mismatch",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200", BodyString: "non-empty body"},
			expectedFilePath: "testdata/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"},
		},
		{
			name: "SCENARIO-LIB-009-005-004 Equiv: Expected file has only status code - status code mismatch",
			actualResponse: &rc.Response{
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
			actualResponse: &rc.Response{
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
			actualResponse: &rc.Response{
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
			actualResponse: &rc.Response{
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
			client, _ := rc.NewClient()

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
