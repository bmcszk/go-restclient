package restclient_test

import "testing"

func TestValidateResponses_StatusString_Matching(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", "", nil).and().
		anExpectedResponseFixture("validator_body_exact_no_body_exp.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_StatusString_Mismatching verifies the validator reports a
// "status string mismatch" when the reason phrase differs from the .hresp expected.
func TestValidateResponses_StatusString_Mismatching(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 Something Else", "", nil).and().
		anExpectedResponseFixture("validator_body_exact_no_body_exp.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "status string mismatch: expected '200 OK', got '200 Something Else'")
}

// TestValidateResponses_StatusString_ActualCorrectFileOnlyCode verifies a status string
// mismatch when the actual status string is "200 OK" but the .hresp expects only "200".
func TestValidateResponses_StatusString_ActualCorrectFileOnlyCode(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", "", nil).and().
		anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "status string mismatch: expected '200', got '200 OK'")
}

// TestValidateResponses_StatusString_MismatchingBoth verifies both status code and status
// string mismatches are reported when both fields differ.
func TestValidateResponses_StatusString_MismatchingBoth(t *testing.T) {
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
}

// TestValidateResponses_StatusString_CodeOnlyBothSides verifies validation succeeds when
// both the expected (.hresp) and actual responses carry only a status code without a reason
// phrase.
func TestValidateResponses_StatusString_CodeOnlyBothSides(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200", "", nil).and().
		anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_StatusCode_Matching verifies the numeric status code alone is
// sufficient for validation when both actual and expected carry only the code.
func TestValidateResponses_StatusCode_Matching(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWithStatus(200, "200").and().
		anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_StatusCode_Mismatching verifies the validator reports two errors
// (code + string) when the numeric status code does not match.
func TestValidateResponses_StatusCode_Mismatching(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWithStatus(404, "404").and().
		anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

	when.
		validateResponses()

	then.
		validationFails(2)
}

// TestValidateResponses_StatusCode_NilActual verifies the validator reports code + string
// mismatches when the actual status code is 0 (the "nil" case, which should not happen with
// a real http.Response).
func TestValidateResponses_StatusCode_NilActual(t *testing.T) {
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
}

// TestValidateResponses_StatusCode_NilExpected verifies a parse failure plus count mismatch
// when the .hresp expected file has no status line.
func TestValidateResponses_StatusCode_NilExpected(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWithStatus(200, "200").and().
		anExpectedResponseFixture("validator_status_code_nil_expected.hresp")

	when.
		validateResponses()

	then.
		validationFails(2,
			"failed to parse expected response file",
			"mismatch in number of responses: got 1 actual, but expected 0")
}

// TestValidateResponses_StatusCode_BothCodeOnly verifies validation succeeds when both
// sides carry only the numeric status code and no reason phrase.
func TestValidateResponses_StatusCode_BothCodeOnly(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWithStatus(200, "200").and().
		anExpectedResponseFixture("validator_status_code_only.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}
