package restclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// ExpectedResponse is defined in response.go

// ValidateResponse compares an actual Response against an ExpectedResponse.
// It returns a list of validation errors, or nil if everything matches.
func ValidateResponse(actual *Response, expected *ExpectedResponse) []error {
	var validationErrors []error

	if actual == nil {
		validationErrors = append(validationErrors, errors.New("actual response is nil"))
		return validationErrors
	}
	if expected == nil {
		validationErrors = append(validationErrors, errors.New("expected response is nil"))
		return validationErrors
	}

	// 1. Validate Status Code
	if expected.StatusCode != nil {
		if actual.StatusCode != *expected.StatusCode {
			validationErrors = append(validationErrors, fmt.Errorf("status code mismatch: expected %d, got %d", *expected.StatusCode, actual.StatusCode))
		}
	}

	// 2. Validate Status (string, e.g., "200 OK") - less common to validate precisely but can be useful
	if expected.Status != nil {
		if *expected.Status != "" {
			if actual.Status != *expected.Status {
				validationErrors = append(validationErrors, fmt.Errorf("status string mismatch: expected '%s', got '%s'", *expected.Status, actual.Status))
			}
		}
	}

	// 3. Validate Headers
	for expectedHeaderName, expectedHeaderValues := range expected.Headers {
		actualHeaderValues, ok := actual.Headers[http.CanonicalHeaderKey(expectedHeaderName)]
		if !ok {
			validationErrors = append(validationErrors, fmt.Errorf("expected header '%s' not found in actual response", expectedHeaderName))
			continue
		}
		// For now, we check if all expected values for a header are present in the actual header values.
		// This means actual can have more values, but not fewer for the ones we expect.
		// TODO: Consider options for exact match of header values counts or specific ordering if necessary.
		for _, ehv := range expectedHeaderValues {
			found := false
			for _, ahv := range actualHeaderValues {
				if ahv == ehv {
					found = true
					break
				}
			}
			if !found {
				validationErrors = append(validationErrors, fmt.Errorf("expected value '%s' for header '%s' not found in actual values: %v", ehv, expectedHeaderName, actualHeaderValues))
			}
		}
	}

	// 4. Validate Body (exact match)
	if expected.Body != nil {
		if actual.BodyString != *expected.Body {
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(*expected.Body),
				B:        difflib.SplitLines(actual.BodyString),
				FromFile: "Expected Body",
				ToFile:   "Actual Body",
				Context:  3,
			}
			diffText, _ := difflib.GetUnifiedDiffString(diff)
			validationErrors = append(validationErrors, fmt.Errorf("body mismatch:\n%s", diffText))
		}
	}

	// 5. Validate BodyContains
	for _, substr := range expected.BodyContains {
		if !strings.Contains(actual.BodyString, substr) {
			validationErrors = append(validationErrors, fmt.Errorf("actual body does not contain expected substring: '%s'", substr))
		}
	}

	// 6. Validate BodyNotContains
	for _, substr := range expected.BodyNotContains {
		if strings.Contains(actual.BodyString, substr) {
			validationErrors = append(validationErrors, fmt.Errorf("actual body contains unexpected substring: '%s'", substr))
		}
	}

	return validationErrors
}

// Helper to load ExpectedResponse from a JSON file (example)
func LoadExpectedResponseFromJSONFile(filePath string) (*ExpectedResponse, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read expected response file %s: %w", filePath, err)
	}
	exp := &ExpectedResponse{}
	if err := json.Unmarshal(data, exp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal expected response file %s: %w", filePath, err)
	}
	return exp, nil
}

// TODO: Add LoadExpectedResponseFromYAMLFile
// TODO: Add LoadExpectedResponseFromHTTPFile (parsing a simplified .http format for expected responses)
