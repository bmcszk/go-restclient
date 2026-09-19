package restclient_test

import (
	"net/http"
	"testing"
)

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

// referencing request comes first in the fixture — resolution is name-based, not file-order
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
