package restclient_test

import (
	"net/http"
	"testing"
)

// TestExecuteFile_ScriptBlockBetweenRequestsIsIgnored verifies httpyac inline
// script blocks are skipped without breaking parsing or execution (#46).
func TestExecuteFile_ScriptBlockBetweenRequestsIsIgnored(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### first
GET {{server}}/one

{{
  const { randomUUID } = require('crypto');
  exports.id = randomUUID();
}}

### second
GET {{server}}/two`).
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodGet, "/one").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/two")
}

// TestExecuteFile_ScriptBlockInsideRequestIsIgnored verifies a script block in
// the request section does not become body or orphaned content (#46).
func TestExecuteFile_ScriptBlockInsideRequestIsIgnored(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### only
GET {{server}}/only

{{
  exports.id = 1;
}}

### after
GET {{server}}/after`).
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodGet, "/only").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/after")
}
