package restclient

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// ValidateResponses compares actual HTTP responses against a set of expected responses
// parsed from the specified .hresp file. It leverages the client's configuration for variable substitution.
// The `actualResponses` parameter is variadic, allowing zero or more responses to be passed.
//
// As a method on the `Client`, it uses `c.programmaticVars` for programmatic variables and the client instance `c`
// itself for resolving system variables (e.g., {{$uuid}}) within the .hresp content.
// Variables can also be defined in the .hresp file using `@name = value` syntax.
// The precedence for variable resolution is detailed in `variables.go:resolveVariablesInText`.
//
// It returns a consolidated error (multierror) if any discrepancies are found (e.g., status mismatch,
// header mismatch, body mismatch, or count mismatch between actual and expected responses), or nil
// if all validations pass. Errors during file reading, @define extraction, variable substitution, or
// .hresp parsing are also returned.
func (c *Client) ValidateResponses(responseFilePath string, actualResponses ...*Response) error {
	expectedResponses, errs, parseErr := c.loadAndParseExpectedResponses(responseFilePath)

	// If there was a critical error (file not found, etc.), return immediately
	if parseErr != nil && errs == nil {
		return parseErr
	}

	// Continue with validation even if parsing failed, but use empty expected responses
	if parseErr != nil {
		expectedResponses = nil
	}

	errs = c.validateResponseCounts(responseFilePath, actualResponses, expectedResponses, errs)
	errs = c.validateResponsePairs(responseFilePath, actualResponses, expectedResponses, errs)
	return errs.ErrorOrNil()
}

// ValidateOptions holds optional parameters for ValidateResponses.
type ValidateOptions struct {
	ExpectedName  string // expected response name (not yet supported)
	ExpectedIndex int    // expected response 0-based index
}

// ValidateResponsesWithOptions validates with additional options.
func (c *Client) ValidateResponsesWithOptions(
	responseFilePath string,
	opts ValidateOptions,
	actualResponses ...*Response,
) error {
	expectedResponses, errs, parseErr := c.loadAndParseExpectedResponses(responseFilePath)

	// If there was a critical error (file not found, etc.), return immediately
	if parseErr != nil && errs == nil {
		return parseErr
	}

	// Continue with validation even if parsing failed, but use empty expected responses
	if parseErr != nil {
		expectedResponses = nil
	}

	// If selecting a specific expected response by name or index, filter
	if opts.ExpectedName != "" || opts.ExpectedIndex >= 0 {
		expectedResponses, selectErr := selectExpectedResponse(expectedResponses, opts.ExpectedName, opts.ExpectedIndex)
		if selectErr != nil {
			return selectErr
		}
		// When selecting specific response, only validate that one
		errs = c.validateSingleResponsePair(responseFilePath, actualResponses, expectedResponses, errs)
		return errs.ErrorOrNil()
	}

	errs = c.validateResponseCounts(responseFilePath, actualResponses, expectedResponses, errs)
	errs = c.validateResponsePairs(responseFilePath, actualResponses, expectedResponses, errs)
	return errs.ErrorOrNil()
}

// selectExpectedResponse filters expected responses by name or index.
func selectExpectedResponse(
	expectedResponses []*ExpectedResponse,
	expectedName string,
	expectedIndex int,
) ([]*ExpectedResponse, error) {
	if expectedIndex >= 0 {
		if expectedIndex < 0 || expectedIndex >= len(expectedResponses) {
			return nil, fmt.Errorf(
				"expected response index %d out of range (0-%d)",
				expectedIndex, len(expectedResponses)-1)
		}
		return []*ExpectedResponse{expectedResponses[expectedIndex]}, nil
	}

	// expectedName not supported yet - .hresp format doesn't have named blocks
	if expectedName != "" {
		return nil, errors.New(
			"-e-name not yet supported; .hresp format does not name response blocks")
	}

	return expectedResponses, nil
}

// validateSingleResponsePair validates a single actual response against expected responses.
func (c *Client) validateSingleResponsePair(
	responseFilePath string,
	actualResponses []*Response,
	expectedResponses []*ExpectedResponse,
	errs *multierror.Error,
) *multierror.Error {
	if len(actualResponses) == 0 || actualResponses[0] == nil {
		errs = multierror.Append(errs, errors.New(
			"validation: actual response is nil"))
		return errs
	}
	if len(expectedResponses) == 0 {
		errs = multierror.Append(errs, errors.New(
			"validation: no expected response found"))
		return errs
	}
	return c.validateSingleResponse(responseFilePath, 1, actualResponses[0], expectedResponses[0], errs)
}

func (c *Client) loadAndParseExpectedResponses(
	responseFilePath string) ([]*ExpectedResponse, *multierror.Error, error) {
	var errs *multierror.Error

	hrespFileContent, err := os.ReadFile(responseFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read expected response file %s: %w", responseFilePath, err)
	}

	fileVars, contentWithoutDefines, err := extractHrespDefines(string(hrespFileContent))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract @defines from %s: %w", responseFilePath, err)
	}

	// Plain @define keys are prefixed so fileScopedVars match the main engine's "@name" lookup.
	atFileVars := make(map[string]string, len(fileVars))
	for name, value := range fileVars {
		atFileVars["@"+name] = value
	}
	rctx := c.directiveResolveContext(nil, &Request{ActiveVariables: atFileVars}, os.LookupEnv,
		c.generateRequestScopedSystemVariables(), nil, -1)
	substitutedContent := resolveVariablesInText(contentWithoutDefines, rctx)
	substitutedContent = substituteDynamicSystemVariables(
		substitutedContent, c.currentDotEnvVars, c.programmaticVars)

	expectedResponses, parseErr := parseExpectedResponses(strings.NewReader(substitutedContent), responseFilePath)
	if parseErr != nil {
		errs = multierror.Append(errs, fmt.Errorf(
			"failed to parse expected response file '%s' after variable substitution: %w",
			responseFilePath, parseErr))
		return nil, errs, parseErr
	}

	return expectedResponses, nil, nil
}

func (*Client) validateResponseCounts(responseFilePath string, actualResponses []*Response,
	expectedResponses []*ExpectedResponse, errs *multierror.Error) *multierror.Error {
	effectiveNumActual := countNonNilActuals(actualResponses)
	effectiveNumExpected := 0
	if expectedResponses != nil {
		effectiveNumExpected = len(expectedResponses)
	}

	if effectiveNumActual != effectiveNumExpected {
		errs = multierror.Append(errs, fmt.Errorf(
			"mismatch in number of responses: got %d actual, but expected %d from file '%s'",
			effectiveNumActual, effectiveNumExpected, responseFilePath))
	}

	return errs
}

func (c *Client) validateResponsePairs(responseFilePath string, actualResponses []*Response,
	expectedResponses []*ExpectedResponse, errs *multierror.Error) *multierror.Error {
	effectiveNumActual := countNonNilActuals(actualResponses)
	effectiveNumExpected := 0
	if expectedResponses != nil {
		effectiveNumExpected = len(expectedResponses)
	}

	// Only validate pairs where both actual and expected responses exist
	maxPairs := effectiveNumActual
	if effectiveNumExpected < maxPairs {
		maxPairs = effectiveNumExpected
	}

	for i := 0; i < maxPairs; i++ {
		actual := actualResponses[i]
		expected := expectedResponses[i]

		if actual == nil {
			errs = multierror.Append(errs, fmt.Errorf(
				"validation for response #%d ('%s'): actual response is nil",
				i+1, responseFilePath))
			continue
		}

		errs = c.validateSingleResponse(responseFilePath, i+1, actual, expected, errs)
	}

	return errs
}

func (c *Client) validateSingleResponse(responseFilePath string, responseIndex int,
	actual *Response, expected *ExpectedResponse, errs *multierror.Error) *multierror.Error {
	errs = c.validateStatusCode(responseFilePath, responseIndex, actual, expected, errs)
	errs = c.validateStatusString(responseFilePath, responseIndex, actual, expected, errs)
	errs = c.validateHeaders(responseFilePath, responseIndex, actual, expected, errs)
	errs = c.validateBody(responseFilePath, responseIndex, actual, expected, errs)
	return errs
}

func (*Client) validateStatusCode(responseFilePath string, responseIndex int,
	actual *Response, expected *ExpectedResponse, errs *multierror.Error) *multierror.Error {
	if expected.StatusCode != nil && (actual.StatusCode != *expected.StatusCode) {
		errs = multierror.Append(errs, fmt.Errorf(
			"validation for response #%d ('%s'): status code mismatch: expected %d, got %d",
			responseIndex, responseFilePath, *expected.StatusCode, actual.StatusCode))
	}
	return errs
}

func (*Client) validateStatusString(responseFilePath string, responseIndex int,
	actual *Response, expected *ExpectedResponse, errs *multierror.Error) *multierror.Error {
	if expected.Status != nil && *expected.Status != "" && (actual.Status != *expected.Status) {
		errs = multierror.Append(errs, fmt.Errorf(
			"validation for response #%d ('%s'): status string mismatch: expected '%s', got '%s'",
			responseIndex, responseFilePath, *expected.Status, actual.Status))
	}
	return errs
}

func (c *Client) validateHeaders(responseFilePath string, responseIndex int,
	actual *Response, expected *ExpectedResponse, errs *multierror.Error) *multierror.Error {
	if expected.Headers == nil {
		return errs
	}

	for key, expectedValues := range expected.Headers {
		actualValues, ok := actual.Headers[key]
		if !ok {
			errs = multierror.Append(errs, fmt.Errorf(
				"validation for response #%d ('%s'): expected header '%s' not found",
				responseIndex, responseFilePath, key))
			continue
		}

		errs = c.validateHeaderValues(responseFilePath, responseIndex, key, expectedValues, actualValues, errs)
	}

	return errs
}

func (*Client) validateHeaderValues(responseFilePath string, responseIndex int, key string,
	expectedValues, actualValues []string, errs *multierror.Error) *multierror.Error {
	for _, ev := range expectedValues {
		if !isHeaderValuePresent(ev, actualValues) {
			errs = multierror.Append(errs, fmt.Errorf(
				"validation for response #%d ('%s'): expected value '%s' for "+
					"header '%s' not found in actual values %v",
				responseIndex, responseFilePath, ev, key, actualValues))
		}
	}
	return errs
}

// isHeaderValuePresent checks if an expected header value is present in the actual values.
func isHeaderValuePresent(expectedValue string, actualValues []string) bool {
	for _, av := range actualValues {
		if av == expectedValue {
			return true
		}
	}
	return false
}

// countNonNilActuals counts non-nil responses in a slice.
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
