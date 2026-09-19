package restclient_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	rc "github.com/bmcszk/go-restclient"
)

// TestExecuteFile_WithCustomVariables: Custom Variables: Basic Definition and Substitution.

func TestExecuteFile_WithCustomVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aCannedServer(map[string]cannedRoute{
			"/users/testuser123": {method: http.MethodPost, code: http.StatusOK, body: "response for user testuser123"},
			"/products/testuser123": {
				method: http.MethodGet,
				code:   http.StatusOK,
				body:   "response from products/testuser123",
			},
			"/items/": {method: http.MethodGet, code: http.StatusOK, body: "response for items ()"},
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

// TestExecuteFile_WithDotEnvSystemVariable_Scenario1_EnvFileExists: System Variables {{$dotenv.VAR_NAME}}.
// .env file exists and both DOTENV_VAR1 and DOTENV_VAR2 are present.
func TestExecuteFile_WithDotEnvSystemVariable_Scenario1_EnvFileExists(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aDotEnvFileRemoved().and().
		aDotEnvFile("DOTENV_VAR1=dotenv_value_one\nDOTENV_VAR2=another val from dotenv").and().
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFileFromTemplate("system_var_dotenv_present.http").and().
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
}

// TestExecuteFile_WithDotEnvSystemVariable_Scenario2_EnvFileMissing: System Variables {{$dotenv.VAR_NAME}}.
// No .env file: all {{$dotenv.*}} variables resolve to empty strings.
func TestExecuteFile_WithDotEnvSystemVariable_Scenario2_EnvFileMissing(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aDotEnvFileRemoved().and().
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFileFromTemplate("system_var_dotenv_missing.http").and().
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

// TestExecuteFile_WithHttpClientEnvJson_NoEnvSelected: Environment Configuration Files (http-client.env.json).
// SCENARIO-LIB-018-004: env == "" means the http-client.env.json lookup is skipped, so
// {{host}} in the request body is left as a literal placeholder.
func TestExecuteFile_WithHttpClientEnvJson_NoEnvSelected(t *testing.T) {
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
}

// TestExecuteFile_WithHttpClientEnvJson_PrivateOverridesPublic: Environment Configuration Files.
// SCENARIO-LIB-018-005: when both http-client.env.json and http-client.private.env.json
// exist and an env is selected, the private file's values override the public ones.
func TestExecuteFile_WithHttpClientEnvJson_PrivateOverridesPublic(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		})

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
