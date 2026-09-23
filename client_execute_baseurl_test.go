package restclient_test

import (
	"fmt"
	"testing"

	rc "github.com/bmcszk/go-restclient"
)

// path is deliberately not merged for absolute request paths (applyBaseURLIfNeeded).
func TestExecuteFile_WithBaseURL_AbsolutePathDropsBasePath(t *testing.T) {
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

// base URL including its path (applyBaseURLIfNeeded relative branch).
func TestExecuteFile_WithBaseURL_RelativePathPrependsFullPath(t *testing.T) {
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
func TestExecuteFile_WithBaseURL_AbsoluteRequestURLIgnored(t *testing.T) {
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
func TestExecuteFile_WithDefaultHeader_SentWithRequest(t *testing.T) {
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

// parseAndSanitizeBaseURL rejects an unparseable BaseURL with a clear error.
func TestExecuteFile_WithBaseURL_InvalidErrors(t *testing.T) {
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
