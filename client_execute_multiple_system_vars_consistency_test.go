package restclient_test

import (
	"net/http"
	"testing"
)

// TestExecuteFile_MultipleSystemVarsConsistency tests that multiple system variables
// defined in file-scoped variables maintain consistency across all requests in the file.
// Capture paths verified against test/data/system_variables/multiple_system_vars_consistency.http.
func TestExecuteFile_MultipleSystemVarsConsistency(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "ok"}`))
		}).and().
		aTemplateFixture("system_variables", "multiple_system_vars_consistency.http",
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
		capturedJSONField(1, "transaction_id").and().
		capturedJSONField(1, "session", "id").and().
		capturedHeaderField(2, "X-Transaction-ID").and().
		capturedJSONField(2, "transaction_ref").and().
		allTrackedValuesAreValidUUIDs().and().
		allTrackedValuesEqual().and().
		tracking("timestamp").and().
		capturedHeaderField(0, "X-Request-Time").and().
		capturedJSONNumberField(1, "timestamp").and().
		capturedHeaderField(1, "X-Request-Time").and().
		capturedJSONNumberField(2, "last_activity").and().
		allTrackedValuesArePositiveIntegers().and().
		allTrackedValuesEqual().and().
		tracking("session token").and().
		capturedHeaderField(0, "X-Session-Token").and().
		firstTrackedValueIsIntegerInRange(1000000, 9999999).and().
		capturedJSONField(1, "session", "token").and().
		capturedURLSegment(2).and().
		capturedJSONField(2, "session_token").and().
		allTrackedValuesEqual()
}
