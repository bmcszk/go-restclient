package restclient_test

import (
	"testing"
)

// A bare `{{ ... }}` httpyac script block between requests is skipped, not
// swallowed as body content (#46).
func TestParseFile_ScriptBlockBetweenRequestsIsIgnored(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### first
GET {{server}}/one

{{
  const { randomUUID } = require('crypto');
  exports.id = randomUUID();
}}

### second
GET {{server}}/two`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestCount(2).and().
		parsedRequestBodyIs(0, "")
}

// A script block inside a request section does not become body and variable
// substitution for the next request keeps working (#46).
func TestExecuteFile_ScriptBlockInRequestKeepsVarsWorking(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`# @name first
GET {{server}}/one

{{
  const { randomUUID } = require('crypto');
  exports.id = randomUUID();
}}

### second
GET {{server}}/two`).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2).and().
		serverReceivedBodyIs(0, "").and().
		serverReceivedMethodAndPath(1, "GET", "/two")
}
