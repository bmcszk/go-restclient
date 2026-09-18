package restclient_test

import "testing"

// TestValidateResponses_Body_ExactMatch verifies exact-body comparison when
// the expected body comes from a .hresp fixture. None of these bodies are
// JSON, so the validator emits plain "body mismatch" on failures (no JSON
// whitespace handling kicks in). Subtest names kept byte-identical.
func TestValidateResponses_Body_ExactMatch(t *testing.T) {
	t.Run("matching body", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"key":"value"}`, nil).and().
			anExpectedResponseFixture("validator_body_exact_match_ok.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("mismatching body", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "Hello Go", nil).and().
			anExpectedResponseFixture("validator_body_exact_match_ok.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "body mismatch")
	})

	t.Run("empty body in file, actual has content", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "Hello World", nil).and().
			anExpectedResponseFixture("validator_body_exact_no_body_exp.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "body mismatch")
	})

	t.Run("empty body in file, actual also empty", func(t *testing.T) {
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
}

// TestValidateResponses_BodyContains verifies the BodyContains path is benign
// when the expected response comes from a file (the file format cannot
// express BodyContains). Subtest names kept byte-identical.
func TestValidateResponses_BodyContains(t *testing.T) {
	t.Run("BodyContains logic is not triggered by file (positive case)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "Hello World Wide Web", nil).and().
			anExpectedResponseFixture("validator_bodycontains_positive.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("BodyContains logic not triggered, body mismatch handled by exact check", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "Hello World", nil).and().
			anExpectedResponseFixture("validator_bodycontains_mismatch.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "body mismatch")
	})
}

// TestValidateResponses_BodyNotContains verifies the BodyNotContains path is
// benign when the expected response comes from a file. Subtest names kept
// byte-identical.
func TestValidateResponses_BodyNotContains(t *testing.T) {
	t.Run("BodyNotContains logic is not triggered by file (positive case)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "Hello World", nil).and().
			anExpectedResponseFixture("validator_bodynotcontains_exact_mismatch.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("BodyNotContains logic not triggered, actual contains something, "+
		"file expects different exact body", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "Hello Universe", nil).and().
			anExpectedResponseFixture("validator_bodynotcontains_exact_mismatch.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "body mismatch")
	})
}
