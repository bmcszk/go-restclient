package restclient_test

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	rc "github.com/bmcszk/go-restclient"
)

// cookiesParts holds the cookie/redirect domain state of the fluent test DSL: the
// cookie round-trip flip flag set by aCookieTestServer's /check-cookie handler and
// read by cookieWasStored / cookieWasNotStored.
type cookiesParts struct {
	// cookieCheck flips true when the cookie server receives its test cookie back.
	cookieCheck bool
}

// newCookiesParts returns the given/when/then fixture for the cookie/redirect-domain fluent DSL.
func newCookiesParts(t *testing.T) (given, when, then *testFixture) {
	t.Helper()

	fx := newFixture(t)
	return fx, fx, fx
}

// aFreshCookieCheck resets the cookie check flag before a scenario runs.
func (p *testFixture) aFreshCookieCheck() *testFixture {
	p.cookieCheck = false

	return p
}

// aCookieTestServer starts the cookie test server: /set-cookie sets test-cookie=test-value,
// /check-cookie flips cookiesParts.cookieCheck when it receives the cookie back.
func (p *testFixture) aCookieTestServer() *testFixture {
	p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
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
	return p
}

// aRedirectTestServer starts the redirect test server: /redirect issues a 302 to /target,
// /target answers 200 with body "Target page".
func (p *testFixture) aRedirectTestServer() *testFixture {
	p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
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
	return p
}

// aCookieJarClient rebuilds the client with a fresh cookie jar.
func (p *testFixture) aCookieJarClient() *testFixture {
	jar, err := cookiejar.New(nil)
	p.require.NoError(err)

	p.aClient(rc.WithVars(p.activeServerVars), rc.WithHTTPClient(&http.Client{Jar: jar}))

	return p
}

// aNoRedirectClient rebuilds the client with an HTTP client that refuses to follow redirects.
func (p *testFixture) aNoRedirectClient() *testFixture {
	noRedirectHTTPClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	p.aClient(rc.WithVars(p.activeServerVars), rc.WithHTTPClient(noRedirectHTTPClient))

	return p
}

// cookieWasStored asserts the cookie server received its cookie back.
func (p *testFixture) cookieWasStored() *testFixture {
	p.assert.True(p.cookieCheck, "cookie check assertion failed")

	return p
}

// cookieWasNotStored asserts the cookie server did not receive the cookie back.
func (p *testFixture) cookieWasNotStored() *testFixture {
	p.assert.False(p.cookieCheck, "cookie check assertion failed")

	return p
}

// aCookieRedirectFixture renders a committed cookie/redirect template into baseDir.
func (p *testFixture) aCookieRedirectFixture(name string) *testFixture {
	tmplContent, err := os.ReadFile(filepath.Join("test", "data", "cookies_redirects", name))
	p.require.NoError(err)

	tmpl, err := template.New(name).Delims("[[", "]]").Parse(string(tmplContent))
	p.require.NoError(err)

	path := filepath.Join(p.baseDir, name)

	file, err := os.Create(path)
	p.require.NoError(err)
	p.require.NoError(tmpl.Execute(file, nil))
	p.require.NoError(file.Close())

	p.httpFilePath = path

	return p
}

// echoes it back wrapped in {"json": ...}, and responds 200 OK with application/json.
func (p *testFixture) aJsonEchoServer() *testFixture {
	p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		_ = json.NewDecoder(r.Body).Decode(&data)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"json": data})
	})
	return p
}

// aRestExtensionServer serves GET with a 200 OK and a fixed JSON body.
func (p *testFixture) aRestExtensionServer() *testFixture {
	p.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok from .rest"}`))
	})
	return p
}
