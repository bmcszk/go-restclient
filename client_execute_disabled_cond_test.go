package restclient_test

import (
	"testing"

	rc "github.com/bmcszk/go-restclient"
)

// TestParseFile_ConditionalDisabledWithBang_TogglesPlainDisabled: parser stores the
// conditional expression text after `!` and does NOT set the plain Disabled flag.
func TestParseFile_ConditionalDisabledWithBang_TogglesPlainDisabled(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### literal-true
# @disabled !true
GET https://example.com/api`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestDisabled(0, false).and().
		parsedRequestDisabledExpr(0, "true")
}

// TestExecuteFile_ConditionalDisabled_TruthyExprSkips: `flag=true` -> request skipped,
// the test server receives zero hits, response entry is marked Skipped.
func TestExecuteFile_ConditionalDisabled_TruthyExprSkips(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("disabled_cond_flagvar.http").and().
		aClient(rc.WithVars(map[string]any{"flag": "true"}))

	when.
		executeFile()

	then.
		noError().and().
		responseCount(1).and().
		requestSkipped(0).and().
		capturedRequestCount(0)
}

// TestExecuteFile_ConditionalDisabled_FalsyExprRuns: `flag=false` -> request runs,
// the test server receives one hit, response entry is NOT marked Skipped.
func TestExecuteFile_ConditionalDisabled_FalsyExprRuns(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("disabled_cond_flagvar.http").and().
		aClient(rc.WithVars(map[string]any{"flag": "false"}))

	when.
		executeFile()

	then.
		noError().and().
		responseCount(1).and().
		requestExecuted(0).and().
		capturedRequestCount(1)
}

// TestExecuteFile_ConditionalDisabled_ZeroIsFalsy: numeric `flag=0` -> request runs.
func TestExecuteFile_ConditionalDisabled_ZeroIsFalsy(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("disabled_cond_flagvar.http").and().
		aClient(rc.WithVars(map[string]any{"flag": 0}))

	when.
		executeFile()

	then.
		noError().and().
		responseCount(1).and().
		requestExecuted(0).and().
		capturedRequestCount(1)
}

// TestExecuteFile_ConditionalDisabled_NumberOneIsTruthy: numeric `flag=1` -> request skipped.
func TestExecuteFile_ConditionalDisabled_NumberOneIsTruthy(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("disabled_cond_flagvar.http").and().
		aClient(rc.WithVars(map[string]any{"flag": 1}))

	when.
		executeFile()

	then.
		noError().and().
		responseCount(1).and().
		requestSkipped(0).and().
		capturedRequestCount(0)
}

// TestExecuteFile_ConditionalDisabled_EmptyStringRuns: substituted expr is empty -> request runs.
func TestExecuteFile_ConditionalDisabled_EmptyStringRuns(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("disabled_cond_flagvar.http").and().
		aClient(rc.WithVars(map[string]any{"flag": ""}))

	when.
		executeFile()

	then.
		noError().and().
		responseCount(1).and().
		requestExecuted(0).and().
		capturedRequestCount(1)
}

// TestExecuteFile_ConditionalDisabled_NonEmptySkips: substituted expr is any non-empty
// non-"true"/non-zero string -> request skipped.
func TestExecuteFile_ConditionalDisabled_NonEmptySkips(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("disabled_cond_flagvar.http").and().
		aClient(rc.WithVars(map[string]any{"flag": "anything"}))

	when.
		executeFile()

	then.
		noError().and().
		responseCount(1).and().
		requestSkipped(0).and().
		capturedRequestCount(0)
}

// TestExecuteFile_ConditionalDisabled_UnknownVariableErrors: `{{missing}}` references an
// undefined variable; execution must surface an error naming the variable, NOT skip silently.
func TestExecuteFile_ConditionalDisabled_UnknownVariableErrors(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("disabled_cond_missing.http").and().
		aClient()

	when.
		executeFile()

	then.
		errorContains("undefined variable", "missing")
}

// TestExecuteFile_ConditionalDisabled_CaseInsensitiveTrue: `flag="TRUE"` -> request skipped.
func TestExecuteFile_ConditionalDisabled_CaseInsensitiveTrue(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("disabled_cond_flagvar.http").and().
		aClient(rc.WithVars(map[string]any{"flag": "TRUE"}))

	when.
		executeFile()

	then.
		noError().and().
		responseCount(1).and().
		requestSkipped(0).and().
		capturedRequestCount(0)
}

// TestExecuteFile_ConditionalDisabled_CaseInsensitiveFalse: `flag="False"` -> request runs.
func TestExecuteFile_ConditionalDisabled_CaseInsensitiveFalse(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("disabled_cond_flagvar.http").and().
		aClient(rc.WithVars(map[string]any{"flag": "False"}))

	when.
		executeFile()

	then.
		noError().and().
		responseCount(1).and().
		requestExecuted(0).and().
		capturedRequestCount(1)
}
