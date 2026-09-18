package restclient_test

import (
	"net/http"
	"testing"
)

// TestValidateResponses_BodyRegexpPlaceholder covers the {{$regexp `pattern`}}
// placeholder. The expected body is built with a JSON-content expected file so
// both bodies are valid JSON, but the placeholder logic still kicks in.
// SCENARIO-LIB-022-004 (regexp with special characters) is intentionally kept
// out to match the legacy comment.
func TestValidateResponses_BodyRegexpPlaceholder(t *testing.T) {
	t.Run("SCENARIO-LIB-022-001: simple regexp match", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"id": "123"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_regexp_simple_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-022-002: simple regexp no match", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"status": "FAILED"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_regexp_simple_no_match.hresp")

		when.
			validateResponses()

		then.
			validationFails(1,
				"body mismatch (regexp/placeholder evaluation failed)",
				`Compiled Regex: ^\{"status": "(SUCCESS)"\}$`)
	})

	t.Run("SCENARIO-LIB-022-003: multiple regexp match", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"userId": "U-abc", "transactionId": "T-12345"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_regexp_multiple_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-022-005: invalid regexp pattern", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"data": "value"}`, nil).and().
			anExpectedResponseFixture("validator_body_regexp_invalid_pattern.hresp")

		when.
			validateResponses()

		then.
			validationFails(1,
				"failed to compile master regex from expected body",
				"error parsing regexp")
	})
}

// TestValidateResponses_BodyAnyGuidPlaceholder covers the {{$anyGuid}}
// placeholder. Both the 8-4-4-4-12 hex shape and occurrences inside larger
// text are exercised.
func TestValidateResponses_BodyAnyGuidPlaceholder(t *testing.T) {
	t.Run("SCENARIO-LIB-023-001: valid GUID match", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK",
				`{"correlationId": "123e4567-e89b-12d3-a456-426614174000"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anyguid_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-023-002: not a GUID", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"id": "not-a-guid"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anyguid_no_match.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "body mismatch (regexp/placeholder evaluation failed)")
	})

	t.Run("SCENARIO-LIB-023-003: GUID in larger text", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK",
				`Session started with ID: 123e4567-e89b-12d3-a456-426614174000. `+
					`Please use this for subsequent requests.`,
				nil).and().
			anExpectedResponseFixture("validator_body_anyguid_larger_text.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})
}

// TestValidateResponses_BodyAnyTimestampPlaceholder covers the
// {{$anyTimestamp}} placeholder. The placeholder accepts a sequence of decimal
// digits and rejects non-integer timestamps (strings, floats).
func TestValidateResponses_BodyAnyTimestampPlaceholder(t *testing.T) {
	const anyTimestampMismatch = "body mismatch (regexp/placeholder evaluation failed)"

	t.Run("SCENARIO-LIB-024-001: valid integer timestamp match", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"createdAt": "1678886400"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anytimestamp_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-024-002: not an integer timestamp (string)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"timestamp": "not-a-timestamp"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anytimestamp_no_match.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, anyTimestampMismatch)
	})

	t.Run("SCENARIO-LIB-024-003: not an integer timestamp (float)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"eventTime": "1678886400.5"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anytimestamp_float.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, anyTimestampMismatch)
	})
}

// TestValidateResponses_BodyAnyDatetimePlaceholder covers the
// {{$anyDatetime format}} placeholder with each supported format keyword
// (rfc1123, iso8601, custom Go layout) plus the failure paths (invalid
// keyword, missing argument, empty format).
func TestValidateResponses_BodyAnyDatetimePlaceholder(t *testing.T) {
	t.Run("SCENARIO-LIB-025-001: rfc1123 match", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"lastModified": "Tue, 15 Mar 2023 12:00:00 GMT"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anydatetime_rfc1123_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-025-002: iso8601 match (RFC3339)", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"eventTime": "2023-03-15T12:00:00Z"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anydatetime_iso8601_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-025-002b: iso8601 match with offset", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"eventTime": "2023-03-15T12:00:00+01:00"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anydatetime_iso8601_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-025-002c: iso8601 match with milliseconds and Z", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"eventTime": "2023-03-15T12:00:00.123Z"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anydatetime_iso8601_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-025-003: custom Go layout match", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"date": "2023-03-15"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anydatetime_custom_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-025-004: format mismatch (rfc1123 expected, actual is YYYY-MM-DD)",
		func(t *testing.T) {
			given, when, then := newParts(t)

			given.
				aClient().and().
				aResponseWith(200, "200 OK", `{"timestamp": "2023-03-15"}`,
					http.Header{"Content-Type": {"application/json"}}).and().
				anExpectedResponseFixture("validator_body_anydatetime_format_mismatch.hresp")

			when.
				validateResponses()

			then.
				validationFails(1, "body mismatch (regexp/placeholder evaluation failed)")
		})

	t.Run("SCENARIO-LIB-025-005: invalid format keyword", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"time": "12:34:56"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anydatetime_invalid_keyword.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "body mismatch (regexp/placeholder evaluation failed)", `(\z.\A)`)
	})

	t.Run("SCENARIO-LIB-025-006: missing format argument", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"time": "12:34:56"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anydatetime_missing_format.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "body mismatch (regexp/placeholder evaluation failed)", `(\z.\A)`)
	})

	t.Run(`custom format empty string literal "" - should fail`, func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"date": ""}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_anydatetime_custom_empty_format.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "body mismatch (regexp/placeholder evaluation failed)", `(\z.\A)`)
	})
}

// TestValidateResponses_BodyAnyPlaceholder covers the generic {{$any}}
// placeholder. Special characters, multiline strings, multiple placeholders in
// the same line and the failure path where the surrounding literal does not
// match are all exercised.
func TestValidateResponses_BodyAnyPlaceholder(t *testing.T) {
	t.Run("SCENARIO-LIB-026-001: $any matching a simple string", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"key": "some value"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_any_simple_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-026-002: $any matching special characters and spaces", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK",
				`Value: !@#$%^&*()_+{}[];':",./<>?         end`,
				http.Header{"Content-Type": {"text/plain"}}).and().
			anExpectedResponseFixture("validator_body_any_special_chars_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-026-003: $any matching an empty string segment", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"prefix": "", "data": "", "suffix": ""}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_any_empty_segment_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-026-004: $any matching a multi-line string", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK",
				"Start:\nThis is line 1.\nThis is line 2.\nAnd line 3.\nEnd.",
				http.Header{"Content-Type": {"text/plain"}}).and().
			anExpectedResponseFixture("validator_body_any_multiline_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("SCENARIO-LIB-026-005: multiple $any placeholders", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK",
				`{"field1": "value1", "field2": "constant", "field3": "value3"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_any_multiple_placeholders_match.hresp")

		when.
			validateResponses()

		then.
			validationSucceeds()
	})

	t.Run("$any fails if preceding literal doesn't match", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aClient().and().
			aResponseWith(200, "200 OK", `{"wrong_key": "some value"}`,
				http.Header{"Content-Type": {"application/json"}}).and().
			anExpectedResponseFixture("validator_body_any_simple_match.hresp")

		when.
			validateResponses()

		then.
			validationFails(1, "body mismatch")
	})
}
