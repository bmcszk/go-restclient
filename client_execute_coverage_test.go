package restclient_test

import (
	"fmt"
	"testing"

	rc "github.com/bmcszk/go-restclient"
)

// --- BaseURL joining (client_url_utils.go) ---

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
func TestExecuteFile_LoopOfJsonEncodedStringCollectionRunsPerElement(t *testing.T) {
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
func TestExecuteFile_LoopOfEmptyJsonCollectionIsNoop(t *testing.T) {
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
func TestExecuteFile_LoopOfMalformedJsonCollectionErrors(t *testing.T) {
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

// BaseURL path + request path starting with "/" takes scheme+host only — the base
// path is deliberately not merged for absolute request paths (applyBaseURLIfNeeded).
func TestExecuteFile_BaseURLPathJoinsWithRequestPath(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### users
GET /api/users`).and().
		aClient(rc.WithBaseURL(given.serverURL + "/v1"))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		tracking("segment").and().
		capturedURLSegmentAt(0, 1).and().
		trackedValueIs("api").and().
		capturedURLSegmentAt(0, 2).and().
		trackedValueIs("users")
}

// BaseURL path + relative request path without leading "/" prepends the full
// base URL including its path (applyBaseURLIfNeeded relative branch).
func TestExecuteFile_BaseURLResolvesRelativeReference(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### users
GET api/users`).and().
		aClient(rc.WithBaseURL(given.serverURL + "/v1"))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		tracking("segment").and().
		capturedURLSegmentAt(0, 1).and().
		trackedValueIs("v1").and().
		capturedURLSegmentAt(0, 2).and().
		trackedValueIs("api").and().
		capturedURLSegmentAt(0, 3).and().
		trackedValueIs("users")
}

// An absolute request URL ignores BaseURL entirely (resolveWithBaseURL IsAbs branch).
func TestExecuteFile_AbsoluteRequestURLIgnoresBaseURL(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(fmt.Sprintf("### direct\nGET %s/absolute", given.serverURL)).and().
		aClient(rc.WithBaseURL("https://ignored.example.com/base"))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		tracking("segment").and().
		capturedURLSegment(0).and().
		trackedValueIs("absolute")
}

// Default headers set via options are sent with every request (addDefaultHeaders).
func TestExecuteFile_DefaultHeadersAreSent(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### h
GET {{server}}/h`).and().
		aClient(rc.WithDefaultHeader("X-Custom-Trace", "fluent-42"))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		tracking("trace").and().
		capturedHeaderField(0, "X-Custom-Trace").and().
		trackedValueIs("fluent-42")
}

// {{$randomPassword N}} substitutes an N-char password from the default charset.
func TestExecuteFile_RandomPasswordSubstitutesConfiguredLength(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### pw
POST {{server}}/pw

{{$randomPassword 8}}`).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		serverReceivedBodyLengthIs(0, 8)
}

// programmatic "password.charset" overrides the random-password charset.
func TestExecuteFile_RandomPasswordCharsetOverride(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### pw
POST {{server}}/pw

{{$randomPassword 6}}`).and().
		aClient(rc.WithVars(map[string]any{
			"password": map[string]string{"charset": "xyz"},
		}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		serverReceivedBodyMatches(0, `^[xyz]{6}$`)
}

// A malformed random-password length leaves the placeholder unresolved.
func TestParseFile_MalformedRandomPasswordStaysRaw(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### pw
POST https://example.com/pw

{{$randomPassword -3}}`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestBodyIs(0, "{{$randomPassword -3}}")
}

// parseAndSanitizeBaseURL rejects an unparseable BaseURL with a clear error.
func TestExecuteFile_InvalidBaseURLErrors(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### rel
GET /relative`).and().
		aClient(rc.WithBaseURL("ht tp://bad url"))

	when.
		executeFile()

	then.
		errorContains("invalid BaseURL")
}

// serverReceivedBodyLengthIs asserts the captured request body byte length.
func (p *parts) serverReceivedBodyLengthIs(index int, want int) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.require.Equal(want, len(p.capturedBodies[index]))

	return p
}

// serverReceivedBodyMatches asserts the captured request body against a regexp.
func (p *parts) serverReceivedBodyMatches(index int, pattern string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedBodies[index])

	return p
}

// trackedValueIs asserts the most recent tracked value equals want.
func (p *parts) trackedValueIs(want string) *parts {
	values, ok := p.trackedValues[p.activeTrack]
	p.require.True(ok, "no tracking bucket %q active", p.activeTrack)
	p.require.NotEmpty(values)
	p.assert.Equal(want, values[len(values)-1])

	return p
}
