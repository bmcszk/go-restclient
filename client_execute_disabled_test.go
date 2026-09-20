package restclient_test

import (
	"testing"
)

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

// requestSkipped asserts the response at index i is marked as skipped.
func (p *parts) requestSkipped(i int) *parts {
	p.require.Greater(len(p.responses), i)
	p.assert.True(p.responses[i].Skipped)

	return p
}

// requestExecuted asserts the response at index i is not marked as skipped.
func (p *parts) requestExecuted(i int) *parts {
	p.require.Greater(len(p.responses), i)
	p.assert.False(p.responses[i].Skipped)

	return p
}
