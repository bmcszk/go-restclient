package restclient_test

import (
	"io"
	"net/http"
	"testing"

	"golang.org/x/text/encoding/charmap"

	rc "github.com/bmcszk/go-restclient"
)

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
