package restclient_test

import (
	"net/http"
	"testing"
)

// TestValidateResponses_Headers_Matching verifies the validator passes when the actual

func TestValidateResponses_Headers_Matching(t *testing.T) {
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
}

// TestValidateResponses_Headers_MismatchingValue verifies the validator reports a header
// value mismatch when an actual header value differs from the expected.
func TestValidateResponses_Headers_MismatchingValue(t *testing.T) {
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
}

// TestValidateResponses_Headers_MissingExpected verifies the validator reports a
// missing-expected-header error when an expected header key is absent from the actual.
func TestValidateResponses_Headers_MissingExpected(t *testing.T) {
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
}

// TestValidateResponses_Headers_ExtraActualIgnored verifies that extra actual headers
// (not present in the expected file) are silently ignored.
func TestValidateResponses_Headers_ExtraActualIgnored(t *testing.T) {
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
}

// TestValidateResponses_Headers_MultiValueOrderPreserved verifies multi-value headers match
// when the values appear in the same order in both expected and actual.
func TestValidateResponses_Headers_MultiValueOrderPreserved(t *testing.T) {
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
}

// TestValidateResponses_Headers_MultiValueDifferentOrder verifies multi-value headers match
// even when the values appear in a different order in actual vs expected.
func TestValidateResponses_Headers_MultiValueDifferentOrder(t *testing.T) {
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
}

// TestValidateResponses_Headers_MultiValueDifferentValue verifies the validator reports a
// missing-expected-value error when one multi-value differs.
func TestValidateResponses_Headers_MultiValueDifferentValue(t *testing.T) {
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
}

// TestValidateResponses_Headers_MultiValueActualHasMore verifies that a subset relationship
// (actual has more values than expected) passes validation.
func TestValidateResponses_Headers_MultiValueActualHasMore(t *testing.T) {
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
}

// TestValidateResponses_Headers_CaseInsensitiveKeyMatching verifies header keys are matched
// case-insensitively between expected and actual.
func TestValidateResponses_Headers_CaseInsensitiveKeyMatching(t *testing.T) {
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
}

// TestValidateResponses_HeadersContain_NotTriggeredMatching verifies the HeadersContain
// logic is benign when the expected response comes from a file and other fields match.
func TestValidateResponses_HeadersContain_NotTriggeredMatching(t *testing.T) {
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
}

// TestValidateResponses_HeadersContain_MismatchOnStandardHeader verifies the validator
// reports a header-value mismatch on a standard header (Content-Type) when HeadersContain
// is not triggered by the file format.
func TestValidateResponses_HeadersContain_MismatchOnStandardHeader(t *testing.T) {
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
}

// TestValidateResponses_HeadersContain_ExpectedKeyNotFound verifies the validator reports
// a missing-expected-header error when the expected Content-Type key is absent and the
func TestValidateResponses_HeadersContain_ExpectedKeyNotFound(t *testing.T) {
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
}
