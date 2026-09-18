// Code in this file is migrated from test/client_execute_vars.go (RunExecuteFile_WithCustomVariables,
// RunExecuteFile_WithProcessEnvSystemVariable, RunExecuteFile_WithDotEnvSystemVariable,
// RunExecuteFile_WithProgrammaticVariables, RunExecuteFile_WithLocalDatetimeSystemVariable,
// RunExecuteFile_VariableFunctionConsistency, RunExecuteFile_WithHttpClientEnvJson,
// RunExecuteFile_WithExtendedRandomSystemVariables, RunExecuteFile_WithIndirectEnvironmentVariables).
package restclient_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	rc "github.com/bmcszk/go-restclient"
)

// TestExecuteFile_WithCustomVariables: Custom Variables: Basic Definition and Substitution.
func TestExecuteFile_WithCustomVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			respondToCustomVariablesRequest(w, r)
		}).and().
		aHttpFileFromTemplate("custom_variables.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(3).and().
		requestCount(3).and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responseBodyIs("response for user testuser123").and().
		responseAt(1).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responseBodyIs("response from products/testuser123").and().
		responseAt(2).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responseBodyIs("response for items ()")
}

// respondToCustomVariablesRequest dispatches the per-path response logic for the custom-
// variables mock server. Extracted from the test func to keep cognitive complexity below
// the revive threshold; it is only called from inside TestExecuteFile_WithCustomVariables.
func respondToCustomVariablesRequest(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/users/testuser123":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "response for user testuser123")
	case "/products/testuser123":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "response from products/testuser123")
	case "/items/":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "response for items ()")
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// TestExecuteFile_WithProcessEnvSystemVariable: System Variables {{$processEnv.VAR_NAME}}.
func TestExecuteFile_WithProcessEnvSystemVariable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("GO_RESTCLIENT_TEST_VAR", "test_env_value_123").and().
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFileFromTemplateWithData("system_var_process_env.http",
			struct {
				ServerURL           string
				TestEnvVarName      string
				UndefinedEnvVarName string
			}{given.serverURL, "GO_RESTCLIENT_TEST_VAR", "GO_RESTCLIENT_UNDEFINED_VAR"}).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		capturedRequestURLIs(0, "/path-test_env_value_123/data").and().
		serverReceivedHeaderValue(0, "X-Env-Value", "test_env_value_123").and().
		capturedJSONStringMapIs(0, map[string]string{
			"env_payload":       "test_env_value_123",
			"undefined_payload": "{{$processEnv GO_RESTCLIENT_UNDEFINED_VAR}}",
		}).and().
		serverReceivedHeaderValue(0, "Cache-Control", "{{$processEnv UNDEFINED_CACHE_VAR_SHOULD_BE_EMPTY}}")
}

// TestExecuteFile_WithDotEnvSystemVariable: System Variables {{$dotenv.VAR_NAME}}.
//
// Two subtests are preserved with their original names. Each subtest creates fresh
// parts so the .env file state from a previous scenario cannot leak.
func TestExecuteFile_WithDotEnvSystemVariable(t *testing.T) {
	t.Run("Scenario 1: .env file exists and variable is present", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aDotEnvFileRemoved().and().
			aDotEnvFile("DOTENV_VAR1=dotenv_value_one\nDOTENV_VAR2=another val from dotenv").and().
			aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "ok")
			}).and().
			aHttpFile(fmt.Sprintf(`
GET %s/path-{{$dotenv DOTENV_VAR1}}/data
Content-Type: application/json
X-Dotenv-Value: {{$dotenv DOTENV_VAR2}}

{
  "payload": "{{$dotenv DOTENV_VAR1}}",
  "missing_payload": "{{$dotenv MISSING_DOTENV_VAR}}"
}`, given.serverURL)).and().
			aClient()

		when.
			executeFile()

		then.
			responseCount(1).and().
			noError().and().
			responseAt(0).and().
			responseHasNoError().and().
			responseCode(http.StatusOK).and().
			capturedRequestURLIs(0, "/path-dotenv_value_one/data").and().
			serverReceivedHeaderValue(0, "X-Dotenv-Value", "another val from dotenv").and().
			capturedJSONStringMapIs(0, map[string]string{
				"payload":         "dotenv_value_one",
				"missing_payload": "",
			})
	})

	t.Run("Scenario 2: .env file does not exist", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aDotEnvFileRemoved().and().
			aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "ok")
			}).and().
			aHttpFile(fmt.Sprintf(`
GET %s/path-{{$dotenv DOTENV_VAR_SHOULD_BE_EMPTY}}/data
User-Agent: test-client

{
  "payload": "{{$dotenv DOTENV_VAR_ALSO_EMPTY}}"
}`, given.serverURL)).and().
			aClient()

		when.
			executeFile()

		then.
			responseCount(1).and().
			noError().and().
			responseAt(0).and().
			responseHasNoError().and().
			responseCode(http.StatusOK).and().
			capturedRequestURLIs(0, "/path-/data").and().
			capturedJSONStringMapIs(0, map[string]string{
				"payload": "",
			})
	})
}

// TestExecuteFile_WithProgrammaticVariables: Programmatic Variable Injection.
func TestExecuteFile_WithProgrammaticVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("PROG_ENV_VAR", "env_value_should_be_overridden").and().
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixture("programmatic_variables.http").and().
		aClient(rc.WithVars(map[string]any{
			"prog_baseUrl":         given.serverURL,
			"prog_path":            "items",
			"prog_id":              "prog123",
			"prog_headerVal":       "ProgrammaticHeaderValue",
			"prog_bodyField":       "dataFromProgrammatic",
			"file_var_to_override": "overridden_by_programmatic",
			"PROG_ENV_VAR":         "programmatic_wins_over_env",
		}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		capturedRequestPathIs(0, "/items/prog123").and().
		serverReceivedHeaderValue(0, "X-Test-Header", "ProgrammaticHeaderValue").and().
		capturedJSONStringMapIs(0, map[string]string{
			"field":               "dataFromProgrammatic",
			"overridden_file_var": "overridden_by_programmatic",
			"env_var_check":       "programmatic_wins_over_env",
			"file_only_check":     "file_only",
		}).and().
		requestHeaderIs("X-File-Var", "overridden_by_programmatic").and().
		requestHeaderIs("X-Env-Var", "programmatic_wins_over_env").and().
		requestHeaderIs("X-Unused-File-Var", "file_only")
}

// TestExecuteFile_WithLocalDatetimeSystemVariable: despite the name uses the
// $timestamp fixture. The time window before/after ExecuteFile is captured via
// local vars (parity-preserving; batch-2a precedent).
func TestExecuteFile_WithLocalDatetimeSystemVariable(t *testing.T) {
	runTimestampConsistencyAssertions(t)
}

// runTimestampConsistencyAssertions is the shared body of
// TestExecuteFile_WithLocalDatetimeSystemVariable and
// TestExecuteFile_WithTimestampSystemVariable. Both legacy tests use the same
// system_var_timestamp.http fixture with identical consistency checks.
func runTimestampConsistencyAssertions(t *testing.T) {
	t.Helper()
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFileFromTemplate("system_var_timestamp.http").and().
		aClient()

	beforeSec := time.Now().UTC().Unix()
	when.
		executeFile()
	afterSec := time.Now().UTC().Unix()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		tracking("timestamp").and().
		capturedURLSegment(0).and().
		capturedHeaderField(0, "X-Request-Time").and().
		capturedJSONField(0, "event_time").and().
		capturedJSONField(0, "processed_at").and().
		allTrackedValuesEqual().and().
		firstTrackedValueIsIntegerInRange(beforeSec, afterSec).and().
		allTrackedValuesArePositiveIntegers()
}

// TestExecuteFile_VariableFunctionConsistency: Variable Function Consistency (Internal).
//
// Server-side tracking already proves all values are equal pairwise. The
// request-object equality is verified by requestPathMatchesCapturedPath +
// requestHeadersMatchCaptured + requestRawBodyMatchesCapturedBody (parity 1:1).
func TestExecuteFile_VariableFunctionConsistency(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aRequestFixture("variable_function_consistency.rest").and().
		aClient(rc.WithBaseURL(given.serverURL))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		tracking("uuid").and().
		capturedURLSegment(0).and().
		capturedHeaderField(0, "X-Request-UUID").and().
		capturedJSONField(0, "id").and().
		capturedJSONField(0, "another_id").and().
		allTrackedValuesAreValidUUIDs().and().
		allTrackedValuesEqual().and().
		tracking("timestamp").and().
		capturedHeaderField(0, "X-Request-Timestamp").and().
		capturedJSONField(0, "timestamp").and().
		allTrackedValuesEqual().and().
		allTrackedValuesArePositiveIntegers().and().
		tracking("randomInt").and().
		capturedHeaderField(0, "X-Request-RandomInt").and().
		capturedJSONField(0, "randomInt").and().
		allTrackedValuesEqual().and().
		allTrackedValuesArePositiveIntegers().and().
		requestPathMatchesCapturedPath().and().
		requestHeadersMatchCaptured("X-Request-UUID", "X-Request-Timestamp", "X-Request-RandomInt").and().
		requestRawBodyMatchesCapturedBody()
}

// TestExecuteFile_WithHttpClientEnvJson: Environment Configuration Files (http-client.env.json).
//
// Two subtests preserved with their exact original names. Each subtest builds its
// own server + client (batch-2b subtest pattern) so the http-client.env.json file
// state is fully isolated.
func TestExecuteFile_WithHttpClientEnvJson(t *testing.T) {
	t.Run("SCENARIO-LIB-018-004: no env selected, file exists", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "ok")
			}).and().
			anEnvJsonFile("http-client.env.json",
				"test/data/execute_file_httpclientenv/no_env_selected_env_template.json", given.serverURL).and().
			aFixtureCopy("test/data/execute_file_httpclientenv/no_env_selected_request.http",
				"request.http", nil).and().
			aClientWithEnvironment("")

		when.
			executeFile()

		then.
			errorContains("unsupported protocol scheme \"\"").and().
			responseCount(1).and().
			responseAt(0).and().
			responseHasError("unsupported protocol scheme \"\"").and().
			requestRawURLContains("{{host}}")
	})

	t.Run("SCENARIO-LIB-018-005: private env overrides public env", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "ok")
			})

		// serverURL is now populated; compute the expected host for assertions.
		expectedHost := urlHost(given.serverURL)

		given.
			anEnvJsonFile("http-client.env.json",
				"test/data/execute_file_httpclientenv/private_overrides_public_env_template.json",
				given.serverURL).and().
			anEnvJsonFile("http-client.private.env.json",
				"test/data/execute_file_httpclientenv/private_overrides_private_env.json",
				given.serverURL).and().
			aFixtureCopy("test/data/execute_file_httpclientenv/private_overrides_request.http",
				"request.http", nil).and().
			aClientWithEnvironment("dev")

		when.
			executeFile()

		then.
			responseCount(1).and().
			noError().and().
			responseAt(0).and().
			responseHasNoError().and().
			responseCode(http.StatusOK).and().
			serverReceivedHostIs(0, expectedHost).and().
			capturedRequestPathIs(0, "/test").and().
			serverReceivedHeaderValue(0, "X-Custom-Header", "private_override_value").and().
			serverReceivedJSONBody(0, `{"public":"public_value","private_only":"private_specific_value"}`)
	})
}

// TestExecuteFile_WithIndirectEnvironmentVariables: Indirect Environment Variable Lookup
// {{$processEnv %VAR}}.
func TestExecuteFile_WithIndirectEnvironmentVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("TEST_SECRET_KEY", "secret123").and().
		withEnv("TEST_DATABASE_URL", "postgres://localhost:5432/test").and().
		withEnv("PROD_ENV", "production").and().
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aTemplateFixture("system_variables", "indirect_env_lookup.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient(rc.WithVars(map[string]any{
			"secretKeyVar": "TEST_SECRET_KEY",
			"dbUrlVar":     "TEST_DATABASE_URL",
			"envVar":       "PROD_ENV",
			"missingVar":   "NONEXISTENT_ENV_VAR",
		}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		serverReceivedHeaderValue(0, "X-Secret-Key", "secret123").and().
		serverReceivedHeaderValue(0, "X-Database-URL", "postgres://localhost:5432/test").and().
		serverReceivedHeaderValue(0, "X-Missing-Var", "").and().
		capturedJSONStringMapIs(0, map[string]string{
			"environment": "production",
			"missing":     "{{$processEnv %undefinedVar}}",
		})
}

// TestExecuteFile_WithExtendedRandomSystemVariables: System Variables {{$random.*}}.
func TestExecuteFile_WithExtendedRandomSystemVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aHttpFileFromTemplate("system_var_extended_random.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		tracking("randInt").and().
		capturedJSONField(0, "randInt").and().
		firstTrackedValueIsIntegerInRange(10, 20).and().
		tracking("randIntNegative").and().
		capturedJSONField(0, "randIntNegative").and().
		firstTrackedValueIsIntegerInRange(-5, 5).and().
		tracking("randFloat").and().
		capturedJSONField(0, "randFloat").and().
		firstTrackedValueIsFloatInRange(1.0, 2.5).and().
		tracking("randFloatNegative").and().
		capturedJSONField(0, "randFloatNegative").and().
		firstTrackedValueIsFloatInRange(-1.5, 0.5).and().
		tracking("randAlphabetic").and().
		capturedJSONField(0, "randAlphabetic").and().
		allTrackedValuesMatchRegexp(`^[a-zA-Z]{10}$`).and().
		capturedJSONFieldIs(0, "", "randAlphabeticZero").and().
		capturedJSONFieldIs(0, "{{$random.alphabetic abc}}", "randAlphabeticInvalid").and().
		tracking("randAlphanumeric").and().
		capturedJSONField(0, "randAlphanumeric").and().
		allTrackedValuesMatchRegexp(`^[a-zA-Z0-9_]{15}$`).and().
		tracking("randHex").and().
		capturedJSONField(0, "randHex").and().
		allTrackedValuesMatchRegexp(`^[0-9a-f]{8}$`).and().
		tracking("randEmail").and().
		capturedJSONField(0, "randEmail").and().
		allTrackedValuesMatchRegexp(`^[a-zA-Z0-9_]+@[a-zA-Z]+\.[a-zA-Z]{2,3}$`)
}

// Silence unused imports.
var _ = time.Time{}
