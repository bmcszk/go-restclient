package restclient_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"golang.org/x/text/encoding/charmap"

	rc "github.com/bmcszk/go-restclient"
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

func TestExecuteFile_ExternalFileWithVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aJsonEchoServer().and().
		aHttpFileWithExternalFileFixture("test_vars.json").and().
		aClient(rc.WithVars(map[string]any{"userName": "Override Name"}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/post").and().
		serverReceivedHeaderValue(0, "Content-Type", "application/json").and().
		serverReceivedBodyParsesAsJSON(0).and().
		requestRawBodyContains(
			`"userId": "user123"`,
			`"name": "Override Name"`,
			`"environment": "testing"`,
			`"timestamp":`,
		).and().
		requestRawBodyMatchesRegexp(`"timestamp": "\d+"`)
}

func TestExecuteFile_ExternalFileWithoutVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aJsonEchoServer().and().
		aHttpFileWithExternalFileStaticFixture("test_static.json").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/post").and().
		serverReceivedHeaderValue(0, "Content-Type", "application/json").and().
		serverReceivedBodyParsesAsJSON(0).and().
		requestRawBodyContains(
			`"userId": "{{userId}}"`,
			`"name": "{{userName}}"`,
			`"literal": "this should stay as-is"`,
		)
}

//
// aHttpFileWithExternalFileEncoding so each test stays a single fluent chain.

func TestExecuteFile_ExternalFileWithEncoding_Latin1(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileWithExternalFileEncoding("latin1", "encoded_body.txt",
			[]byte("Hällo Wörld! Ñice to meet you. ?"), charmap.ISO8859_1.NewEncoder()).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		serverReceivedBodyIs(0, "Hällo Wörld! Ñice to meet you. ?")
}

func TestExecuteFile_ExternalFileWithEncoding_CP1252(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileWithExternalFileEncoding("cp1252", "encoded_body.txt",
			[]byte("Hällo Wörld! Ñice to meet you. €™"), charmap.Windows1252.NewEncoder()).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		serverReceivedBodyIs(0, "Hällo Wörld! Ñice to meet you. €™")
}

func TestExecuteFile_ExternalFileWithEncoding_ASCII(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileWithExternalFileEncoding("ascii", "encoded_body.txt",
			[]byte("Hello World! Nice to meet you."), nil).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		serverReceivedBodyIs(0, "Hello World! Nice to meet you.")
}

func TestExecuteFile_ExternalFileWithEncoding_UTF8(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFileWithExternalFileEncoding("utf-8", "encoded_body.txt",
			[]byte("Hällo Wörld! Ñice to meet you. €😊"), nil).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		serverReceivedBodyIs(0, "Hällo Wörld! Ñice to meet you. €😊")
}
func TestExecuteFile_ExternalFileWithEncoding(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}).and().
		anExternalFile("test_encoding.txt", "Café français: été, naïve, résumé").and().
		aHttpFileFromTemplate("external_file_with_utf8_encoding.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		requestRawBodyIs("Café français: été, naïve, résumé").and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/post").and().
		serverReceivedHeaderValue(0, "Content-Type", "text/plain")
}

func TestExecuteFile_ExternalFileWithVariablesAndEncoding(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}).and().
		anExternalFile("vars_latin1.txt", "name={{name}}, city={{city}}, id={{id}}").and().
		aHttpFileFromTemplate("external_file_with_vars_and_encoding.http").and().
		aClient(rc.WithVars(map[string]any{"city": "ProgrammaticCity"}))

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		requestRawBodyIs("name=TestName, city=ProgrammaticCity, id=12345").and().
		serverReceivedHeaderValue(0, "Content-Type", "text/plain; charset=iso-8859-1").and().
		serverReceivedLatin1BodyIs(0, "name=TestName, city=ProgrammaticCity, id=12345")
}

func TestExecuteFile_WithRestExtension(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aRestExtensionServer().and().
		aHttpFileFromTemplate("test_request_rest_extension.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		responseContains(`{"status": "ok from .rest"}`).and().
		serverReceivedMethodAndPath(0, http.MethodGet, "/get_test_rest_extension").and().
		serverReceivedHeaderValue(0, "X-Test-Header-Rest", "rest-extension-test-value")
}

func TestExecuteFile_ExternalFileNotFound(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### External File Not Found
POST https://httpbin.org/post
Content-Type: application/json

<@ ./nonexistent.json`).and().
		aClient()

	when.
		executeFile()

	then.
		errorContains("error processing body for request", "nonexistent.json").and().
		responseCount(0)
}

// TestExecuteFile_DotEnvProcessEnvPlaceholderExpands: .env values support {{$processEnv}}
// so secrets can come from the real environment (#49).
func TestExecuteFile_DotEnvProcessEnvPlaceholderExpands(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("ISS49_CID", "env-value").and().
		aDotEnvFile("CID={{$processEnv ISS49_CID}}").and().
		anEchoServer().and().
		aHttpFile("@client = {{CID}}\n\nPOST {{server}}/echo\nX-Client: {{client}}\n\nok").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-Client", "env-value")
}

// TestExecuteFile_DotEnvUnresolvablePlaceholderStaysRaw: missing process env leaves the
// {{$processEnv}} placeholder literal in the .env value (#49 pinned semantics).
func TestExecuteFile_DotEnvUnresolvablePlaceholderStaysRaw(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aDotEnvFile("CID={{$processEnv ISS49_MISSING}}").and().
		anEchoServer().and().
		aHttpFile("@client = {{CID}}\n\nPOST {{server}}/echo\nX-Client: {{client}}\n\nok").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-Client", "{{$processEnv ISS49_MISSING}}")
}

// TestExecuteFile_EnvFlagLoadsNamedEnvFile: --env staging loads .env first, then .env.staging
// overrides only the keys it defines (#48).
func TestExecuteFile_EnvFlagLoadsNamedEnvFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aDotEnvFile("A=base").and().
		aDotEnvNamedFile("staging", "A=staging\nB=two").and().
		anEchoServer().and().
		aHttpFile("@client = {{A}}\n\nPOST {{server}}/echo\nX-A: {{A}}\nX-B: {{B}}\n\nok").and().
		aClientWithEnvName("staging")

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-A", "staging").and().
		serverReceivedHeaderValue(0, "X-B", "two")
}

// TestExecuteFile_NamedEnvFilePlaceholderExpands: .env.staging values support
// {{$processEnv}} expansion like .env (#49 + #48 combined).
func TestExecuteFile_NamedEnvFilePlaceholderExpands(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("ISS49_S", "expanded-staging").and().
		aDotEnvFileRemoved().and().
		aDotEnvNamedFile("staging", "CID={{$processEnv ISS49_S}}").and().
		anEchoServer().and().
		aHttpFile("@client = {{CID}}\n\nPOST {{server}}/echo\nX-Client: {{client}}\n\nok").and().
		aClientWithEnvName("staging")

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-Client", "expanded-staging")
}
