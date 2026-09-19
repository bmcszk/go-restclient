package restclient_test

import (
	"net/http"
	"testing"
)

func TestExecuteFile_InvalidMethodInFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aRequestFixture("invalid_method.http").and().
		aClient()

	when.
		executeFile()

	then.
		errorContains(
			"1 error occurred:",
			"unsupported protocol scheme",
			"request 1 (INVALIDMETHOD /test) processing resulted in error",
		).and().
		responseCount(1).and().
		responseAt(0).and().
		responseHasError("unsupported protocol scheme", "Invalidmethod")
}

func TestExecuteFile_IgnoreEmptyBlocks_ValidThenCommentOnly(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aCannedServer(map[string]cannedRoute{
			"/first": {method: http.MethodGet, code: http.StatusOK, body: "response from /first"},
		}).and().
		aFormattedRequestFixture("scenario_004_template.http", "{{server}}").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		responseBodyIs("response from /first")
}

func TestExecuteFile_IgnoreEmptyBlocks_CommentOnlyThenValid(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aCannedServer(map[string]cannedRoute{
			"/second": {method: http.MethodGet, code: http.StatusOK, body: "response from /second"},
		}).and().
		aFormattedRequestFixture("scenario_005_template.http", "{{server}}").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		responseBodyIs("response from /second")
}

func TestExecuteFile_IgnoreEmptyBlocks_TwoValidAcrossCommentBlock(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aCannedServer(map[string]cannedRoute{
			"/req1": {method: http.MethodGet, code: http.StatusAccepted, body: "response from /req1"},
			"/req2": {method: http.MethodPost, code: http.StatusCreated, body: "response from /req2", validJSON: true},
		}).and().
		aFormattedRequestFixture("scenario_006_template.http", "{{server}}", "{{server}}").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusAccepted).and().
		responseBodyIs("response from /req1").and().
		responseAt(1).and().
		responseCode(http.StatusCreated).and().
		responseBodyIs("response from /req2")
}
func TestExecuteFile_IgnoreEmptyBlocks_OnlyVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aRequestFixtureAbs("test/data/execute_file_ignore_empty_blocks/only_vars.http").and().
		aClient()

	when.
		executeFile()

	then.
		errorContains("no requests found in file").and().
		responseCount(0)
}
