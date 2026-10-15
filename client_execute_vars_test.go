package restclient_test

import (
	"fmt"
	"net/http"
	"testing"

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

// TestExecuteFile_InPlace_SimpleVariableInURL: In-Place Variables - Simple Definition and URL Substitution.

func TestExecuteFile_InPlace_SimpleVariableInURL(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"status":"ok"}`)
		}).and().
		aRequestFixtureAbs("test/data/execute_inplace_vars/simple_variable_in_url/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responsesValidateAgainst("test/data/execute_inplace_vars/simple_variable_in_url/expected.hresp").and().
		capturedRequestPathIs(0, "/api/v1/items/123")
}

// TestExecuteFile_InPlace_VariableInHeader: In-Place Variables - Header Substitution.
func TestExecuteFile_InPlace_VariableInHeader(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"status":"ok"}`)
		}).and().
		aRequestFixtureAbs("test/data/execute_inplace_vars/variable_in_header/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responsesValidateAgainst("test/data/execute_inplace_vars/variable_in_header/expected.hresp").and().
		serverReceivedHeaderValue(0, "Authorization", "Bearer_secret_token_123").and().
		serverReceivedHeaderValue(0, "User-Agent", "test-client")
}

// TestExecuteFile_InPlace_VariableInBody: In-Place Variables - Body Substitution.
func TestExecuteFile_InPlace_VariableInBody(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"status":"created"}`)
		}).and().
		aRequestFixtureAbs("test/data/execute_inplace_vars/variable_in_body/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responsesValidateAgainst("test/data/execute_inplace_vars/variable_in_body/expected.hresp").and().
		serverReceivedJSONBody(0, `{"id":"SW1000","name":"SuperWidget","price":49.99}`)
}

// TestExecuteFile_InPlace_VariableDefinedByAnotherVariable: In-Place - Referencing Other In-Place.
func TestExecuteFile_InPlace_VariableDefinedByAnotherVariable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"id":"123", "status":"ok"}`)
		}).and().
		aRequestFixtureAbs("test/data/execute_inplace_vars/variable_defined_by_another_variable/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responsesValidateAgainst(
			"test/data/execute_inplace_vars/variable_defined_by_another_variable/expected.hresp").and().
		capturedRequestURLIs(0, "/users/123")
}

// TestExecuteFile_InPlace_VariablePrecedenceOverEnvironment: Variable Precedence - In-Place over Environment.
func TestExecuteFile_InPlace_VariablePrecedenceOverEnvironment(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs("test/data/execute_inplace_vars/variable_precedence_over_environment/request.http").and().
		aClient(
			rc.WithEnvironment("testPrecedenceEnv"),
			rc.WithVars(map[string]any{"test_server_url": given.serverURL}),
		)

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		capturedRequestPathIs(0, "/expected_path")
}

// TestExecuteFile_InPlace_VariableInCustomHeader: In-Place - Custom Header Substitution.
func TestExecuteFile_InPlace_VariableInCustomHeader(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs("test/data/execute_inplace_vars/variable_in_custom_header/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responsesValidateAgainst("test/data/execute_inplace_vars/variable_in_custom_header/expected.hresp").and().
		serverReceivedHeaderValue(0, "X-Custom-Header", "secret-token")
}

// TestExecuteFile_InPlace_VariableSubstitutionInBody: In-Place - Complex Body Substitution.
func TestExecuteFile_InPlace_VariableSubstitutionInBody(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs("test/data/execute_inplace_vars/variable_substitution_in_body/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responsesValidateAgainst(
			"test/data/execute_inplace_vars/variable_substitution_in_body/expected.hresp").and().
		serverReceivedJSONBody(0, `{"id":"user123","status":"active"}`)
}

// TestExecuteFile_InPlace_VariableDefinedBySystemVariable covers pathSegments[0] being a 36-char UUID plus "resource".
func TestExecuteFile_InPlace_VariableDefinedBySystemVariable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs(
			"test/data/execute_inplace_vars/inplace_variable_defined_by_system_variable/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responsesValidateAgainst(
			"test/data/execute_inplace_vars/inplace_variable_defined_by_system_variable/expected.hresp").and().
		capturedRequestPathMatchesRegexp(0, `^/[^/]{36}/resource$`)
}

// TestExecuteFile_InPlace_VariableDefinedByOsEnvVariable: In-Place - Referencing OS Environment.
func TestExecuteFile_InPlace_VariableDefinedByOsEnvVariable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("TEST_USER_HOME_INPLACE", "/testhome/userdir").and().
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs(
			"test/data/execute_inplace_vars/inplace_variable_defined_by_os_env_variable/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responsesValidateAgainst(
			"test/data/execute_inplace_vars/inplace_variable_defined_by_os_env_variable/expected.hresp").and().
		capturedRequestPathIs(0, "/testhome/userdir/files")
}

// TestExecuteFile_InPlace_VariableInAuthHeader: In-Place - Authentication Header Substitution.
func TestExecuteFile_InPlace_VariableInAuthHeader(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs("test/data/execute_inplace_vars/inplace_variable_in_auth_header/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		serverReceivedHeaderValue(0, "X-Auth-Token", "secret-token-12345")
}

// TestExecuteFile_InPlace_VariableInJsonRequestBody: In-Place - JSON Request Body.
func TestExecuteFile_InPlace_VariableInJsonRequestBody(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs(
			"test/data/execute_inplace_vars/inplace_variable_in_json_request_body/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		serverReceivedJSONBody(0, `{"id":"user-from-var-456","status":"pending"}`)
}

// TestExecuteFile_InPlace_VariableDefinedByAnotherInPlaceVariable: In-Place - Chained In-Place.
func TestExecuteFile_InPlace_VariableDefinedByAnotherInPlaceVariable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs(
			"test/data/execute_inplace_vars/inplace_variable_defined_by_another_inplace_variable/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		capturedRequestPathIs(0, "/api/v1/items/123")
}

// TestExecuteFile_InPlace_VariableDefinedByDotEnvOsVariable: In-Place - {{$env.VAR}}.
func TestExecuteFile_InPlace_VariableDefinedByDotEnvOsVariable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("MY_CONFIG_PATH_TEST_DOT_ENV", "/usr/local/appconfig_dotenv").and().
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs(
			"test/data/execute_inplace_vars/inplace_variable_defined_by_dot_env_os_variable/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		capturedRequestPathIs(0, "/usr/local/appconfig_dotenv/data")
}

// TestExecuteFile_InPlace_Malformed_NameOnlyNoEqualsNoValue: Malformed Definition (Name Only).
func TestExecuteFile_InPlace_Malformed_NameOnlyNoEqualsNoValue(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aRequestFixtureAbs(
			"test/data/execute_inplace_vars/malformed_name_only_no_equals_no_value/request.http").and().
		aClient()

	when.
		executeFile()

	then.
		errorContains(
			"failed to parse request file",
			"malformed in-place variable definition, missing '=' or name part invalid",
		).and().
		responseCount(0)
}

// TestExecuteFile_InPlace_Malformed_NoNameEqualsValue: Malformed Definition (No Name).
func TestExecuteFile_InPlace_Malformed_NoNameEqualsValue(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aRequestFixtureAbs(
			"test/data/execute_inplace_vars/malformed_no_name_equals_value/request.http").and().
		aClient()

	when.
		executeFile()

	then.
		errorContains(
			"failed to parse request file",
			"malformed in-place variable definition, variable name cannot be empty",
		).and().
		responseCount(0)
}

// TestExecuteFile_InPlace_VariableDefinedByDotEnvSystemVariable: In-Place - {{$dotenv VAR}}.
//
// The .http file references it via {{$dotenv}}. capturedRequestPathIs asserts the substitution.
func TestExecuteFile_InPlace_VariableDefinedByDotEnvSystemVariable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs(
			"test/data/execute_inplace_vars/inplace_variable_defined_by_dotenv_system_variable/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		capturedRequestPathIs(0, "/actual_dotenv_value")
}

// TestExecuteFile_InPlace_VariableDefinedByRandomInt: In-Place - {{$randomInt MIN MAX}}.
func TestExecuteFile_InPlace_VariableDefinedByRandomInt(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aRequestFixtureAbs(
			"test/data/execute_inplace_vars/inplace_variable_defined_by_random_int/request.http").and().
		aClient(rc.WithVars(map[string]any{"test_server_url": given.serverURL}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		tracking("port").and().
		capturedURLSegmentAt(0, 2).and().
		capturedHeaderField(0, "X-Random-Port").and().
		allTrackedValuesEqual().and().
		firstTrackedValueIsIntegerInRange(8000, 8080)
}
