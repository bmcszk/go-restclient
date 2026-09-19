package restclient_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestParseFile_SingleRequest(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFile(fmt.Sprintf("GET %s/health\n", given.serverURL)).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestCount(1).and().
		parsedRequestMethod(0, http.MethodGet).and().
		parsedRequestRawURLIs(0, given.serverURL+"/health")
}

func TestParseFile_MultipleRequests(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile("GET http://example.com/a\n###\n" +
			"POST http://example.com/b\nContent-Type: application/json\n\n{\"x\":1}\n").and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestCount(2).and().
		parsedRequestMethod(0, http.MethodGet).and().
		parsedRequestMethod(1, http.MethodPost)
}

func TestParseFile_NamedRequests(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile("### get users\nGET http://example.com/users\n###\n" +
			"### create user\nPOST http://example.com/users\n").and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestCount(2).and().
		parsedRequestName(0, "get users").and().
		parsedRequestName(1, "create user")
}

func TestParseFile_NoRequests(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile("# just a comment\n").and().
		aClient()

	when.
		parsingFile()

	then.
		parseErrorContains("no requests found")
}

func TestParseFile_FileNotFound(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aMissingFile("/nonexistent/file.http").and().
		aClient()

	when.
		parsingFile()

	then.
		parseErrorContains()
}

func TestExecuteRequest_ByIndex(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response1")
		}).and().
		aHttpFile(fmt.Sprintf("GET %s/a\n###\nGET %s/b\n", given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executingRequestAt(0)

	then.
		noError().and().
		responseCount(1).and().
		responseAt(0).and().
		responseHasNoError().and().
		responseBodyIs("response1").and().
		serverReceivedMethodAndPath(0, http.MethodGet, "/a")
}

func TestExecuteRequest_OutOfRange(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile("GET http://example.com/a\n").and().
		aClient()

	when.
		executingRequestAt(5)

	then.
		errorContains("out of range")
}

func TestExecuteRequest_NegativeIndex(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile("GET http://example.com/a\n").and().
		aClient()

	when.
		executingRequestAt(-1)

	then.
		errorContains("out of range")
}

func TestExecuteRequest_SecondOfThree(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aHttpFile(fmt.Sprintf("GET %s/1\n###\nGET %s/2\n###\nGET %s/3\n",
			given.serverURL, given.serverURL, given.serverURL)).and().
		aClient()

	when.
		executingRequestAt(1)

	then.
		noError().and().
		responseCount(1).and().
		responseAt(0).and().
		responseHasNoError().and().
		capturedRequestCount(1).and().
		serverReceivedMethodAndPath(0, http.MethodGet, "/2")
}
