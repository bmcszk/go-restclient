package restclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pmezard/go-difflib/difflib"
)

// ExpectedResponse is defined in response.go

// ValidateResponses compares a slice of actual HTTP responses against a set of expected responses
// parsed from the specified file. It returns a consolidated error (multierror) if any
// discrepancies are found, or nil if all validations pass.
func ValidateResponses(ctx context.Context, responseFilePath string, actualResponses ...*Response) error {
	var errs *multierror.Error

	// Attempt to parse the expected responses from the file.
	expectedResponses, parseErr := parseExpectedResponseFile(responseFilePath)
	if parseErr != nil {
		errs = multierror.Append(errs, fmt.Errorf("failed to parse expected response file '%s': %w", responseFilePath, parseErr))
	}

	// Determine effective counts for actual and expected responses.
	effectiveNumActual := countNonNilActuals(actualResponses)
	effectiveNumExpected := 0
	if expectedResponses != nil {
		effectiveNumExpected = len(expectedResponses)
	}

	// Check for count mismatch.
	if effectiveNumActual != effectiveNumExpected {
		errs = multierror.Append(errs, fmt.Errorf("mismatch in number of responses: got %d actual, but expected %d from file '%s'", effectiveNumActual, effectiveNumExpected, responseFilePath))
	}

	// If parsing failed, we cannot proceed with per-response validation. Return collected errors so far.
	if parseErr != nil {
		return errs.ErrorOrNil()
	}

	// If there was no parse error, but counts mismatched, return the count mismatch error.
	if effectiveNumActual != effectiveNumExpected { // This implies parseErr was nil to reach here
		return errs.ErrorOrNil()
	}

	// If we reach here, parsing succeeded and counts match. Proceed to validate pairs.
	for i := 0; i < effectiveNumActual; i++ {
		actual := actualResponses[i]
		expected := expectedResponses[i]

		if actual == nil {
			errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): actual response is nil", i+1, responseFilePath))
			continue
		}

		// Validate Status Code
		if expected.StatusCode != nil && (actual.StatusCode != *expected.StatusCode) {
			errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): status code mismatch: expected %d, got %d", i+1, responseFilePath, *expected.StatusCode, actual.StatusCode))
		}

		// Validate Status String
		if expected.Status != nil && *expected.Status != "" && (actual.Status != *expected.Status) {
			errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): status string mismatch: expected '%s', got '%s'", i+1, responseFilePath, *expected.Status, actual.Status))
		}

		// Validate Headers (Exact Match for specified keys)
		if expected.Headers != nil {
			for key, expectedValues := range expected.Headers {
				actualValues, ok := actual.Headers[key]
				if !ok {
					errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): expected header '%s' not found", i+1, responseFilePath, key))
					continue
				}
				for _, ev := range expectedValues {
					found := false
					for _, av := range actualValues {
						if av == ev {
							found = true
							break
						}
					}
					if !found {
						errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): expected value '%s' for header '%s' not found in actual values %v", i+1, responseFilePath, ev, key, actualValues))
					}
				}
			}
		}

		// Validate HeadersContain
		if expected.HeadersContain != nil {
			for key, expectedValues := range expected.HeadersContain {
				actualValues, ok := actual.Headers[key]
				if !ok {
					errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): expected header '%s' (for HeadersContain) not found", i+1, responseFilePath, key))
					continue
				}
				for _, ev := range expectedValues {
					found := false
					for _, av := range actualValues {
						if av == ev {
							found = true
							break
						}
					}
					if !found {
						errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): expected value '%s' for header '%s' (for HeadersContain) not found in actual values %v", i+1, responseFilePath, ev, key, actualValues))
					}
				}
			}
		}

		// Validate Body (Exact Match)
		if expected.Body != nil {
			normalizedExpectedBody := strings.TrimSpace(strings.ReplaceAll(*expected.Body, "\r\n", "\n"))
			normalizedActualBody := strings.TrimSpace(strings.ReplaceAll(actual.BodyString, "\r\n", "\n"))

			if normalizedActualBody != normalizedExpectedBody {
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(normalizedExpectedBody),
					B:        difflib.SplitLines(normalizedActualBody),
					FromFile: "Expected Body",
					ToFile:   "Actual Body",
					Context:  3,
				}
				diffText, _ := difflib.GetUnifiedDiffString(diff)
				errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): body mismatch:\n%s", i+1, responseFilePath, diffText))
			}
		}

		// Validate BodyContains
		if len(expected.BodyContains) > 0 {
			normalizedActualBody := strings.TrimSpace(strings.ReplaceAll(actual.BodyString, "\r\n", "\n"))
			for _, sub := range expected.BodyContains {
				normalizedSub := strings.TrimSpace(strings.ReplaceAll(sub, "\r\n", "\n"))
				if !strings.Contains(normalizedActualBody, normalizedSub) {
					errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): actual body does not contain expected substring: '%s'", i+1, responseFilePath, sub))
				}
			}
		}

		// Validate BodyNotContains
		if len(expected.BodyNotContains) > 0 {
			normalizedActualBody := strings.TrimSpace(strings.ReplaceAll(actual.BodyString, "\r\n", "\n"))
			for _, sub := range expected.BodyNotContains {
				normalizedSub := strings.TrimSpace(strings.ReplaceAll(sub, "\r\n", "\n"))
				if strings.Contains(normalizedActualBody, normalizedSub) {
					errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): actual body contains unexpected substring: '%s'", i+1, responseFilePath, sub))
				}
			}
		}
	}

	return errs.ErrorOrNil()
}

// Helper function to count non-nil actual responses
func countNonNilActuals(responses []*Response) int {
	count := 0
	for _, r := range responses {
		if r != nil {
			count++
		}
	}
	return count
}

// TODO: Add LoadExpectedResponseFromHTTPFile (parsing a simplified .http format for expected responses)
