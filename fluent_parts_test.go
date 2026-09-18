package restclient_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
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
	// requestFilesDir is the committed request-fixture directory relative to the repo root.
	requestFilesDir = "test/data/http_request_files"

	// responseFilesDir is the committed expected-response fixture directory relative to the repo root.
	responseFilesDir = "test/data/http_response_files"
)

// clientConfigSnapshot records the options applied via aClientWithDefaults for later assertion.
type clientConfigSnapshot struct {
	baseURL     string
	headerKey   string
	headerValue string
	httpTimeout time.Duration
}

// parts is the shared state for the fluent Given/When/Then test DSL.
// The same instance is handed out as given, when and then.
type parts struct {
	*testing.T
	require   *require.Assertions
	assert    *assert.Assertions
	client    *rc.Client
	servers   []*httptest.Server
	serverURL string
	// requestHits backs the requestCount assertion; a field and method cannot
	// share the name requestCount in Go.
	requestHits atomic.Int64
	// capturedRequests records every request served by the most recent aHttpServer.
	capturedRequests []*http.Request
	// intercepted holds the outgoing request captured by aMockTransportClient.
	intercepted *http.Request
	// cookieCheck records whether the cookie test server received its cookie back.
	cookieCheck bool
	// clientConfig records the options applied via aClientWithDefaults for later assertion.
	clientConfig clientConfigSnapshot
	// activeServerVars holds the scheme/host/port variables of the running test server.
	activeServerVars map[string]any
	httpFilePath     string
	expectedFilePath string
	programmaticVars map[string]any
	responses        []*rc.Response
	execErr          error
	validationErr    error
	cursor           int
}

// newParts returns the given, when and then entry points of the DSL.
func newParts(t *testing.T) (given, when, then *parts) {
	t.Helper()

	p := &parts{T: t, require: require.New(t), assert: assert.New(t)}

	return p, p, p
}

// and keeps the fluent chain readable.
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

// --- Given ---

// aHttpServer starts a local test server that counts every request it serves.
func (p *parts) aHttpServer(h http.HandlerFunc) *parts {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.capturedRequests = append(p.capturedRequests, r)
		p.requestHits.Add(1)
		h(w, r)
	}))
	p.servers = append(p.servers, srv)
	p.serverURL = srv.URL
	p.Cleanup(srv.Close)

	return p
}

// aHttpFile writes the given .http content, replacing {{server}} with the test server URL.
func (p *parts) aHttpFile(content string) *parts {
	resolved := strings.ReplaceAll(content, "{{server}}", p.serverURL)
	path := filepath.Join(p.TempDir(), "requests.http")
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	p.httpFilePath = path

	return p
}

// aHttpFileFromTemplate renders a committed request fixture template with the test server
// URL ([[.ServerURL]] delimiters) and stores the processed file path for execution.
func (p *parts) aHttpFileFromTemplate(templateName string) *parts {
	p.require.NotEmpty(p.serverURL)

	return p.aHttpFileFromTemplateWithData(templateName, struct{ ServerURL string }{ServerURL: p.serverURL})
}

// aHttpFileFromTemplateWithData renders a committed request fixture template with the given
// template data ([[ ]] delimiters) and stores the processed file path for execution.
func (p *parts) aHttpFileFromTemplateWithData(templateName string, data any) *parts {
	tmplContent, err := os.ReadFile(filepath.Join(requestFilesDir, templateName))
	p.require.NoError(err)

	tmpl, err := template.New(templateName).Delims("[[", "]]").Parse(string(tmplContent))
	p.require.NoError(err)

	path := filepath.Join(p.TempDir(), templateName)

	file, err := os.Create(path)
	p.require.NoError(err)
	p.require.NoError(tmpl.Execute(file, data))
	p.require.NoError(file.Close())

	p.httpFilePath = path

	return p
}

// aRequestFixture points the DSL at a committed request fixture file without templating.
func (p *parts) aRequestFixture(name string) *parts {
	p.httpFilePath = filepath.Join(requestFilesDir, name)

	return p
}

// anExpectedResponseFixture points the DSL at a committed expected-response fixture file.
func (p *parts) anExpectedResponseFixture(name string) *parts {
	p.expectedFilePath = filepath.Join(responseFilesDir, name)

	return p
}

// aClient builds the restclient under test with the given options.
func (p *parts) aClient(opts ...rc.ClientOption) *parts {
	client, err := rc.NewClient(opts...)
	p.require.NoError(err)
	p.client = client

	return p
}

// withProgrammaticVars sets programmatic variables on the client.
func (p *parts) withProgrammaticVars(vars map[string]any) *parts {
	p.require.NotNil(p.client)
	p.client.SetProgrammaticVars(vars)
	p.programmaticVars = vars

	return p
}

// withServerAddressVars rebuilds the client with scheme/host/port variables from the
// running test server. The map is kept in parts.activeServerVars so later client
// rebuilds (cookie jar, no-redirect) can reuse the same variables.
func (p *parts) withServerAddressVars() *parts {
	parsed, err := url.Parse(p.serverURL)
	p.require.NoError(err)

	p.activeServerVars = map[string]any{
		"scheme": parsed.Scheme,
		"host":   parsed.Hostname(),
		"port":   parsed.Port(),
	}

	return p.aClient(rc.WithVars(p.activeServerVars))
}

// withEnv sets an environment variable for the duration of the test.
func (p *parts) withEnv(k, v string) *parts {
	p.Setenv(k, v)

	return p
}

// aClientWithDefaults builds the client under test from the parts-recorded config options:
// a 15s-timeout HTTP client, a base URL and one default header. The recorded values are
// stored in parts.clientConfig for later assertion.
func (p *parts) aClientWithDefaults() *parts {
	p.clientConfig = clientConfigSnapshot{
		baseURL:     "https://api.example.com",
		headerKey:   "X-Default",
		headerValue: "DefaultValue",
		httpTimeout: 15 * time.Second,
	}

	return p.aClient(
		rc.WithHTTPClient(&http.Client{Timeout: p.clientConfig.httpTimeout}),
		rc.WithBaseURL(p.clientConfig.baseURL),
		rc.WithDefaultHeader(p.clientConfig.headerKey, p.clientConfig.headerValue),
	)
}

// aMockTransportClient builds the client under test with a mock round tripper that
// records the outgoing request into parts.intercepted and answers 200 with a fixed body.
func (p *parts) aMockTransportClient() *parts {
	transport := &mockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			p.intercepted = req.Clone(req.Context())

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("mocked response")),
				Header:     make(http.Header),
			}, nil
		},
	}

	return p.aClient(rc.WithHTTPClient(&http.Client{Transport: transport}))
}

// aCookieRedirectFixture renders the named committed cookie/redirect fixture into a temp
// file. The fixture's {{scheme}}/{{host}}/{{port}} placeholders are resolved at request
// time from client variables, so the text/template pass needs no data.
func (p *parts) aCookieRedirectFixture(name string) *parts {
	tmplContent, err := os.ReadFile(filepath.Join("test", "data", "cookies_redirects", name))
	p.require.NoError(err)

	tmpl, err := template.New(name).Delims("[[", "]]").Parse(string(tmplContent))
	p.require.NoError(err)

	path := filepath.Join(p.TempDir(), name)

	file, err := os.Create(path)
	p.require.NoError(err)
	p.require.NoError(tmpl.Execute(file, nil))
	p.require.NoError(file.Close())

	p.httpFilePath = path

	return p
}

// anUploadsFixtureCopy copies the committed multipart fixture into a temp file with the
// [[.ServerURL]] token replaced by the running test server URL and the < file references
// rewritten from ./test/data/request_body/... to repo-root-relative paths that resolve
// from the working directory.
func (p *parts) anUploadsFixtureCopy() *parts {
	p.require.NotEmpty(p.serverURL)

	content, err := os.ReadFile(filepath.Join("test", "data", "http_request_files", "multipart_file_uploads.http"))
	p.require.NoError(err)

	resolved := strings.ReplaceAll(string(content), "< ./test/data/request_body/", "< test/data/request_body/")
	resolved = strings.ReplaceAll(resolved, "[[.ServerURL]]", p.serverURL)
	path := filepath.Join(p.TempDir(), "multipart_file_uploads.http")

	p.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	p.httpFilePath = path

	return p
}

// aCookieTestServer starts the cookie test server: /set-cookie sets test-cookie=test-value,
// /check-cookie flips parts.cookieCheck when it receives the cookie back.
func (p *parts) aCookieTestServer() *parts {
	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set-cookie":
			http.SetCookie(w, &http.Cookie{Name: "test-cookie", Value: "test-value"})
			w.WriteHeader(http.StatusOK)
		case "/check-cookie":
			if cookie, err := r.Cookie("test-cookie"); err == nil && cookie.Value == "test-value" {
				p.cookieCheck = true
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// aRedirectTestServer starts the redirect test server: /redirect issues a 302 to /target,
// /target answers 200 with body "Target page".
func (p *parts) aRedirectTestServer() *parts {
	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/target", http.StatusFound)
		case "/target":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Target page"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// aCookieJarClient rebuilds the client with a fresh cookie jar.
func (p *parts) aCookieJarClient() *parts {
	jar, err := cookiejar.New(nil)
	p.require.NoError(err)

	return p.aClient(rc.WithVars(p.activeServerVars), rc.WithHTTPClient(&http.Client{Jar: jar}))
}

// aNoRedirectClient rebuilds the client with an HTTP client that refuses to follow redirects.
func (p *parts) aNoRedirectClient() *parts {
	noRedirectHTTPClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return p.aClient(rc.WithVars(p.activeServerVars), rc.WithHTTPClient(noRedirectHTTPClient))
}

// aFreshCookieCheck resets the cookie check flag before a scenario runs.
func (p *parts) aFreshCookieCheck() *parts {
	p.cookieCheck = false

	return p
}

// expectedResponseFile writes the given .hresp content as the expected responses file.
func (p *parts) expectedResponseFile(content string) *parts {
	path := filepath.Join(p.TempDir(), "expected.hresp")
	p.require.NoError(os.WriteFile(path, []byte(content), 0644))
	p.expectedFilePath = path

	return p
}

// --- When ---

// executeFile runs every request in the .http file.
func (p *parts) executeFile() *parts {
	p.require.NotNil(p.client)
	p.responses, p.execErr = p.client.ExecuteFile(context.Background(), p.httpFilePath)

	return p
}

// sendingRequest executes the named request from the .http file.
func (p *parts) sendingRequest(name string) *parts {
	p.require.NotNil(p.client)

	parsed, err := p.client.ParseFile(p.httpFilePath)
	p.require.NoError(err)

	index := -1
	for i, req := range parsed.Requests {
		if req.Name == name {
			index = i

			break
		}
	}

	if index < 0 {
		p.Fatalf("request %q not found in %s", name, p.httpFilePath)
	}

	resp, execErr := p.client.ExecuteRequest(context.Background(), parsed, index)
	p.responses = append(p.responses, resp)
	p.execErr = execErr

	return p
}

// validateResponses checks the actual responses against the expected file.
func (p *parts) validateResponses() *parts {
	p.validationErr = p.client.ValidateResponses(p.expectedFilePath, p.responses...)

	return p
}

// --- Then ---

// responseAt selects the response the following assertions operate on.
func (p *parts) responseAt(i int) *parts {
	p.cursor = i

	return p
}

// current returns the response the cursor points at.
func (p *parts) current() *rc.Response {
	p.require.Greater(len(p.responses), p.cursor)

	return p.responses[p.cursor]
}

// responseCode asserts the status code of the selected response.
func (p *parts) responseCode(code int) *parts {
	p.require.Equal(code, p.current().StatusCode)

	return p
}

// responseContains asserts the body of the selected response contains s.
func (p *parts) responseContains(s string) *parts {
	p.require.Contains(p.current().BodyString, s)

	return p
}

// responseNotContains asserts the body of the selected response does not contain s.
func (p *parts) responseNotContains(s string) *parts {
	p.require.NotContains(p.current().BodyString, s)

	return p
}

// responseBodyIs asserts the exact body of the selected response.
func (p *parts) responseBodyIs(s string) *parts {
	p.require.Equal(s, p.current().BodyString)

	return p
}

// responseHeader asserts the value of a header of the selected response.
func (p *parts) responseHeader(k, v string) *parts {
	p.require.Equal(v, p.current().Headers.Get(k))

	return p
}

// executionSucceeded asserts the file execution reported no error.
func (p *parts) executionSucceeded() *parts {
	p.require.NoError(p.execErr)

	return p
}

// responseCount asserts how many responses the execution produced.
func (p *parts) responseCount(n int) *parts {
	p.require.Len(p.responses, n)

	return p
}

// responseHasNoError asserts the selected response carries no execution error.
func (p *parts) responseHasNoError() *parts {
	p.require.NoError(p.current().Error)

	return p
}

// responseHasError asserts the selected response carries an execution error mentioning
// every given text.
func (p *parts) responseHasError(texts ...string) *parts {
	p.require.Error(p.current().Error)

	for _, text := range texts {
		p.require.Contains(p.current().Error.Error(), text)
	}

	return p
}

// responseHeaderValues asserts the exact values of a multi-value response header.
func (p *parts) responseHeaderValues(key string, values ...string) *parts {
	p.require.Equal(values, p.current().Headers[key])

	return p
}

// responseHeaderEmpty asserts a header is absent from the selected response.
func (p *parts) responseHeaderEmpty(key string) *parts {
	p.require.Empty(p.current().Headers.Get(key))

	return p
}

// clientBaseURLIs asserts the client's base URL equals the recorded configured value.
func (p *parts) clientBaseURLIs(_ string) *parts {
	p.require.Equal(p.clientConfig.baseURL, p.client.BaseURL)

	return p
}

// clientBaseURLIsEmpty asserts the client has no base URL configured.
func (p *parts) clientBaseURLIsEmpty() *parts {
	p.assert.Empty(p.client.BaseURL)

	return p
}

// clientDefaultHeaderIs asserts a client default header equals the recorded configured value.
func (p *parts) clientDefaultHeaderIs(key, _ string) *parts {
	p.require.Equal(p.clientConfig.headerValue, p.client.DefaultHeaders.Get(key))

	return p
}

// clientExists asserts a client was built successfully and is not nil.
func (p *parts) clientExists() *parts {
	p.require.NotNil(p.client)

	return p
}

// clientDefaultHeadersEmpty asserts the client has initialized but empty default headers.
func (p *parts) clientDefaultHeadersEmpty() *parts {
	p.require.NotNil(p.client.DefaultHeaders)
	p.assert.Empty(p.client.DefaultHeaders)

	return p
}

// clientRequestInterceptorCaptures asserts the outgoing request was intercepted and its
// method and URL match the given values.
func (p *parts) clientRequestInterceptorCaptures(method, wantURL string) *parts {
	p.require.NotNil(p.intercepted)
	p.assert.Equal(method, p.intercepted.Method)
	p.assert.Equal(wantURL, p.intercepted.URL.String())

	return p
}

// clientSentNoHeaders asserts the intercepted outgoing request carried no headers.
func (p *parts) clientSentNoHeaders() *parts {
	p.require.NotNil(p.intercepted)
	p.assert.Empty(p.intercepted.Header)

	return p
}

// cookieWasStored asserts the cookie server received its cookie back.
func (p *parts) cookieWasStored() *parts {
	p.assert.True(p.cookieCheck, "cookie check assertion failed")

	return p
}

// cookieWasNotStored asserts the cookie server did not receive the cookie back.
func (p *parts) cookieWasNotStored() *parts {
	p.assert.False(p.cookieCheck, "cookie check assertion failed")

	return p
}

// serverReceivedMethodAndPath asserts the request at the given server-hit index used the
// expected HTTP method and path.
func (p *parts) serverReceivedMethodAndPath(index int, method, path string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(method, p.capturedRequests[index].Method)
	p.assert.Equal(path, p.capturedRequests[index].URL.Path)

	return p
}

// serverReceivedJSONBody asserts the request body at the given server-hit index equals the
// expected JSON regardless of key order or whitespace.
func (p *parts) serverReceivedJSONBody(index int, expectedJSON string) *parts {
	p.require.Greater(len(p.capturedRequests), index)

	body, err := io.ReadAll(p.capturedRequests[index].Body)
	p.require.NoError(err)
	p.assert.JSONEq(expectedJSON, string(body))

	return p
}

// serverReceivedHeaderValue asserts a request header value at the given server-hit index.
func (p *parts) serverReceivedHeaderValue(index int, key, value string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(value, p.capturedRequests[index].Header.Get(key))

	return p
}

// serverReceivedFormValue asserts a form field value at the given server-hit index.
func (p *parts) serverReceivedFormValue(index int, field, value string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(value, p.capturedRequests[index].FormValue(field))

	return p
}

// serverReceivedUploadedFile asserts a multipart file upload at the given server-hit index:
// the form file must exist with the expected filename and content type.
func (p *parts) serverReceivedUploadedFile(index int, fieldName, filename, contentType string) *parts {
	p.require.Greater(len(p.capturedRequests), index)

	p.require.NoError(p.capturedRequests[index].ParseMultipartForm(32 << 20)) // 32MB max

	file, header, err := p.capturedRequests[index].FormFile(fieldName)
	p.require.NoError(err)
	defer func() { _ = file.Close() }()

	p.assert.Equal(filename, header.Filename)
	p.assert.Equal(contentType, header.Header.Get("Content-Type"))

	return p
}

// requestRawURLContains asserts every given fragment appears in the raw URL string of the
// selected response's originating request.
func (p *parts) requestRawURLContains(fragments ...string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.RawURLString, fragment)
	}

	return p
}

// responsesValidateAgainstFixture validates the executed responses against the committed
// expected-response fixture named name and asserts the validation passes.
func (p *parts) responsesValidateAgainstFixture(name string) *parts {
	path := filepath.Join(responseFilesDir, name)
	p.assert.NoError(p.client.ValidateResponses(path, p.responses...))

	return p
}

// capturedRequestCount asserts how many distinct requests reached the test server.
func (p *parts) capturedRequestCount(n int) *parts {
	p.require.Equal(n, len(p.capturedRequests))

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

// errorCount asserts the number of errors returned by the execution.
func (p *parts) errorCount(n int) *parts {
	p.require.Equal(n, multierrorCount(p.execErr))

	return p
}

// requestCount asserts how many requests hit the test server.
func (p *parts) requestCount(n int64) *parts {
	p.require.Equal(n, p.requestHits.Load())

	return p
}

// validationSucceeds asserts the response validation passed.
func (p *parts) validationSucceeds() *parts {
	p.require.NoError(p.validationErr)

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

// mockRoundTripper adapts a function into an http.RoundTripper for the DSL.
type mockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

// RoundTrip delegates to the configured function.
func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.RoundTripFunc != nil {
		return m.RoundTripFunc(req)
	}

	return nil, errors.New("RoundTripFunc not set")
}

// dslSymbols keeps every DSL entry point referenced so the unused linter stays
// quiet in this definitions-only file until tests adopt the DSL.
var _ = []any{
	newParts,
	(*parts).and,
	(*parts).aHttpServer,
	(*parts).aHttpFile,
	(*parts).aHttpFileFromTemplate,
	(*parts).aHttpFileFromTemplateWithData,
	(*parts).aRequestFixture,
	(*parts).anExpectedResponseFixture,
	(*parts).aClient,
	(*parts).aClientWithDefaults,
	(*parts).aMockTransportClient,
	(*parts).aCookieRedirectFixture,
	(*parts).aCookieTestServer,
	(*parts).aRedirectTestServer,
	(*parts).aCookieJarClient,
	(*parts).aNoRedirectClient,
	(*parts).aFreshCookieCheck,
	(*parts).withProgrammaticVars,
	(*parts).withEnv,
	(*parts).withServerAddressVars,
	(*parts).expectedResponseFile,
	(*parts).executeFile,
	(*parts).sendingRequest,
	(*parts).validateResponses,
	(*parts).responseAt,
	(*parts).current,
	(*parts).responseCode,
	(*parts).responseContains,
	(*parts).responseNotContains,
	(*parts).responseBodyIs,
	(*parts).responseHeader,
	(*parts).responseHeaderValues,
	(*parts).responseHeaderEmpty,
	(*parts).clientBaseURLIs,
	(*parts).clientBaseURLIsEmpty,
	(*parts).clientDefaultHeaderIs,
	(*parts).clientExists,
	(*parts).clientDefaultHeadersEmpty,
	(*parts).clientRequestInterceptorCaptures,
	(*parts).clientSentNoHeaders,
	(*parts).cookieWasStored,
	(*parts).cookieWasNotStored,
	(*parts).responseHasNoError,
	(*parts).responseHasError,
	(*parts).responseCount,
	(*parts).capturedRequestCount,
	(*parts).executionSucceeded,
	(*parts).noError,
	(*parts).errorContains,
	(*parts).errorCount,
	(*parts).requestCount,
	(*parts).validationSucceeds,
	(*parts).validationFails,
	(*parts).anUploadsFixtureCopy,
	(*parts).serverReceivedMethodAndPath,
	(*parts).serverReceivedJSONBody,
	(*parts).serverReceivedHeaderValue,
	(*parts).serverReceivedFormValue,
	(*parts).serverReceivedUploadedFile,
	(*parts).requestRawURLContains,
	(*parts).responsesValidateAgainstFixture,
}
