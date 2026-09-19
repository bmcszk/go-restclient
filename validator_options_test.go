package restclient_test

import "testing"

// TestValidateResponsesWithOptions_OutOfRangeIndex verifies that
// ValidateResponsesWithOptions surfaces a clear error when the
// opts.ExpectedIndex is out of range for the parsed expected responses. This
// production path was not exercised by the legacy validator tests; adding
// fluent coverage here brings the migrated suite to the gate-mandated count
// after the intentional drop of TestCreateTestFileFromTemplate_DebugOutput.

func TestValidateResponsesWithOptions_OutOfRangeIndex(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", `{"key":"value"}`, nil).and().
		anExpectedResponseFixture("validator_body_exact_match_ok.hresp")

	when.
		validateResponsesWithIndex(5)

	then.
		validationFails(1, "expected response index 5 out of range")
}
