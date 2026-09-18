package restclient_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// TestExecuteFile_UuidVariableConsistency tests that a variable defined as
// @scenarioId = {{$uuid}} maintains the same value throughout all uses in the file.
func TestExecuteFile_UuidVariableConsistency(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			body, _ := readAllBody(r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			switch r.Method {
			case http.MethodGet:
				fmt.Fprintf(w, `{"uuid": "%s"}`, strings.TrimPrefix(r.URL.Path, "/uuid/"))
			case http.MethodPost, http.MethodPut:
				fmt.Fprintf(w, `{"json": %s, "headers": {"X-Scenario-Id": "%s"}}`,
					string(body), r.Header.Get("X-Scenario-ID"))
			default:
				w.WriteHeader(http.StatusNotImplemented)
			}
		}).and().
		aTemplateFixture("system_variables", "uuid_consistency.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(3).and().
		noError().and().
		allResponseCodesAre(http.StatusOK).and().
		capturedRequestCount(3).and().
		tracking("uuid").and().
		capturedURLSegment(0).and().
		capturedJSONField(1, "scenario_id").and().
		capturedJSONField(1, "test_data", "uuid").and().
		capturedJSONField(1, "test_data", "metadata", "scenario").and().
		capturedHeaderField(2, "X-Scenario-ID").and().
		capturedJSONField(2, "update_scenario").and().
		allTrackedValuesAreValidUUIDs().and().
		allTrackedValuesEqual()
}
