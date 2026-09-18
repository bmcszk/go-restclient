package restclient_test

import (
	"net/http"
	"testing"
)

// TestExtractHrespDefines verifies the production @name=value extractor inside
// .hresp content through the public ValidateResponses API. The legacy
// implementation called a copy of `extractHrespDefines` from the test package;
// this version exercises the unexported real function indirectly by feeding
// content with @defines through a freshly created expected-response file and
// asserting that:
//   - valid defines are extracted and substituted into the expected body via
//     {{name}} placeholders,
//   - malformed defines (no equals, empty name) are silently dropped,
//   - defines with empty value substitute to "".
//
// Subtest names are kept byte-identical with the legacy implementation.
func TestExtractHrespDefines(t *testing.T) {
	t.Run("no defines", func(t *testing.T) {
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
	})

	t.Run("simple defines", func(t *testing.T) {
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
	})

	t.Run("defines with extra spaces", func(t *testing.T) {
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
	})

	t.Run("defines mixed with comments and blank lines", func(t *testing.T) {
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
	})

	t.Run("malformed define - no equals", func(t *testing.T) {
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
	})

	t.Run("malformed define - empty name", func(t *testing.T) {
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
	})

	t.Run("define with empty value", func(t *testing.T) {
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
	})
}
