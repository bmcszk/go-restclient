package restclient_test

import (
	"fmt"
	rc "github.com/bmcszk/go-restclient"
	"net/http"
	"testing"
)

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

// %7B%7B$dotenv VAR%7D%7D in a URL query resolves from .env (URL-safe embedding).
func TestExecuteFile_WithDotEnvSystemVariable_URLEncodedPlaceholder(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aDotEnvFile("SECRET_VAR=s3cr3t-value").and().
		aHttpFile(`### enc
GET {{server}}/q?token=%7B%7B$dotenv SECRET_VAR%7D%7D`).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		capturedRequestURLIs(0, "/q?token=s3cr3t-value")
}

// URL-encoded placeholder for an undefined dotenv variable resolves to empty.
func TestExecuteFile_WithDotEnvSystemVariable_URLEncodedUndefinedIsEmpty(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aDotEnvFile("OTHER_VAR=x").and().
		aHttpFile(`### enc
GET {{server}}/q?token=%7B%7B$dotenv MISSING_VAR%7D%7D`).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		capturedRequestURLIs(0, "/q?token=")
}

// %7B%7B$processEnv VAR%7D%7D in a URL query resolves from the process env.
func TestExecuteFile_WithProcessEnvSystemVariable_URLEncodedPlaceholder(t *testing.T) {
	t.Setenv("GO_RESTCLIENT_ENCODED_VAR", "enc-val")
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### enc
GET {{server}}/q?v=%7B%7B$processEnv GO_RESTCLIENT_ENCODED_VAR%7D%7D`).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		capturedRequestURLIs(0, "/q?v=enc-val")
}

// TestExecuteFile_DotEnvProcessEnvPlaceholderExpands: .env values support {{$processEnv}}
// so secrets can come from the real environment (#49).
func TestExecuteFile_DotEnvProcessEnvPlaceholderExpands(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("ISS49_CID", "env-value").and().
		aDotEnvFile("CID={{$processEnv ISS49_CID}}").and().
		anEchoServer().and().
		aHttpFile("@client = {{CID}}\n\nPOST {{server}}/echo\nX-Client: {{client}}\n\nok").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-Client", "env-value")
}

// TestExecuteFile_DotEnvUnresolvablePlaceholderStaysRaw: missing process env leaves the
// {{$processEnv}} placeholder literal in the .env value (#49 pinned semantics).
func TestExecuteFile_DotEnvUnresolvablePlaceholderStaysRaw(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aDotEnvFile("CID={{$processEnv ISS49_MISSING}}").and().
		anEchoServer().and().
		aHttpFile("@client = {{CID}}\n\nPOST {{server}}/echo\nX-Client: {{client}}\n\nok").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-Client", "{{$processEnv ISS49_MISSING}}")
}

// TestExecuteFile_EnvFlagLoadsNamedEnvFile: --env staging loads .env first, then .env.staging
// overrides only the keys it defines (#48).
func TestExecuteFile_EnvFlagLoadsNamedEnvFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aDotEnvFile("A=base").and().
		aDotEnvNamedFile("staging", "A=staging\nB=two").and().
		anEchoServer().and().
		aHttpFile("@client = {{A}}\n\nPOST {{server}}/echo\nX-A: {{A}}\nX-B: {{B}}\n\nok").and().
		aClientWithEnvName("staging")

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-A", "staging").and().
		serverReceivedHeaderValue(0, "X-B", "two")
}

// TestExecuteFile_NamedEnvFilePlaceholderExpands: .env.staging values support
// {{$processEnv}} expansion like .env (#49 + #48 combined).
func TestExecuteFile_NamedEnvFilePlaceholderExpands(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("ISS49_S", "expanded-staging").and().
		aDotEnvFileRemoved().and().
		aDotEnvNamedFile("staging", "CID={{$processEnv ISS49_S}}").and().
		anEchoServer().and().
		aHttpFile("@client = {{CID}}\n\nPOST {{server}}/echo\nX-Client: {{client}}\n\nok").and().
		aClientWithEnvName("staging")

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-Client", "expanded-staging")
}
