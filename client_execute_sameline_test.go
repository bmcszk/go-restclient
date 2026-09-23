package restclient_test

import (
	"fmt"
	"testing"
)

// `GET url ### NextName` splits the request and names the next one on the same line.
func TestExecuteFile_SameLineSeparatorStartsNewNamedRequest(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(fmt.Sprintf(
			"### first\nGET %s/one ### second\nGET %s/two\n",
			given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2).and().
		tracking("segment").and().
		capturedURLSegmentAt(0, 1).and().
		trackedValueIs("one").and().
		capturedURLSegmentAt(1, 1).and().
		trackedValueIs("two")
}

// Same-line separator without a name still splits requests.
func TestExecuteFile_SameLineSeparatorWithoutNameSplits(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(fmt.Sprintf(
			"### first\nGET %s/one ###\nGET %s/two\n",
			given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2).and().
		tracking("segment").and().
		capturedURLSegmentAt(0, 1).and().
		trackedValueIs("one").and().
		capturedURLSegmentAt(1, 1).and().
		trackedValueIs("two")
}
