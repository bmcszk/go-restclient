package restclient_test

import (
	"fmt"
	"testing"
)

// TestValidateResponses_StatusString verifies the validator's handling of the
// HTTP status-line reason phrase. Each subtest builds an actual response by
// hand and validates against a fixed fixture. Subtest names are kept
// byte-identical with the legacy implementation.
func TestValidateResponses_StatusString(t *testing.T) {
	t.Run("matching status string", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "", nil).and().
			anExpectedResponseFixture("validator_body_exact_no_body_exp.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("mismatching status string", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 Something Else", "", nil).and().
			anExpectedResponseFixture("validator_body_exact_no_body_exp.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "status string mismatch: expected '200 OK', got '200 Something Else'")
	})

	t.Run("actual status string is correct, expected file has only status code", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "", nil).and().
			anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "status string mismatch: expected '200', got '200 OK'")
	})

	t.Run("mismatching status code, status strings also mismatch", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(404, "404 Not Found", "", nil).and().
			anExpectedResponseFixture("validator_body_exact_no_body_exp.hresp")

		when.
			validateResponses()

		then.
			validationFails(2,
				"status code mismatch: expected 200, got 404",
				"status string mismatch: expected '200 OK', got '404 Not Found'")
	})

	t.Run("matching status code, expected file only code, actual also only code in status",
		func(t *testing.T) {
			given, when, then := newParts(t)

			given.
				aClient().and().
				aResponseWith(200, "200", "", nil).and().
				anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

			when.
				validateResponses()

			then.
				validationSucceeds()
		})
}

// TestValidateResponses_StatusCode verifies the validator's handling of the
// numeric HTTP status code alone. The actual response uses the integer
// formatted as its reason phrase so both the code and the string representation
// can be evaluated against each fixture.
func TestValidateResponses_StatusCode(t *testing.T) {
	t.Run("matching status code only", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWithStatus(200, fmt.Sprintf("%d", 200)).and().
			anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("mismatching status code only", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWithStatus(404, fmt.Sprintf("%d", 404)).and().
			anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

		when.
			validateResponses()

		then.
			validationFails(2)
	})

	t.Run("nil actual status code (should not happen with real http.Response)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWithStatus(0, "0").and().
			anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

		when.
			validateResponses()

		then.
			validationFails(2,
				"status code mismatch: expected 200, got 0",
				"status string mismatch: expected '200', got '0'")
	})

	t.Run("nil expected status code (file has no status line)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWithStatus(200, fmt.Sprintf("%d", 200)).and().
			anExpectedResponseFixture("validator_status_code_nil_expected.hresp")

		when.
			validateResponses()

		then.
			validationFails(2,
				"failed to parse expected response file",
				"mismatch in number of responses: got 1 actual, but expected 0")
	})

	t.Run("matching status code, actual and expected only have code", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWithStatus(200, fmt.Sprintf("%d", 200)).and().
			anExpectedResponseFixture("validator_status_code_only.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})
}
