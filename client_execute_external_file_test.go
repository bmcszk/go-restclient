package restclient_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	rc "github.com/bmcszk/go-restclient"
)

// decodeJSONBody reads the whole request body and unmarshals it into v.
func decodeJSONBody(r *http.Request, v any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, v)
}

// writeJSONResponse encodes v as JSON into the response writer.
func writeJSONResponse(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}

// readAllBody reads and returns the whole request body.
func readAllBody(r *http.Request) ([]byte, error) {
	return io.ReadAll(r.Body)
}

// encodingCase is the table row for TestClientExecuteFileWithEncoding.
type encodingCase struct {
	name             string
	encodingName     string // e.g. "latin1", "cp1252"; written into the <@ directive
	contentToWrite   string // raw content, encoded via encoder before writing
	expectedUTF8Body string // body the server must receive after decoding
	encoder          transform.Transformer
}

// PRD-COMMENT: FR4.1 - Request Body: External File with Variables (<@)
func TestExecuteFile_ExternalFileWithVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %v, want %v", r.Method, http.MethodPost)
			}

			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}

			var data map[string]any
			if err := decodeJSONBody(r, &data); err != nil {
				t.Errorf("invalid JSON body: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = writeJSONResponse(w, map[string]any{"json": data})
		}).and().
		anExternalFile("test_vars.json", `{
  "userId": "{{userId}}",
  "name": "{{userName}}",
  "timestamp": "{{$timestamp}}",
  "environment": "{{env}}"
}`).and().
		aHttpFile(fmt.Sprintf(`@userId = user123
@userName = John Doe
@env = testing

### External File with Variable Substitution
POST %s/post
Content-Type: application/json

<@ ./test_vars.json`, given.serverURL)).and().
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

// PRD-COMMENT: FR4.2 - Request Body: External File Static (<)
func TestExecuteFile_ExternalFileWithoutVariables(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %v, want %v", r.Method, http.MethodPost)
			}

			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}

			var data map[string]any
			if err := decodeJSONBody(r, &data); err != nil {
				t.Errorf("invalid JSON body: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = writeJSONResponse(w, map[string]any{"json": data})
		}).and().
		anExternalFile("test_static.json", `{
  "userId": "{{userId}}",
  "name": "{{userName}}",
  "literal": "this should stay as-is"
}`).and().
		aHttpFile(fmt.Sprintf(`@userId = user123
@userName = John Doe

### External File without Variable Substitution
POST %s/post
Content-Type: application/json

< ./test_static.json`, given.serverURL)).and().
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

// PRD-COMMENT: FR4.3 - Request Body: External File with Encoding (<@|encoding)
func TestClientExecuteFileWithEncoding(t *testing.T) {
	tests := []encodingCase{
		{
			name:             "Latin-1 encoded file",
			encodingName:     "latin1",
			contentToWrite:   "H\u00e4llo W\u00f6rld! \u00d1ice to meet you. ?",
			expectedUTF8Body: "Hällo Wörld! Ñice to meet you. ?",
			encoder:          charmap.ISO8859_1.NewEncoder(),
		},
		{
			name:             "CP1252 (Windows-1252) encoded file",
			encodingName:     "cp1252",
			contentToWrite:   "H\u00e4llo W\u00f6rld! \u00d1ice to meet you. \u20ac\u2122",
			expectedUTF8Body: "Hällo Wörld! Ñice to meet you. €™",
			encoder:          charmap.Windows1252.NewEncoder(),
		},
		{
			name:             "ASCII encoded file (as subset of UTF-8)",
			encodingName:     "ascii",
			contentToWrite:   "Hello World! Nice to meet you.",
			expectedUTF8Body: "Hello World! Nice to meet you.",
			encoder:          nil,
		},
		{
			name:             "UTF-8 encoded file (explicit)",
			encodingName:     "utf-8",
			contentToWrite:   "Hällo Wörld! Ñice to meet you. €😊",
			expectedUTF8Body: "Hällo Wörld! Ñice to meet you. €😊",
			encoder:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			given, when, then := newParts(t)

			var encoded []byte

			if tt.encoder != nil {
				encoded, _, _ = transform.Bytes(tt.encoder, []byte(tt.contentToWrite))
			} else {
				encoded = []byte(tt.contentToWrite)
			}

			given.
				aHttpServer(func(w http.ResponseWriter, r *http.Request) {
					body, _ := readAllBody(r)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(body)
				}).and().
				anExternalFileBytes("encoded_body.txt", encoded).and().
				aHttpFile(fmt.Sprintf(
					"POST %s\nContent-Type: text/plain\n\n<@%s encoded_body.txt",
					given.serverURL, tt.encodingName)).and().
				aClient()

			when.
				executeFile()

			then.
				responseCount(1).and().
				noError().and().
				responseAt(0).and().
				responseCode(http.StatusOK).and().
				serverReceivedBodyIs(0, tt.expectedUTF8Body)
		})
	}
}

// PRD-COMMENT: FR4.3 - Request Body: External File with Encoding (<@|encoding)
func TestExecuteFile_ExternalFileWithEncoding(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %v, want %v", r.Method, http.MethodPost)
			}

			if got := r.Header.Get("Content-Type"); got != "text/plain" {
				t.Errorf("Content-Type = %q, want text/plain", got)
			}

			body, _ := readAllBody(r)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}).and().
		anExternalFile("test_encoding.txt", "Café français: été, naïve, résumé").and().
		aHttpFile(fmt.Sprintf(`### External File with UTF-8 Encoding
POST %s/post
Content-Type: text/plain

<@utf-8 ./test_encoding.txt`, given.serverURL)).and().
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

// PRD-COMMENT: FR4.5 / FR4.6 - External File with Variables and Encoding (<@encoding)
func TestExecuteFile_ExternalFileWithVariablesAndEncoding(t *testing.T) {
	given, when, then := newParts(t)

	encoded, _, _ := transform.Bytes(
		charmap.ISO8859_1.NewEncoder(),
		[]byte("name={{name}}, city={{city}}, id={{id}}"),
	)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %v, want %v", r.Method, http.MethodPost)
			}

			if got := r.Header.Get("Content-Type"); got != "text/plain; charset=iso-8859-1" {
				t.Errorf("Content-Type = %q, want text/plain; charset=iso-8859-1", got)
			}

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}).and().
		anExternalFileBytes("vars_latin1.txt", encoded).and().
		aHttpFile(fmt.Sprintf(`@name = TestName
@id = 12345

### External File with Variable Substitution and Encoding
POST %s/post
Content-Type: text/plain; charset=iso-8859-1

<@latin1 ./vars_latin1.txt`, given.serverURL)).and().
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

// PRD-COMMENT: FR1.1 - File Type: .rest extension support
func TestExecuteFile_WithRestExtension(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %v, want %v", r.Method, http.MethodGet)
			}

			if r.URL.Path != "/get_test_rest_extension" {
				t.Errorf("path = %q, want /get_test_rest_extension", r.URL.Path)
			}

			if got := r.Header.Get("X-Test-Header-Rest"); got != "rest-extension-test-value" {
				t.Errorf("X-Test-Header-Rest = %q, want rest-extension-test-value", got)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "ok from .rest"}`))
		}).and().
		anExternalFile("test_request.rest", fmt.Sprintf(`### Test Request with .rest extension
GET %s/get_test_rest_extension
X-Test-Header-Rest: rest-extension-test-value
`, given.serverURL)).and().
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

// PRD-COMMENT: FR4.4 - Request Body: External File Not Found
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
