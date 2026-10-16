package restclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"text/template"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rc "github.com/bmcszk/go-restclient"
)

const (
	responseFilesDir = "test/data/http_response_files"
)

// parts is the shared state for the fluent Given/When/Then test DSL.
type parts struct {
	*testing.T
	require *require.Assertions
	assert  *assert.Assertions
	client  *rc.Client
	// server holds the server domain state (the httptest servers and their
	// address variables); parts_test.go builds on it.
	serverParts
	// capture holds the capture/tracking domain state (recorded requests, bodies and
	// consistency-tracking buckets); parts_test.go builds on it.
	captureParts
	// parse holds the parse-domain state (the parsed .http file and its error);
	// parts_test.go builds on it.
	parseParts
	// file holds the file/fixture domain state (the .http file path the file
	// builders write and executeFile / parse read); parts_test.go builds on it.
	fileParts
	// env holds the env domain state (dotenv files, environment-backed client
	// builds); parts_test.go builds on it.
	envParts
	// oauth2 holds the oauth2 domain state (the OAuth2 token server address);
	// client_oauth2_test.go builds on it.
	oauth2Parts
	// cookies holds the cookie/redirect domain state (the cookie round-trip flip
	// flag); client_cookies_redirects_test.go builds on it.
	cookiesParts
	clientConfig     clientConfigSnapshot
	expectedFilePath string
	responses        []*rc.Response
	execErr          error
	validationErr    error
	cursor           int
	// baseDir is the single per-test directory every given* file-writing method uses.
	// (testing.T.TempDir creates a fresh subdirectory on every call, so external
	// files and the .http file would otherwise land in different directories.)
	baseDir string
}

// newParts returns the given, when and then entry points of the DSL.
func newParts(t *testing.T) (given, when, then *parts) {
	t.Helper()

	p := &parts{
		T:       t,
		require: require.New(t),
		assert:  assert.New(t),
		baseDir: t.TempDir(),
	}
	p.serverParts.parts = p
	p.captureParts.parts = p
	p.parseParts.parts = p
	p.fileParts.parts = p
	p.envParts.parts = p
	p.oauth2Parts.parts = p
	p.cookiesParts.parts = p
	p.trackedValues = make(map[string][]string)

	return p, p, p
}

func (p *parts) and() *parts { return p }

// multierrorCount counts the wrapped errors inside err.
func multierrorCount(err error) int {
	if err == nil {
		return 0
	}

	if merr, ok := err.(*multierror.Error); ok {
		return len(merr.Errors)
	}

	return 1
}

// aClient builds the restclient under test with the given options.
func (p *parts) aClient(opts ...rc.ClientOption) *parts {
	client, err := rc.NewClient(opts...)
	p.require.NoError(err)
	p.client = client

	return p
}

// executeFile runs every request in the .http file.
func (p *parts) executeFile() *parts {
	p.require.NotNil(p.client)
	p.responses, p.execErr = p.client.ExecuteFile(context.Background(), p.httpFilePath)

	return p
}

// validateResponses checks the actual responses against the expected file.
func (p *parts) validateResponses() *parts {
	p.validationErr = p.client.ValidateResponses(p.expectedFilePath, p.responses...)

	return p
}

func (p *parts) responseAt(i int) *parts {
	p.cursor = i

	return p
}

// current returns the response the cursor points at.
func (p *parts) current() *rc.Response {
	p.require.Greater(len(p.responses), p.cursor)

	return p.responses[p.cursor]
}

func (p *parts) responseCode(code int) *parts {
	p.require.Equal(code, p.current().StatusCode)

	return p
}

func (p *parts) responseContains(s string) *parts {
	p.require.Contains(p.current().BodyString, s)

	return p
}

func (p *parts) responseBodyIs(s string) *parts {
	p.require.Equal(s, p.current().BodyString)

	return p
}

func (p *parts) responseHeader(k, v string) *parts {
	p.require.Equal(v, p.current().Headers.Get(k))

	return p
}

func (p *parts) responseCount(n int) *parts {
	p.require.Len(p.responses, n)

	return p
}

// responseHasNoError asserts the current response has no error.
func (p *parts) responseHasNoError() *parts {
	p.require.NoError(p.current().Error)

	return p
}

// responseHasError asserts the current response carries an error mentioning every text.
func (p *parts) responseHasError(texts ...string) *parts {
	p.require.Error(p.current().Error)

	for _, text := range texts {
		p.require.Contains(p.current().Error.Error(), text)
	}

	return p
}

// noError asserts the execution succeeded and no response carries an error.
func (p *parts) noError() *parts {
	p.require.NoError(p.execErr)

	for i, resp := range p.responses {
		if resp == nil {
			p.require.Failf("nil response", "response at index %d is nil", i)

			continue
		}

		p.require.NoError(resp.Error, "response %d", i)
	}

	return p
}

// errorContains asserts the execution failed and the error mentions every text.
func (p *parts) errorContains(texts ...string) *parts {
	p.require.Error(p.execErr)

	for _, text := range texts {
		p.require.Contains(p.execErr.Error(), text)
	}

	return p
}

func (p *parts) requestCount(n int64) *parts {
	p.require.Equal(n, p.requestHits.Load())

	return p
}

// parseParts holds the parse-domain state of the fluent test DSL: the result of
// parsing the .http file under test plus the parse error asserted by the
// parsed* builders.
type parseParts struct {
	*parts
	// parsedFile holds the result of parsingFile for parsed-file assertions.
	parsedFile *rc.ParsedFile
	// parseErr holds the error returned by parsingFile.
	parseErr error
}

var _ = []any{
	parseParts{},
}

// parsingFile parses the .http file without executing any request.
func (c *parseParts) parsingFile() *parts {
	p := c.parts

	p.require.NotNil(p.client)
	p.parsedFile, p.parseErr = p.client.ParseFile(p.httpFilePath)

	return p
}

// parseSucceeded asserts the file was parsed without error.
func (c *parseParts) parseSucceeded() *parts {
	p := c.parts

	p.require.NoError(p.parseErr)

	return p
}

// parseErrorContains asserts parsing failed and the error mentions every given text.
// Called without texts it only asserts the error itself.
func (c *parseParts) parseErrorContains(texts ...string) *parts {
	p := c.parts

	p.require.Error(p.parseErr)

	for _, text := range texts {
		p.require.Contains(p.parseErr.Error(), text)
	}

	return p
}

// parsedRequestCount asserts how many requests the parse produced.
func (c *parseParts) parsedRequestCount(n int) *parts {
	p := c.parts

	p.require.NotNil(p.parsedFile)
	p.require.Len(p.parsedFile.Requests, n)

	return p
}

// parsedRequestBodyIs asserts the raw body of the parsed request at index i.
func (c *parseParts) parsedRequestBodyIs(i int, want string) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].RawBody)

	return p
}

// parsedRequestName asserts the name of the parsed request at index i.
func (c *parseParts) parsedRequestName(i int, name string) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(name, p.parsedFile.Requests[i].Name)

	return p
}

// fileParts holds the file/fixture domain state of the fluent test DSL: the path
// of the .http file under test that the file and fixture builders write and the
// executeFile / parse builders read back.
type fileParts struct {
	*parts
	// httpFilePath holds the path of the .http file under test.
	httpFilePath string
}

var _ = []any{
	fileParts{},
}

const requestFilesDir = "test/data/http_request_files"

// aTemplateFixture renders a committed test/data/<dir> template with [[ ]] delimiters.
func (f *fileParts) aTemplateFixture(dir, name string, data any) *parts {
	p := f.parts
	tmplContent, err := os.ReadFile(filepath.Join("test", "data", dir, name))
	p.require.NoError(err)

	tmpl, err := template.New(name).Delims("[[", "]]").Parse(string(tmplContent))
	p.require.NoError(err)

	path := filepath.Join(p.baseDir, name)

	file, err := os.Create(path)
	p.require.NoError(err)
	p.require.NoError(tmpl.Execute(file, data))
	p.require.NoError(file.Close())

	p.httpFilePath = path

	return p
}

// aRequestFixtureAbs points the DSL at a repo-root-relative request fixture path.
func (f *fileParts) aRequestFixtureAbs(name string) *parts {
	f.httpFilePath = name

	return f.parts
}

// aHttpFile writes the given .http content, replacing {{server}} with the test server URL.
func (f *fileParts) aHttpFile(content string) *parts {
	p := f.parts
	resolved := strings.ReplaceAll(content, "{{server}}", p.serverURL)
	path := filepath.Join(p.baseDir, "requests.http")
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	p.httpFilePath = path

	return p
}

// aHttpFileFromTemplate renders a committed request fixture template with the test server
// URL ([[.ServerURL]] delimiters) and stores the processed file path for execution.
// serverURL may be empty for parse-only fixtures; the template still renders (URL stays relative).
func (f *fileParts) aHttpFileFromTemplate(templateName string) *parts {
	p := f.parts
	return p.aHttpFileFromTemplateWithData(templateName, struct{ ServerURL string }{ServerURL: p.serverURL})
}

// aRequestFixture points the DSL at a committed request fixture file without templating.
func (f *fileParts) aRequestFixture(name string) *parts {
	p := f.parts
	p.httpFilePath = filepath.Join(requestFilesDir, name)

	return p
}

// envParts holds the env-domain state of the fluent test DSL: dotenv files,
// named dotenv files and environment-backed client builds carry no extra
// per-test state, so only the fluent-chain back pointer lives here.
type envParts struct {
	*parts
}

var _ = []any{
	envParts{},
}

// aDotEnvFile writes content to <baseDir>/.env. The parser resolves .env relative to the
// so this works for aTempDir-based .http files. No-op safety: writing empty content is allowed.
func (e *envParts) aDotEnvFile(content string) *parts {
	p := e.parts
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env"), []byte(content), 0644))
	return p
}

// aDotEnvFileRemoved removes <baseDir>/.env if present (os.Remove, ignore not-exists) so a
// previous scenario's .env cannot leak into the next subtest.
func (e *envParts) aDotEnvFileRemoved() *parts {
	p := e.parts
	_ = os.Remove(filepath.Join(p.baseDir, ".env"))
	return p
}

// withEnv sets an environment variable for the duration of the test.
func (e *envParts) withEnv(k, v string) *parts {
	p := e.parts
	p.Setenv(k, v)

	return p
}

// serverParts holds the server domain state of the fluent test DSL: the httptest
// servers started by the server builders, the address of the most recent one,
// its scheme/host/port variables and the request counter shared with the DSL.
type serverParts struct {
	*parts
	// servers holds every httptest server started by the server builders.
	servers []*httptest.Server
	// serverURL holds the address of the most recently started test server.
	serverURL string
	// share the name requestCount in Go.
	requestHits atomic.Int64
	// activeServerVars holds the scheme/host/port variables of the running test server.
	activeServerVars map[string]any
}

var _ = []any{
	serverParts{},
}

// aHttpServer counts requests; bodies are read eagerly because net/http closes them on return.
func (s *serverParts) aHttpServer(h http.HandlerFunc) *parts {
	p := s.parts
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		p.capturedRequests = append(p.capturedRequests, r)
		p.capturedBodies = append(p.capturedBodies, string(body))
		p.capturedRequestTimes = append(p.capturedRequestTimes, time.Now())
		p.requestHits.Add(1)
		h(w, r)
	}))
	p.servers = append(p.servers, srv)
	p.serverURL = srv.URL
	p.Cleanup(srv.Close)

	return p
}

// cannedRoute is one aCannedServer route; validJSON also requires a json.Valid body.
type cannedRoute struct {
	method    string
	code      int
	body      string
	validJSON bool
}

// serveCannedRoute writes the cannedRoute response for r: 405 on method mismatch,
// 400 on validJSON with an unparseable body. The caller handled unknown paths.
func serveCannedRoute(w http.ResponseWriter, r *http.Request, route cannedRoute) {
	if r.Method != route.method {
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}
	if route.validJSON {
		body, _ := io.ReadAll(r.Body)
		if !json.Valid(body) {
			w.WriteHeader(http.StatusBadRequest)

			return
		}
	}
	w.WriteHeader(route.code)
	_, _ = fmt.Fprint(w, route.body)
}

// aCannedServer serves a fixed routing table of canned responses: exact path lookup,
// 405 on method mismatch, 404 for unknown paths. Replaces multi-path handler vars.
func (s *serverParts) aCannedServer(routes map[string]cannedRoute) *parts {
	p := s.parts
	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		route, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}
		serveCannedRoute(w, r, route)
	})
}

// anEchoServer echoes the request body back verbatim with 200 OK — for tests that only
// need a round-tripped body (e.g. external-file encoding round trips).
func (s *serverParts) anEchoServer() *parts {
	p := s.parts
	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

// anExpectedResponseFixture points the DSL at a committed expected-response fixture file.
func (p *parts) anExpectedResponseFixture(name string) *parts {
	p.expectedFilePath = filepath.Join(responseFilesDir, name)

	return p
}

// aResponseWith appends a hand-constructed rc.Response to p.responses.
func (p *parts) aResponseWith(statusCode int, statusText, body string, headers http.Header) *parts {
	resp := &rc.Response{
		StatusCode: statusCode,
		Status:     statusText,
		Headers:    headers,
	}
	if body != "" {
		resp.BodyString = body
		resp.Body = []byte(body)
	}
	p.responses = append(p.responses, resp)
	return p
}

// validationFails asserts the response validation failed with count errors mentioning every text.
func (p *parts) validationFails(count int, texts ...string) *parts {
	p.require.Error(p.validationErr)

	p.require.Equal(count, multierrorCount(p.validationErr))

	for _, text := range texts {
		p.require.Contains(p.validationErr.Error(), text)
	}

	return p
}

// validationSucceeds asserts the response validation passed.
func (p *parts) validationSucceeds() *parts {
	p.require.NoError(p.validationErr)

	return p
}
