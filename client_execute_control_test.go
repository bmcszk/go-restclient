package restclient_test

import (
	"net/http"
	"testing"
	"time"

	rc "github.com/bmcszk/go-restclient"
)

// `# @loop for 3` parses and stores the literal count.
func TestParseFile_LoopForLiteralStoresCount(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### loop
# @loop for 3
GET https://example.com/loop`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestLoopCount(0, 3)
}

// `# @loop for {{n}}` parses and stores the raw expression.
func TestParseFile_LoopForVariableStoresExpression(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### loop
# @loop for {{n}}
GET https://example.com/loop`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestLoopExpr(0, "{{n}}")
}

// `# @loop for item of items` parses and stores the collection variable name.
func TestParseFile_LoopOfCollectionStoresCollectionRef(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFileFromTemplate("loop_collection.http").and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestLoopCollection(0, "items")
}

// `# @loop for 3` sends the request exactly 3 times.
func TestExecuteFile_LoopForLiteralRunsNTimes(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### loop
# @loop for 3
GET {{server}}/loop`).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(3)
}

// `# @loop for {{n}}` resolves n at runtime and runs that many times.
func TestExecuteFile_LoopForVariableRunsNTimes(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### loop
# @loop for {{n}}
GET {{server}}/loop`).and().
		aClient(rc.WithVars(map[string]any{"n": 2}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2)
}

// loop counts of 0 and negative values produce zero HTTP requests, zero response entries, and no execution error.
func TestExecuteFile_LoopZeroOrNegativeIsNoop(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("loop_zero_negative.http").and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(0).and().
		responseCount(0)
}

// a 2-element collection produces 2 hits with `{{item.name}}` substituted per element.
func TestExecuteFile_LoopOfCollectionRunsPerElement(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("loop_collection.http").and().
		aClient(rc.WithVars(map[string]any{
			"items": []any{
				map[string]any{"name": "first"},
				map[string]any{"name": "second"},
			},
		}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2).and().
		capturedJSONFieldIs(0, "first", "name").and().
		capturedJSONFieldIs(1, "second", "name")
}

// per-iteration `{{$index}}` is substituted as the 0-based index.
func TestExecuteFile_LoopIndexOfCollectionAvailable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("loop_collection.http").and().
		aClient(rc.WithVars(map[string]any{
			"items": []any{
				map[string]any{"name": "first"},
				map[string]any{"name": "second"},
			},
		}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2).and().
		capturedJSONFieldIs(0, "0", "i").and().
		capturedJSONFieldIs(1, "1", "i")
}

// named looped request exposes responses as name0, name1, ... addressable via `{{nameN.response.body.X}}`.
func TestExecuteFile_LoopNamedResponsesAddressableAsNameN(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("loop_named_resp.http").and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(3).and().
		serverReceivedMethodAndPath(2, http.MethodGet, "/use").and().
		serverReceivedHeaderValue(2, "X-Loop0", "0").and().
		serverReceivedHeaderValue(2, "X-Loop1", "1")
}

// an empty collection produces zero HTTP requests and zero response entries with no execution error.
func TestExecuteFile_LoopEmptyCollectionIsNoop(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("loop_collection.http").and().
		aClient(rc.WithVars(map[string]any{"items": []any{}}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(0).and().
		responseCount(0)
}

// a non-array collection variable yields a clear execution error naming the collection.
func TestExecuteFile_LoopNonArrayCollectionErrors(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileFromTemplate("loop_collection.http").and().
		aClient(rc.WithVars(map[string]any{"items": "not an array"}))

	when.
		executeFile()

	then.
		errorContains("@loop", "items")
}

// a collection variable that is never defined yields a runtime error naming the missing variable.
func TestExecuteFile_LoopUnknownCollectionVarErrors(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### missing
# @loop for item of missing
GET {{server}}/items`).and().
		aClient()

	when.
		executeFile()

	then.
		errorContains("undefined variable", "missing")
}

// `# @loop for item of items` with a []string programmatic variable loops per element (liftStrings).
func TestExecuteFile_LoopOfStringsCollectionRunsPerElement(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### loop
# @loop for item of items
GET {{server}}/{{item}}`).and().
		aClient(rc.WithVars(map[string]any{"items": []string{"alpha", "beta"}}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2).and().
		tracking("segment").and().
		capturedURLSegment(0).and().
		trackedValueIs("alpha").and().
		capturedURLSegment(1).and().
		trackedValueIs("beta")
}

// A JSON-encoded string collection variable decodes into elements (decodeStringCollection).
func TestExecuteFile_LoopOfJsonEncodedCollectionRunsPerElement(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### loop
# @loop for item of items
GET {{server}}/{{item}}`).and().
		aClient(rc.WithVars(map[string]any{"items": `["one","two","three"]`}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(3).and().
		tracking("segment").and().
		capturedURLSegment(2).and().
		trackedValueIs("three")
}

// An empty JSON-encoded string collection produces zero requests.
func TestExecuteFile_LoopOfJsonEncodedEmptyCollectionIsNoop(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### loop
# @loop for item of items
GET {{server}}/{{item}}`).and().
		aClient(rc.WithVars(map[string]any{"items": "   "}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(0)
}

// A non-JSON string collection errors with the collection name.
func TestExecuteFile_LoopOfJsonEncodedMalformedCollectionErrors(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### loop
# @loop for item of items
GET {{server}}/{{item}}`).and().
		aClient(rc.WithVars(map[string]any{"items": "{not-json"}))

	when.
		executeFile()

	then.
		errorContains("is not an array")
}

// A collection of JSON numbers loop-substitutes as bare JSON numbers.
func TestExecuteFile_LoopOfNumericCollectionSubstitutesNumbers(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### loop
# @loop for item of items
GET {{server}}/id/{{item}}`).and().
		aClient(rc.WithVars(map[string]any{"items": []any{float64(7), float64(42)}}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2).and().
		tracking("segment").and().
		capturedURLSegmentAt(0, 2).and().
		trackedValueIs("7").and().
		capturedURLSegmentAt(1, 2).and().
		trackedValueIs("42")
}

// A collection of objects loop-substitutes as compact JSON per element.
func TestExecuteFile_LoopOfObjectCollectionSubstitutesJSON(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### loop
# @loop for item of users
GET {{server}}/u/{{item}}`).and().
		aClient(rc.WithVars(map[string]any{"users": []any{
			map[string]any{"name": "ann"},
			map[string]any{"name": "joe"},
		}}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2).and().
		tracking("segment").and().
		capturedURLSegmentAt(0, 2).and().
		trackedValueIs(`{"name":"ann"}`).and().
		capturedURLSegmentAt(1, 2).and().
		trackedValueIs(`{"name":"joe"}`)
}

// A nil collection element substitutes as an empty string.
func TestExecuteFile_LoopOfCollectionWithNilElementSubstitutesEmpty(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### loop
# @loop for item of items
GET {{server}}/x/{{item}}`).and().
		aClient(rc.WithVars(map[string]any{"items": []any{"a", nil, "c"}}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(3).and().
		tracking("segment").and().
		capturedURLSegmentAt(0, 2).and().
		trackedValueIs("a").and().
		capturedURLSegmentAt(1, 2).and().
		trackedValueIs("").and().
		capturedURLSegmentAt(2, 2).and().
		trackedValueIs("c")
}

func TestParseFile_DisabledDirectiveSetsFlag(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### disabled
# @disabled
GET https://example.com/api`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestDisabled(0, true)
}

func TestParseFile_DisabledDirectiveSlashSlashStyle(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### disabled
// @disabled
GET https://example.com/api`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestDisabled(0, true)
}

func TestExecuteFile_DisabledRequest_SkipsWithoutHTTPCall(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### disabled
# @disabled
GET {{server}}/skipped`).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(0)
}

func TestExecuteFile_DisabledAmongEnabled_KeepsStableIndexing(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "disabled_then_enabled.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		responseCount(2).and().
		requestSkipped(0).and().
		requestExecuted(1).and().
		capturedRequestCount(1)
}

func TestExecuteFile_DisabledIsNotAnError(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### disabled
# @disabled
GET https://disabled.invalid/api`).and().
		aClient()

	when.
		executeFile()

	then.
		noError()
}

// parser stores the conditional expression text after `!` and does NOT set the plain Disabled flag.
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

// `flag=true` -> request skipped; the test server receives zero hits, response entry is marked Skipped.
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

// `flag=false` -> request runs; the test server receives one hit, response entry is NOT marked Skipped.
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

// numeric `flag=0` -> request runs.
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

// numeric `flag=1` -> request skipped.
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

// substituted expr is empty -> request runs.
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

// substituted expr is any non-empty non-"true"/non-zero string -> request skipped.
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

// `{{missing}}` references an undefined variable; execution must surface an error naming it, NOT skip silently.
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

// `flag="TRUE"` -> request skipped (case-insensitive truthy).
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

// `flag="False"` -> request runs (case-insensitive falsy).
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

// `# @sleep 250` parses and stores 250ms duration.
func TestParseFile_SleepDirectiveStoresDuration(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### sleeping
# @sleep 250
GET https://example.com/api`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestSleep(0, 250*time.Millisecond)
}

// `// @sleep 250` also parses (mirror of disabled).
func TestParseFile_SleepDirectiveSlashSlashStyle(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### sleeping
// @sleep 250
GET https://example.com/api`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestSleep(0, 250*time.Millisecond)
}

// TestParseFile_SleepNonNumericArgErrors verifies @sleep with a non-numeric arg fails to parse and names the directive.
func TestParseFile_SleepNonNumericArgErrors(t *testing.T) {
	given, when, then := newParts(t)
	given.
		aHttpFile(`### sleeping
# @sleep abc
GET https://example.com/api`).and().
		aClient()
	when.parsingFile()
	then.parseErrorContains("@sleep")
}

// TestParseFile_SleepNegativeArgErrors verifies @sleep with a negative arg fails to parse and names the directive.
func TestParseFile_SleepNegativeArgErrors(t *testing.T) {
	given, when, then := newParts(t)
	given.
		aHttpFile(`### sleeping
# @sleep -5
GET https://example.com/api`).and().
		aClient()
	when.parsingFile()
	then.parseErrorContains("@sleep")
}

// TestParseFile_SleepMissingArgErrors verifies @sleep with no arg fails to parse and names the directive.
func TestParseFile_SleepMissingArgErrors(t *testing.T) {
	given, when, then := newParts(t)
	given.
		aHttpFile(`### sleeping
# @sleep
GET https://example.com/api`).and().
		aClient()
	when.parsingFile()
	then.parseErrorContains("@sleep")
}

// single request with `# @sleep 200` must take >= ~200ms before sending; measure elapsed with loose lower bound.
func TestExecuteFile_SleepWaitsBeforeSend(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### sleeping
# @sleep 200
GET {{server}}/slow`).and().
		aClient()

	start := time.Now()
	when.
		executeFile()
	elapsed := time.Since(start)

	then.
		noError().and().
		capturedRequestCount(1)
	// Loose lower bound (180ms) — accounts for timer slop and go-routine scheduling.
	then.assert.GreaterOrEqual(elapsed, 180*time.Millisecond,
		"@sleep 200 should delay send; elapsed=%s", elapsed)
}

// `@sleep 0` is a valid no-op — runs immediately, no error, 1 hit.
func TestExecuteFile_SleepZeroIsNoop(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### instant
# @sleep 0
GET {{server}}/instant`).and().
		aClient()

	start := time.Now()
	when.
		executeFile()
	elapsed := time.Since(start)

	then.
		noError().and().
		capturedRequestCount(1)
	// Loose upper bound (1s) — generous, just guards against accidentally introducing a delay.
	then.assert.Less(elapsed, time.Second,
		"@sleep 0 should be a no-op; elapsed=%s", elapsed)
}

// two requests, only second has `# @sleep 200`; inter-request gap between server-side timestamps >= ~200ms.
func TestExecuteFile_SleepAppliesPerRequest(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### first
GET {{server}}/first

### second
# @sleep 200
GET {{server}}/second`).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2).and().
		gapBetweenCapturedRequestTimesAtLeast(0, 1, 180*time.Millisecond)
}
