package restclient_test

import (
	"net/http"
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

// TestExecuteFile_RequestRefs_TransitiveChain verifies the executor resolves a 3-deep
// @ref chain depth-first: requesting `a` (which refs `b`, which refs `c`) must execute
// `c`, then `b`, then `a`, with each ref still using the cache-only `# @ref` semantics.
func TestExecuteFile_RequestRefs_TransitiveChain(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "refs_transitive.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		requestCount(3).and().
		capturedRequestPathIs(0, "/c").and().
		capturedRequestPathIs(1, "/b").and().
		capturedRequestPathIs(2, "/a")
}

// TestExecuteFile_RequestRefs_RefResponseUsableInReferencingRequest verifies that
// the `@ref`-ed request runs before the referencing request even when the referencing
// request appears FIRST in the file, and that its response variables
// (`{{name.response.body.x}}`, `{{name.response.headers.X}}`, `{{name.response.status}}`)
// resolve correctly in the referencing request.
func TestExecuteFile_RequestRefs_RefResponseUsableInReferencingRequest(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "refs_respvar.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		requestCount(2).and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/login").and().
		serverReceivedMethodAndPath(1, http.MethodPost, "/use").and().
		serverReceivedHeaderValue(1, "X-Trace", "tok-123").and().
		serverReceivedHeaderValue(1, "X-Via", "text/plain; charset=utf-8").and().
		capturedBodyContains(1, `"statusRef":"200"`)
}

// TestExecuteFile_RequestImports_CrossFileRefAndVars verifies `@import ./file.http` exposes
// the imported file's file-global variables and named requests to the importing file:
// the importing request substitutes `{{importedBase}}` in the URL and
// `{{token.response.body.token}}` in a header from the imported named request.
func TestExecuteFile_RequestImports_CrossFileRefAndVars(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "refs_import_def.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aTemplateFixture("http_request_files", "refs_import_use.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		requestCount(2).and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/token").and().
		serverReceivedMethodAndPath(1, http.MethodPost, "/users").and().
		serverReceivedHeaderValue(1, "X-Token", "tok-9")
}

// TestExecuteFile_RequestImports_MissingFileFails verifies an `@import` pointing at a file
// that does not exist produces an error naming the missing path.
func TestExecuteFile_RequestImports_MissingFileFails(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "refs_import_missing.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		errorContains("does_not_exist.http")
}

// TestExecuteFile_RequestImports_ImportedRequestRefCached verifies the `@ref` cache semantics
// carry over to imported-file named requests: two importing requests both referencing the
// same imported named request run the imported request exactly once.
func TestExecuteFile_RequestImports_ImportedRequestRefCached(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aTemplateFixture("http_request_files", "refs_import_def.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aTemplateFixture("http_request_files", "refs_import_use_cache.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		requestCount(3).and().
		capturedRequestPathIs(0, "/token").and().
		capturedRequestPathIs(1, "/users/a").and().
		capturedRequestPathIs(2, "/users/b")
}
