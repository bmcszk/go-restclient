package restclient_test

import "testing"

// TestValidateResponses_Body_ExactMatch_Matching verifies exact-body comparison when
// the expected body comes from a .hresp fixture and matches the actual response.

func TestValidateResponses_Body_ExactMatch_Matching(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", `{"key":"value"}`, nil).and().
		anExpectedResponseFixture("validator_body_exact_match_ok.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_Body_ExactMatch_Mismatching verifies the validator reports a
// "body mismatch" error when the actual body differs from the .hresp expected body.
func TestValidateResponses_Body_ExactMatch_Mismatching(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", "Hello Go", nil).and().
		anExpectedResponseFixture("validator_body_exact_match_ok.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "body mismatch")
}

// TestValidateResponses_Body_ExactMatch_EmptyFileActualContent verifies a body mismatch
// when the .hresp expects an empty body but the actual has content.
func TestValidateResponses_Body_ExactMatch_EmptyFileActualContent(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", "Hello World", nil).and().
		anExpectedResponseFixture("validator_body_exact_no_body_exp.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "body mismatch")
}

// TestValidateResponses_Body_ExactMatch_EmptyFileActualEmpty verifies validation succeeds
// when both expected and actual bodies are empty.
func TestValidateResponses_Body_ExactMatch_EmptyFileActualEmpty(t *testing.T) {
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

// TestValidateResponses_BodyContains_NotTriggeredByFile verifies the BodyContains path
// is benign when the expected response comes from a file (the file format cannot
// express BodyContains).
func TestValidateResponses_BodyContains_NotTriggeredByFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", "Hello World Wide Web", nil).and().
		anExpectedResponseFixture("validator_bodycontains_positive.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_BodyContains_ExactMismatchHandled verifies that when BodyContains
// is not triggered by the file, body mismatch is still handled by the exact-match check.
func TestValidateResponses_BodyContains_ExactMismatchHandled(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", "Hello World", nil).and().
		anExpectedResponseFixture("validator_bodycontains_mismatch.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "body mismatch")
}

// TestValidateResponses_BodyNotContains_NotTriggeredByFile verifies the BodyNotContains path
// is benign when the expected response comes from a file.
func TestValidateResponses_BodyNotContains_NotTriggeredByFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", "Hello World", nil).and().
		anExpectedResponseFixture("validator_bodynotcontains_exact_mismatch.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_BodyNotContains_ActualContainsUnwanted verifies that when the
// expected body (file) does not match the actual body, validation reports a body mismatch
// regardless of BodyNotContains not being expressible in the file format.
func TestValidateResponses_BodyNotContains_ActualContainsUnwanted(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", "Hello Universe", nil).and().
		anExpectedResponseFixture("validator_bodynotcontains_exact_mismatch.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "body mismatch")
}
