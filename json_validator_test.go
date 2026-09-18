package restclient_test

import (
	"net/http"
	"testing"
)

// TestValidateResponses_JSON_WhitespaceComparison verifies that the validator
// treats JSON with different whitespace / indentation / key ordering as equal.
func TestValidateResponses_JSON_WhitespaceComparison(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", `{"key":"value"}`,
			http.Header{"Content-Type": {"application/json"}}).and().
		anExpectedResponseFixture("validator_json_indentation_different.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_JSON_WithPlaceholders verifies that JSON with
// placeholders in the expected body (e.g. {{$anyGuid}}) still validates
// against an actual JSON body that satisfies the placeholder.
func TestValidateResponses_JSON_WithPlaceholders(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK",
			`{"id": "550e8400-e29b-41d4-a716-446655440000"}`,
			http.Header{"Content-Type": {"application/json"}}).and().
		anExpectedResponseFixture("validator_json_placeholders_formatting.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestValidateResponses_JSON_WithPlaceholdersInBody verifies that JSON with
// placeholders inside expected bodies (e.g. timestamp placeholders) validates
// against an actual JSON body whose values match the placeholder shape.
func TestValidateResponses_JSON_WithPlaceholdersInBody(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		aResponseWith(200, "200 OK", `{"last_updated": 1634567890}`,
			http.Header{"Content-Type": {"application/json"}}).and().
		anExpectedResponseFixture("validator_json_placeholders_in_body.hresp")

	when.
		validateResponses()

	then.
		validationSucceeds()
}
