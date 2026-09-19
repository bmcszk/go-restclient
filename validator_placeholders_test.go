package restclient_test

import (
	"net/http"
	"testing"
)

// TestValidateResponses_BodyRegexpPlaceholder_SimpleMatch verifies the {{$regexp `pattern`}}
// placeholder succeeds when the actual JSON body matches the regex.

func TestValidateResponses_BodyRegexpPlaceholder_SimpleMatch(t *testing.T) {
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
}

// TestValidateResponses_BodyRegexpPlaceholder_SimpleNoMatch verifies a body mismatch is
// reported when the actual JSON body does not match the expected regex.
func TestValidateResponses_BodyRegexpPlaceholder_SimpleNoMatch(t *testing.T) {
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
}

// TestValidateResponses_BodyRegexpPlaceholder_MultipleMatch verifies the {{$regexp}}
// placeholder succeeds when multiple regex placeholders are used in the same expected body.
func TestValidateResponses_BodyRegexpPlaceholder_MultipleMatch(t *testing.T) {
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
}

// TestValidateResponses_BodyRegexpPlaceholder_InvalidPattern verifies the validator reports
// a regex compile error when the placeholder pattern is malformed.
func TestValidateResponses_BodyRegexpPlaceholder_InvalidPattern(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyGuidPlaceholder_ValidMatch verifies the {{$anyGuid}}
// placeholder succeeds with the standard 8-4-4-4-12 hex GUID shape.
func TestValidateResponses_BodyAnyGuidPlaceholder_ValidMatch(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyGuidPlaceholder_NotAGuid verifies a body mismatch when the
// field value is not a valid GUID.
func TestValidateResponses_BodyAnyGuidPlaceholder_NotAGuid(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyGuidPlaceholder_LargerText verifies the {{$anyGuid}}
// placeholder succeeds when the GUID appears inside a larger text body.
func TestValidateResponses_BodyAnyGuidPlaceholder_LargerText(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyTimestampPlaceholder_IntegerMatch verifies {{$anyTimestamp}}
// succeeds with a valid integer timestamp (Unix seconds as string).
func TestValidateResponses_BodyAnyTimestampPlaceholder_IntegerMatch(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyTimestampPlaceholder_StringValue verifies a body mismatch
// when the timestamp field is a non-numeric string.
func TestValidateResponses_BodyAnyTimestampPlaceholder_StringValue(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", `{"timestamp": "not-a-timestamp"}`,
			http.Header{"Content-Type": {"application/json"}}).and().
		anExpectedResponseFixture("validator_body_anytimestamp_no_match.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "body mismatch (regexp/placeholder evaluation failed)")
}

// TestValidateResponses_BodyAnyTimestampPlaceholder_FloatValue verifies a body mismatch
// when the timestamp field is a float rather than an integer.
func TestValidateResponses_BodyAnyTimestampPlaceholder_FloatValue(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", `{"eventTime": "1678886400.5"}`,
			http.Header{"Content-Type": {"application/json"}}).and().
		anExpectedResponseFixture("validator_body_anytimestamp_float.hresp")

	when.
		validateResponses()

	then.
		validationFails(1, "body mismatch (regexp/placeholder evaluation failed)")
}

// TestValidateResponses_BodyAnyDatetimePlaceholder_RFC1123Match verifies {{$anyDatetime rfc1123}}
// succeeds with a Tue, 15 Mar 2023 12:00:00 GMT body.
func TestValidateResponses_BodyAnyDatetimePlaceholder_RFC1123Match(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyDatetimePlaceholder_ISO8601Match verifies {{$anyDatetime iso8601}}
// succeeds with an RFC3339 body.
func TestValidateResponses_BodyAnyDatetimePlaceholder_ISO8601Match(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyDatetimePlaceholder_ISO8601OffsetMatch verifies
// {{$anyDatetime iso8601}} succeeds with an RFC3339 body carrying a numeric offset.
func TestValidateResponses_BodyAnyDatetimePlaceholder_ISO8601OffsetMatch(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyDatetimePlaceholder_ISO8601MillisecondsMatch verifies
// {{$anyDatetime iso8601}} succeeds with millisecond precision and a Z zone.
func TestValidateResponses_BodyAnyDatetimePlaceholder_ISO8601MillisecondsMatch(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyDatetimePlaceholder_CustomGoLayoutMatch verifies
// {{$anyDatetime 2006-01-02}} succeeds with a YYYY-MM-DD body.
func TestValidateResponses_BodyAnyDatetimePlaceholder_CustomGoLayoutMatch(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyDatetimePlaceholder_FormatMismatch verifies a body mismatch
// when the actual body uses YYYY-MM-DD but the .hresp expects RFC1123.
func TestValidateResponses_BodyAnyDatetimePlaceholder_FormatMismatch(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyDatetimePlaceholder_InvalidKeyword verifies a body mismatch
// when the {{$anyDatetime}} keyword is unrecognized (the failure regex (\z.\A) appears).
func TestValidateResponses_BodyAnyDatetimePlaceholder_InvalidKeyword(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyDatetimePlaceholder_MissingFormatArgument verifies a body
// mismatch when the {{$anyDatetime}} placeholder is used without its format argument.
func TestValidateResponses_BodyAnyDatetimePlaceholder_MissingFormatArgument(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyDatetimePlaceholder_CustomEmptyFormat verifies a body
// mismatch when the {{$anyDatetime}} placeholder is given an empty literal format.
func TestValidateResponses_BodyAnyDatetimePlaceholder_CustomEmptyFormat(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyPlaceholder_SimpleMatch verifies {{$any}} matches a simple
// string body.
func TestValidateResponses_BodyAnyPlaceholder_SimpleMatch(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyPlaceholder_SpecialChars verifies {{$any}} matches special
// characters and spaces in the body.
func TestValidateResponses_BodyAnyPlaceholder_SpecialChars(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyPlaceholder_EmptySegment verifies {{$any}} matches an empty
// segment in the body.
func TestValidateResponses_BodyAnyPlaceholder_EmptySegment(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyPlaceholder_MultiLine verifies {{$any}} matches a
// multi-line string body.
func TestValidateResponses_BodyAnyPlaceholder_MultiLine(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyPlaceholder_MultiplePlaceholders verifies multiple
// {{$any}} placeholders can be used in the same expected body.
func TestValidateResponses_BodyAnyPlaceholder_MultiplePlaceholders(t *testing.T) {
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
}

// TestValidateResponses_BodyAnyPlaceholder_PrecedingLiteralMismatch verifies {{$any}} fails
// when the literal preceding the placeholder does not match the actual body.
func TestValidateResponses_BodyAnyPlaceholder_PrecedingLiteralMismatch(t *testing.T) {
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
}
