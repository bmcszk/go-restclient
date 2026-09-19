package restclient_test

import "testing"

// TestValidateResponsesWithOptions_OutOfRangeIndex covers the out-of-range ExpectedIndex path
// that the legacy validator tests never exercised.

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
