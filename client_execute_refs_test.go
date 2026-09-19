package restclient_test

import (
	"testing"
)

// parsedRequestRefs asserts the parsed request at index i has a ref entry with the given name and force flag.
// RED: rc.RequestRef and Request.Refs are not yet defined in the production code.
func (p *parts) parsedRequestRefs(i int, wantName string, wantForce bool) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	req := p.parsedFile.Requests[i]
	p.require.NotEmpty(req.Refs, "request %d has no refs", i)

	found := false
	for _, ref := range req.Refs {
		if ref.Name == wantName {
			p.assert.Equal(wantForce, ref.Force,
				"ref %q force flag (expected %v)", wantName, wantForce)
			found = true

			break
		}
	}
	p.require.True(found, "request %d missing ref %q", i, wantName)

	return p
}

// TestExecuteFile_RequestRefs_ParserRecordsRefAndForceRef verifies the parser stores
// `@ref` and `@forceRef` metadata on the Request struct, preserving declaration order.
func TestExecuteFile_RequestRefs_ParserRecordsRefAndForceRef(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aTemplateFixture("http_request_files", "refs_parse.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestCount(1).and().
		parsedRequestName(0, "collector").and().
		parsedRequestRefs(0, "login", false).and().
		parsedRequestRefs(0, "refresh", true)
}

// TestExecuteFile_RequestRefs_RefRunsReferencedBeforeReferencing verifies the executor
// runs the referenced `login` request before the referencing one.
func TestExecuteFile_RequestRefs_RefRunsReferencedBeforeReferencing(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "refs_chain.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		requestCount(2).and().
		capturedRequestPathIs(0, "/login").and().
		capturedRequestPathIs(1, "/protected")
}

// TestExecuteFile_RequestRefs_RefCachedWithinRun verifies a `@ref`-ed request runs once
// per file and is cached for subsequent references within the same execution.
func TestExecuteFile_RequestRefs_RefCachedWithinRun(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "refs_cache.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		requestCount(3).and().
		capturedRequestCount(3)
}

// TestExecuteFile_RequestRefs_ForceRefReruns verifies `@forceRef` always re-executes
// the referenced request, ignoring any cached response.
func TestExecuteFile_RequestRefs_ForceRefReruns(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "refs_forceref.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		requestCount(4).and().
		capturedRequestCount(4)
}

// TestExecuteFile_RequestRefs_CycleFails verifies a cyclic `@ref` graph produces an
// execution error mentioning the cycle.
func TestExecuteFile_RequestRefs_CycleFails(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "refs_cycle.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		errorContains("cycle")
}

// TestExecuteFile_RequestRefs_UnknownRefFails verifies `@ref` pointing at an undefined
// request name produces an execution error mentioning that name.
func TestExecuteFile_RequestRefs_UnknownRefFails(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "refs_unknown.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		errorContains("ghost")
}
