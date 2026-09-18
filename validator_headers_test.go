package restclient_test

import (
	"net/http"
	"testing"
)

// TestValidateResponses_Headers verifies the validator's handling of HTTP
// response headers: single-value matching, multi-value matching (and order
// independence), missing keys, extra actual headers, and case-insensitive
// header-key matching. Subtest names are kept byte-identical with the legacy
// implementation.
func TestValidateResponses_Headers(t *testing.T) {
	t.Run("matching headers", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"123"}}).and().
			anExpectedResponseFixture("validator_headerscontain_key_missing.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("mismatching header value", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"Content-Type": {"text/html"}}).and().
			anExpectedResponseFixture("validator_headerscontain_key_missing.hresp")

		when.
			validateResponses()

		then.
			validationFails(1,
				"expected value 'application/json' for header 'Content-Type' not found")
	})

	t.Run("missing expected header", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"X-Other": {"value"}}).and().
			anExpectedResponseFixture("validator_headers_missing_exp.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "expected header 'X-Custom-Header' not found")
	})

	t.Run("extra actual header (should be ignored)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"Content-Type": {"application/json"}, "X-Extra": {"ignored"}}).and().
			anExpectedResponseFixture("validator_headerscontain_key_missing.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("matching multi-value headers (order preserved)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"Accept": {"application/json", "text/xml"}}).and().
			anExpectedResponseFixture("validator_headers_multival_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("mismatching multi-value headers (different order)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"Accept": {"application/json", "text/xml"}}).and().
			anExpectedResponseFixture("validator_headers_multival_mismatch_order.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("mismatching multi-value headers (different value)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"Accept": {"application/json", "application/pdf"}}).and().
			anExpectedResponseFixture("validator_headers_multival_match.hresp")

		when.
			validateResponses()

		then.
			validationFails(1,
				"expected value 'text/xml' for header 'Accept' not found")
	})

	t.Run("subset of multi-value headers (actual has more values)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"Accept": {"application/json", "text/xml", "application/pdf"}}).and().
			anExpectedResponseFixture("validator_headers_multival_subset.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("case-insensitive header key matching", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_headers_case_insensitive_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})
}

// TestValidateResponses_HeadersContain verifies that the HeadersContain logic in
// ValidateResponses is benign when the expected response comes from a file, as
// the file cannot specify HeadersContain requirements. Subtest names kept
// byte-identical with the legacy implementation.
func TestValidateResponses_HeadersContain(t *testing.T) {
	t.Run("HeadersContain logic not triggered (matching case for other fields)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"Content-Type": {"application/json; charset=utf-8"}}).and().
			anExpectedResponseFixture("validator_headerscontain_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("HeadersContain logic not triggered (mismatch on standard header)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", "",
				http.Header{"Content-Type": {"text/html"}}).and().
			anExpectedResponseFixture("validator_headerscontain_key_missing.hresp")

		when.
			validateResponses()

		then.
			validationFails(1,
				"expected value 'application/json' for header 'Content-Type' not found")
	})

	t.Run("HeadersContain logic not triggered (expected header key not found by standard check)",
		func(t *testing.T) {
			given, when, then := newParts(t)

			given.
				aClient().and().
				aResponseWith(200, "200 OK", "",
					http.Header{"X-Other": {"value"}}).and().
				anExpectedResponseFixture("validator_headerscontain_key_missing.hresp")

			when.
				validateResponses()

			then.
				validationFails(1, "expected header 'Content-Type' not found")
		})
}
