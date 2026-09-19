package restclient_test

import (
	"net/http"
	"testing"
)

const validatorSamplePath = "test/data/http_response_files/sample1.http"

// TestValidateResponses_WithSampleFile_PerfectMatch verifies the validator passes against

func TestValidateResponses_WithSampleFile_PerfectMatch(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseFromRawHTTPFile(validatorSamplePath).and().
		anExpectedResponseFixture("sample1.http")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_WithSampleFile_StatusCodeMismatch verifies a "status code mismatch"
// error is reported when the actual status code differs from the .hresp expected.
func TestValidateResponses_WithSampleFile_StatusCodeMismatch(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseFromRawHTTPFile(validatorSamplePath).and().
		withStatusCode(500).and().
		anExpectedResponseFixture("sample1.http")

	when.
		validateResponses()

	then.
		validationFails(1, "status code mismatch: expected 200, got 500")
}

// TestValidateResponses_WithSampleFile_StatusStringMismatch verifies a "status string
// mismatch" error is reported when the actual reason phrase differs from the .hresp expected.
func TestValidateResponses_WithSampleFile_StatusStringMismatch(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseFromRawHTTPFile(validatorSamplePath).and().
		withStatusText("200 Everything is Fine").and().
		anExpectedResponseFixture("sample1.http")

	when.
		validateResponses()

	then.
		validationFails(1, "status string mismatch: expected '200 OK', got '200 Everything is Fine'")
}

// TestValidateResponses_WithSampleFile_HeaderValueMismatch verifies the validator reports
// a header-value mismatch for the Content-Type header.
func TestValidateResponses_WithSampleFile_HeaderValueMismatch(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseFromRawHTTPFile(validatorSamplePath).and().
		withHeader("Content-Type", "text/plain").and().
		anExpectedResponseFixture("sample1.http")

	when.
		validateResponses()

	then.
		validationFails(1,
			"expected value 'application/json; charset=utf-8' for header 'Content-Type' not found")
}

// TestValidateResponses_WithSampleFile_MissingExpectedHeaderDate verifies the validator
// reports a missing-expected-header error for the Date header.
func TestValidateResponses_WithSampleFile_MissingExpectedHeaderDate(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseFromRawHTTPFile(validatorSamplePath).and().
		withoutHeader("Date").and().
		anExpectedResponseFixture("sample1.http")

	when.
		validateResponses()

	then.
		validationFails(1, "expected header 'Date' not found")
}

func TestValidateResponses_WithSampleFile_BodyMismatch(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseFromRawHTTPFile(validatorSamplePath).and().
		withBody(`{"message": "this is not the sample body"}`).and().
		anExpectedResponseFixture("sample1.http")

	when.
		validateResponses()

	then.
		validationFails(1, "JSON content mismatch")
}

func TestValidateResponses_WithSampleFile_BodyContainsNotTriggeredPositive(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseFromRawHTTPFile(validatorSamplePath).and().
		anExpectedResponseFixture("sample1.http")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_WithSampleFile_BodyContainsExactMismatch verifies the validator
// reports a JSON content mismatch when BodyContains cannot rescue an exact-body mismatch
// from the file format.
func TestValidateResponses_WithSampleFile_BodyContainsExactMismatch(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseFromRawHTTPFile(validatorSamplePath).and().
		anExpectedResponseFixture("validator_withsample_bodycontains_exactmismatch.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "JSON content mismatch")
}

func TestValidateResponses_WithSampleFile_BodyNotContainsNotTriggeredPositive(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseFromRawHTTPFile(validatorSamplePath).and().
		anExpectedResponseFixture("sample1.http")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_WithSampleFile_BodyNotContainsExactMismatch verifies the validator
// reports a JSON content mismatch when BodyNotContains cannot rescue an exact-body
// mismatch from the file format.
func TestValidateResponses_WithSampleFile_BodyNotContainsExactMismatch(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseFromRawHTTPFile(validatorSamplePath).and().
		withBody(`{"title": "delectus aut autem"}`).and().
		anExpectedResponseFixture("validator_withsample_bodynotcontains_exactmismatch.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "JSON content mismatch")
}

// TestValidateResponses_PartialExpected_StatusCodeOnly_Match verifies the validator passes
// when the .hresp defines only the status code (no body, no extra headers) and the actual
func TestValidateResponses_PartialExpected_StatusCodeOnly_Match(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200", "",
			http.Header{"Content-Type": {"application/json"}}).and().
		anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_PartialExpected_StatusCodeEmptyBody_Match verifies validation
// succeeds when the .hresp defines status code + empty body and the actual matches.
func TestValidateResponses_PartialExpected_StatusCodeEmptyBody_Match(t *testing.T) {
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

// TestValidateResponses_PartialExpected_StatusCodeEmptyBody_Mismatch verifies a body
// mismatch is reported when the .hresp expects an empty body but the actual has content.
func TestValidateResponses_PartialExpected_StatusCodeEmptyBody_Mismatch(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200", "non-empty body", nil).and().
		anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "body mismatch")
}

// TestValidateResponses_PartialExpected_StatusCodeOnly_StatusMismatch verifies both code
// and string mismatches are reported when the status code differs and the .hresp expects
// only the code.
func TestValidateResponses_PartialExpected_StatusCodeOnly_StatusMismatch(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(404, "404", "",
			http.Header{"Content-Type": {"application/json"}}).and().
		anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

	when.
		validateResponses()

	then.
		validationFails(2,
			"status code mismatch: expected 200, got 404",
			"status string mismatch: expected '200', got '404'")
}

// TestValidateResponses_PartialExpected_HeadersOnly_Match verifies validation passes when
// the .hresp defines only specific headers and the actual matches.
func TestValidateResponses_PartialExpected_HeadersOnly_Match(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200", "",
			http.Header{"Content-Type": {"application/json"}, "X-Custom": {"val"}}).and().
		anExpectedResponseFixture("validator_partial_headers_key_missing.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_PartialExpected_HeadersOnly_ValueMismatch verifies a header-value
// mismatch is reported when an expected header value differs.
func TestValidateResponses_PartialExpected_HeadersOnly_ValueMismatch(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200", "",
			http.Header{"Content-Type": {"text/plain"}, "X-Custom": {"val"}}).and().
		anExpectedResponseFixture("validator_partial_headers_key_missing.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "expected value 'application/json' for header 'Content-Type' not found")
}

func TestValidateResponses_PartialExpected_HeadersOnly_HeaderMissing(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200", "",
			http.Header{"X-Other": {"value"}}).and().
		anExpectedResponseFixture("validator_partial_headers_key_missing.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "expected header 'Content-Type' not found")
}

func TestValidateResponses_NilAndEmptyActuals_NilSlice(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		noActualResponses().and().
		anExpectedResponseFixture("validator_nil_empty_actuals_expected.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "mismatch in number of responses: got 0 actual, but expected 1")
}

func TestValidateResponses_NilAndEmptyActuals_EmptySlice(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		anEmptyResponseSlice().and().
		anExpectedResponseFixture("validator_nil_empty_actuals_expected.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "mismatch in number of responses: got 0 actual, but expected 1")
}

func TestValidateResponses_NilAndEmptyActuals_SliceWithNil(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aNilResponse().and().
		anExpectedResponseFixture("validator_nil_empty_actuals_expected.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "mismatch in number of responses: got 0 actual, but expected 1")
}

// TestValidateResponses_FileErrors_MissingFile verifies the validator reports a "failed to
// read expected response file" error when the .hresp path does not exist.
func TestValidateResponses_FileErrors_MissingFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWithStatus(200, "").and().
		anExpectedResponseFileAt("nonexistent.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "failed to read expected response file", "nonexistent.hresp")
}

// TestValidateResponses_FileErrors_EmptyFile verifies a "mismatch in number of responses"
// error when the .hresp file is empty (parses to zero responses).
func TestValidateResponses_FileErrors_EmptyFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWithStatus(200, "").and().
		anExpectedResponseFixture("validator_empty_expected.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "mismatch in number of responses: got 1 actual, but expected 0")
}

// TestValidateResponses_FileErrors_MalformedStatus verifies the validator reports both a
// parse error (invalid status code) and a count mismatch when the .hresp is malformed.
func TestValidateResponses_FileErrors_MalformedStatus(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWithStatus(200, "").and().
		anExpectedResponseFixture("validator_malformed_status.hresp")

	when.
		validateResponses()

	then.
		validationFails(2,
			"failed to parse expected response file",
			"invalid status code",
			"mismatch in number of responses: got 1 actual, but expected 0")
}
