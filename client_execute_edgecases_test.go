package restclient_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// ignoreEmptyBlocksHandler serves the shared mock behavior for every
// TestExecuteFile_IgnoreEmptyBlocks_* scenario; it is plain test data. Method
// checks use plain comparisons because a package-level handler has no *testing.T.
var ignoreEmptyBlocksHandler = func(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/first":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "response from /first")
	case "/second":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "response from /second")
	case "/req1":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprint(w, "response from /req1")
	case "/req2":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}
		if body, err := readAllBody(r); err != nil || !json.Valid(body) {
			w.WriteHeader(http.StatusBadRequest)

			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, "response from /req2")
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

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
		aHttpServer(ignoreEmptyBlocksHandler).and().
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
		aHttpServer(ignoreEmptyBlocksHandler).and().
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
		aHttpServer(ignoreEmptyBlocksHandler).and().
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