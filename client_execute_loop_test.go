package restclient_test

import (
	"net/http"
	"testing"

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
