package restclient_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// expectedResponse describes the code and body a single executed response must carry.
type expectedResponse struct {
	code int
	body string
}

// ignoreEmptyBlocksCase is the table row for TestExecuteFile_IgnoreEmptyBlocks_Client.
type ignoreEmptyBlocksCase struct {
	name       string
	fixture    string
	serverRefs int // how many times the fixture needs the server URL substituted
	errTexts   []string
	responses  []expectedResponse
}

// PRD-COMMENT: FR11.1 - Client Execution: Invalid HTTP Method Handling
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

// ignoreEmptyBlocksHandler serves the shared mock behavior for every
// TestExecuteFile_IgnoreEmptyBlocks_Client scenario; it is plain test data. Method
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

// PRD-COMMENT: FR2.4 / FR11.2 - Handling of Non-Request Content
func TestExecuteFile_IgnoreEmptyBlocks_Client(t *testing.T) {
	tests := []ignoreEmptyBlocksCase{
		{
			name:       "SCENARIO-LIB-028-004: Valid request, then separator, then only comments",
			fixture:    "scenario_004_template.http",
			serverRefs: 1,
			responses:  []expectedResponse{{code: http.StatusOK, body: "response from /first"}},
		},
		{
			name:       "SCENARIO-LIB-028-005: Only comments, then separator, then valid request",
			fixture:    "scenario_005_template.http",
			serverRefs: 1,
			responses:  []expectedResponse{{code: http.StatusOK, body: "response from /second"}},
		},
		{
			name: "SCENARIO-LIB-028-006: Valid request, separator with comments, " +
				"then another valid request",
			fixture:    "scenario_006_template.http",
			serverRefs: 2,
			responses: []expectedResponse{
				{code: http.StatusAccepted, body: "response from /req1"},
				{code: http.StatusCreated, body: "response from /req2"},
			},
		},
		{
			name:     "File with only variable definitions - ExecuteFile",
			fixture:  "only_vars.http",
			errTexts: []string{"no requests found in file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runIgnoreEmptyBlocksScenario(t, tt)
		})
	}
}

// runIgnoreEmptyBlocksScenario executes one TestExecuteFile_IgnoreEmptyBlocks_Client
// scenario as a single fluent Given/When/Then chain. (Extracted from the table loop
// because golangci-lint's revive cognitive-complexity limit applies per function.)
func runIgnoreEmptyBlocksScenario(t *testing.T, tt ignoreEmptyBlocksCase) {
	t.Helper()

	given, when, then := newParts(t)

	// Constant placeholder tokens; aHttpFile resolves them to the server URL at chain
	// time, so no URL is captured before aHttpServer starts.
	subs := make([]string, tt.serverRefs)
	for i := range subs {
		subs[i] = "{{server}}"
	}

	given.
		aHttpServer(ignoreEmptyBlocksHandler).and().
		aFormattedRequestFixture(tt.fixture, subs...).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(len(tt.responses))

	if len(tt.errTexts) > 0 {
		then.
			errorContains(tt.errTexts...)

		return
	}

	then.
		noError()

	for i, expected := range tt.responses {
		then.
			responseAt(i).and().
			responseCode(expected.code).and().
			responseBodyIs(expected.body)
	}
}
