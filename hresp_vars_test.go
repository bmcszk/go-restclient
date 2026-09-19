package restclient_test

import (
	"net/http"
	"testing"
)

// TestExtractHrespDefines_NoDefines verifies the .hresp @name=value extractor passes through
// a file with no defines unchanged. The actual body must match the inline body from the .hresp.

func TestExtractHrespDefines_NoDefines(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		expectedResponseFile(
			`HTTP/1.1 200 OK
Content-Type: application/json

{
  "message": "hello"
}`,
		).and().
		aResponseWith(200, "200 OK", "{\n  \"message\": \"hello\"\n}",
			http.Header{"Content-Type": {"application/json"}})

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestExtractHrespDefines_SimpleDefines verifies that @name=value pairs are extracted and
// substituted into the expected body via {{name}} placeholders.
func TestExtractHrespDefines_SimpleDefines(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		expectedResponseFile(
			`@name = John Doe
@age = 30
HTTP/1.1 200 OK
Content-Type: application/json

{
  "user": "{{name}}",
  "age": {{age}}
}`,
		).and().
		aResponseWith(200, "200 OK", "{\n  \"user\": \"John Doe\",\n  \"age\": 30\n}",
			http.Header{"Content-Type": {"application/json"}})

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestExtractHrespDefines_ExtraSpaces verifies the extractor tolerates leading/trailing
// whitespace around both name and value.
func TestExtractHrespDefines_ExtraSpaces(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		expectedResponseFile(
			`  @name  =   John Doe  
@age=30
HTTP/1.1 200 OK`,
		).and().
		aResponseWith(200, "200 OK", "", nil)

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestExtractHrespDefines_MixedWithComments verifies defines interleave with comments and
// blank lines before the status line.
func TestExtractHrespDefines_MixedWithComments(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		expectedResponseFile(
			`# This is a comment
@name = Value1

  @key2 = Another Value  

HTTP/1.1 200 OK

Body content here.`,
		).and().
		aResponseWith(200, "200 OK", "Body content here.", nil)

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestExtractHrespDefines_MalformedNoEquals verifies a malformed define (no `=` sign) is
// silently dropped and does not affect parsing of the status line.
func TestExtractHrespDefines_MalformedNoEquals(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		expectedResponseFile(
			`@name
HTTP/1.1 200 OK`,
		).and().
		aResponseWith(200, "200 OK", "", nil)

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestExtractHrespDefines_MalformedEmptyName verifies a define with an empty name (`@ = value`)
// is silently dropped.
func TestExtractHrespDefines_MalformedEmptyName(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		expectedResponseFile(
			`@ = value
HTTP/1.1 200 OK`,
		).and().
		aResponseWith(200, "200 OK", "", nil)

	when.
		validateResponses()

	then.
		validationSucceeds()
}

// TestExtractHrespDefines_EmptyValue verifies a define with an empty value substitutes to "".
func TestExtractHrespDefines_EmptyValue(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aClient().and().
		expectedResponseFile(
			`@name = 
HTTP/1.1 200 OK`,
		).and().
		aResponseWith(200, "200 OK", "", nil)

	when.
		validateResponses()

	then.
		validationSucceeds()
}
