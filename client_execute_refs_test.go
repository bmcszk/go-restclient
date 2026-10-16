package restclient_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestExecuteFile_RequestRefs_ParserRecordsRefAndForceRef(t *testing.T) {
	given, when, then := newCaptureParts(t)

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
	given, when, then := newCaptureParts(t)

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
	given, when, then := newCaptureParts(t)

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
	given, when, then := newCaptureParts(t)

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
	given, when, then := newCaptureParts(t)

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
	given, when, then := newCaptureParts(t)

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
	given, when, then := newCaptureParts(t)

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
	given, when, then := newCaptureParts(t)

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
	given, when, then := newCaptureParts(t)

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
	given, when, then := newCaptureParts(t)

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
	given, when, then := newCaptureParts(t)

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

func TestResponseRef_StatusCode(t *testing.T) {
	given, when, then := newCaptureParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"id": 42}`)
		}).and().
		aHttpFile(fmt.Sprintf("### create\n"+
			"POST %s/items\n"+
			"Content-Type: application/json\n\n"+
			`{"name":"test"}`+"\n\n"+
			"###\n"+
			"### check status\n"+
			"GET %s/status/{{create.response.status}}\n", given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusCreated).and().
		responseAt(1).and().
		requestURLContains("/status/201")
}

func TestResponseRef_BodyField(t *testing.T) {
	given, when, then := newCaptureParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/login" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"access_token":"abc123","token_type":"Bearer"}`)

				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "protected data")
		}).and().
		aHttpFile(fmt.Sprintf("### login\n"+
			"POST %s/login\n"+
			"Content-Type: application/json\n\n"+
			`{"user":"admin","pass":"secret"}`+"\n\n"+
			"###\n"+
			"### access protected\n"+
			"GET %s/protected\n"+
			"Authorization: Bearer {{login.response.body.access_token}}\n",
			given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		capturedRequestCount(2).and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/login").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/protected").and().
		responseAt(1).and().
		requestHeaderIs("Authorization", "Bearer abc123")
}

func TestResponseRef_HeaderValue(t *testing.T) {
	given, when, then := newCaptureParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Session-Id", "sess-999")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFile(fmt.Sprintf("### first\n"+
			"GET %s/init\n\n"+
			"###\n"+
			"### second\n"+
			"GET %s/use\n"+
			"X-Session: {{first.response.headers.X-Session-Id}}\n",
			given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		responseAt(1).and().
		requestHeaderIs("X-Session", "sess-999")
}

func TestResponseRef_NestedBodyField(t *testing.T) {
	given, when, then := newCaptureParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"data":{"user":{"name":"Alice","id":123}}}`)
		}).and().
		aHttpFile(fmt.Sprintf("### get user\n"+
			"GET %s/user\n\n"+
			"###\n"+
			"### use name\n"+
			"GET %s/greet/{{get user.response.body.data.user.name}}\n",
			given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		responseAt(1).and().
		requestURLContains("/greet/Alice")
}

func TestResponseRef_ArrayIndex(t *testing.T) {
	given, when, then := newCaptureParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"items":["alpha","beta","gamma"]}`)
		}).and().
		aHttpFile(fmt.Sprintf("### list\n"+
			"GET %s/list\n\n"+
			"###\n"+
			"### get second\n"+
			"GET %s/item/{{list.response.body.items[1]}}\n",
			given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		responseAt(1).and().
		requestURLContains("/item/beta")
}

func TestResponseRef_MissingName(t *testing.T) {
	given, when, then := newCaptureParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aHttpFile(fmt.Sprintf("### actual\n"+
			"GET %s/ok\n\n"+
			"###\n"+
			"### ref non-existent\n"+
			"GET %s/ref/{{nonexistent.response.status}}\n",
			given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		responseAt(1).and().
		requestURLContains("/ref/")
}

func TestResponseRef_BodyWhole(t *testing.T) {
	given, when, then := newCaptureParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = fmt.Fprint(w, "raw-body-content")
		}).and().
		aHttpFile(fmt.Sprintf("### get\n"+
			"GET %s/data\n\n"+
			"###\n"+
			"### echo\n"+
			"POST %s/echo\n"+
			"Content-Type: text/plain\n\n"+
			"{{get.response.body}}\n", given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		responseAt(1).and().
		requestRawBodyIs("raw-body-content")
}
