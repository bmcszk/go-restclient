package restclient_test

import (
	"net/http"
	"testing"
)

// TestValidateResponses_WithSampleFile verifies the validator against
// test/data/http_response_files/sample1.http. Each subtest loads the sample as
// the actual response, applies zero or more with-mutator tweaks, validates
// against the named fixture, and asserts success or the expected error texts.
// Subtests keep the legacy strings byte-identical for junit parity.
func TestValidateResponses_WithSampleFile(t *testing.T) {
	const samplePath = "test/data/http_response_files/sample1.http"

	t.Run("perfect match with sample1.http", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseFromRawHTTPFile(samplePath).and().
			anExpectedResponseFixture("sample1.http")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("status code mismatch", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseFromRawHTTPFile(samplePath).and().
			withStatusCode(500).and().
			anExpectedResponseFixture("sample1.http")

		when.
			validateResponses()

		then.
			validationFails(1, "status code mismatch: expected 200, got 500")
	})

	t.Run("status string mismatch", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseFromRawHTTPFile(samplePath).and().
			withStatusText("200 Everything is Fine").and().
			anExpectedResponseFixture("sample1.http")

		when.
			validateResponses()

		then.
			validationFails(1, "status string mismatch: expected '200 OK', got '200 Everything is Fine'")
	})

	t.Run("header value mismatch for Content-Type", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseFromRawHTTPFile(samplePath).and().
			withHeader("Content-Type", "text/plain").and().
			anExpectedResponseFixture("sample1.http")

		when.
			validateResponses()

		then.
			validationFails(1,
				"expected value 'application/json; charset=utf-8' for header 'Content-Type' not found")
	})

	t.Run("missing expected header Date", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseFromRawHTTPFile(samplePath).and().
			withoutHeader("Date").and().
			anExpectedResponseFixture("sample1.http")

		when.
			validateResponses()

		then.
			validationFails(1, "expected header 'Date' not found")
	})

	t.Run("body mismatch", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseFromRawHTTPFile(samplePath).and().
			withBody(`{"message": "this is not the sample body"}`).and().
			anExpectedResponseFixture("sample1.http")

		when.
			validateResponses()

		then.
			validationFails(1, "JSON content mismatch")
	})

	t.Run("BodyContains logic not triggered by exact file match (positive case)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseFromRawHTTPFile(samplePath).and().
			anExpectedResponseFixture("sample1.http")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("BodyContains logic not triggered, exact body mismatch from file", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseFromRawHTTPFile(samplePath).and().
			anExpectedResponseFixture("validator_withsample_bodycontains_exactmismatch.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "JSON content mismatch")
	})

	t.Run("BodyNotContains logic not triggered by exact file match (positive case)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseFromRawHTTPFile(samplePath).and().
			anExpectedResponseFixture("sample1.http")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("BodyNotContains logic not triggered, exact body mismatch from file "+
		"(actual contains something unwanted by this hypothetical check)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseFromRawHTTPFile(samplePath).and().
			withBody(`{"title": "delectus aut autem"}`).and().
			anExpectedResponseFixture("validator_withsample_bodynotcontains_exactmismatch.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "JSON content mismatch")
	})
}

// TestValidateResponses_PartialExpected covers .hresp files that contain only a
// subset of response fields (e.g. just status code, just headers). Each
// subtest hand-constructs the actual response and validates against the named
// partial-expected fixture. Subtest names are byte-identical with the legacy
// SCENARIO-LIB-009-* ids.
func TestValidateResponses_PartialExpected(t *testing.T) {
	t.Run("SCENARIO-LIB-009-005 Equiv: Expected file has only status code - match", func(t *testing.T) {
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
	})

	t.Run("SCENARIO-LIB-009-005 Corrected: File has status code and empty body - actual matches",
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

	t.Run("SCENARIO-LIB-009-005-003 Corrected: File has status code and "+
		"empty body - actual body mismatch", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200", "non-empty body", nil).and().
			anExpectedResponseFixture("validator_partial_status_code_mismatch.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "body mismatch")
	})

	t.Run("SCENARIO-LIB-009-005-004 Equiv: Expected file has only status code - status code mismatch",
		func(t *testing.T) {
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
		})

	t.Run("SCENARIO-LIB-009-006 Equiv: Expected file has only specific headers "+
		"(and status, empty body) - match", func(t *testing.T) {
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
	})

	t.Run("SCENARIO-LIB-009-006-002 Equiv: Expected file has only specific headers - header value mismatch",
		func(t *testing.T) {
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
		})

	t.Run("SCENARIO-LIB-009-006-003 Equiv: Expected file has only specific headers - header missing in actual",
		func(t *testing.T) {
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
		})
}

// TestValidateResponses_NilAndEmptyActuals covers the three ways the caller can
// pass "no responses" to ValidateResponses: a nil slice, an empty slice, and a
// slice containing a single nil. All three should be reported as a count
// mismatch against an expected file that defines one response.
func TestValidateResponses_NilAndEmptyActuals(t *testing.T) {
	t.Run("nil actual response slice", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			noActualResponses().and().
			anExpectedResponseFixture("validator_nil_empty_actuals_expected.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "mismatch in number of responses: got 0 actual, but expected 1")
	})

	t.Run("empty actual response slice", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			anEmptyResponseSlice().and().
			anExpectedResponseFixture("validator_nil_empty_actuals_expected.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "mismatch in number of responses: got 0 actual, but expected 1")
	})

	t.Run("slice with one nil actual response", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aNilResponse().and().
			anExpectedResponseFixture("validator_nil_empty_actuals_expected.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "mismatch in number of responses: got 0 actual, but expected 1")
	})
}

// TestValidateResponses_FileErrors covers the three expected-response-file error
// cases: missing file, empty file (parses to zero responses) and malformed
// status line (parses to zero responses after parse error).
func TestValidateResponses_FileErrors(t *testing.T) {
	t.Run("missing expected response file", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWithStatus(200, "").and().
			anExpectedResponseFileAt("nonexistent.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "failed to read expected response file", "nonexistent.hresp")
	})

	t.Run("empty expected response file", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWithStatus(200, "").and().
			anExpectedResponseFixture("validator_empty_expected.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "mismatch in number of responses: got 1 actual, but expected 0")
	})

	t.Run("malformed expected response file", func(t *testing.T) {
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
	})
}

// dslSymbolsValidator keeps every validator-DSL extension referenced so the
// `unused` linter stays quiet in this definitions-only file until tests adopt
// the DSL.
var _ = []any{
	(*parts).aResponseWith,
	(*parts).aResponseWithStatus,
	(*parts).aResponseFromRawHTTPFile,
	(*parts).withStatusCode,
	(*parts).withStatusText,
	(*parts).withHeader,
	(*parts).withoutHeader,
	(*parts).withBody,
	(*parts).noActualResponses,
	(*parts).anEmptyResponseSlice,
	(*parts).aNilResponse,
	(*parts).anExpectedResponseFileAt,
	(*parts).validateResponsesWithIndex,
}
