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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cookiesParts struct {
	parts
}

func newCookiesParts(t *testing.T) (given, when, then *cookiesParts) {
	t.Helper()

	c := &cookiesParts{parts: parts{
		trackedValues: make(map[string][]string),
		T:             t,
		require:       require.New(t),
		assert:        assert.New(t),
		baseDir:       t.TempDir(),
	}}

	return c, c, c
}

func (p *parts) aFreshCookieCheck() *parts {
	p.cookieCheck = false

	return p
}

// aCookieTestServer starts the cookie test server: /set-cookie sets test-cookie=test-value,
// /check-cookie flips cookieCheck when it receives the cookie back.
func (p *parts) aCookieTestServer() *parts {
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
func (p *parts) aRedirectTestServer() *parts {
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

func (p *parts) aCookieJarClient() *parts {
	jar, err := cookiejar.New(nil)
	p.require.NoError(err)

	p.aClient(rc.WithVars(p.activeServerVars), rc.WithHTTPClient(&http.Client{Jar: jar}))

	return p
}

func (p *parts) aNoRedirectClient() *parts {
	noRedirectHTTPClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	p.aClient(rc.WithVars(p.activeServerVars), rc.WithHTTPClient(noRedirectHTTPClient))

	return p
}

func (p *parts) cookieWasStored() *parts {
	p.assert.True(p.cookieCheck, "cookie check assertion failed")

	return p
}

func (p *parts) cookieWasNotStored() *parts {
	p.assert.False(p.cookieCheck, "cookie check assertion failed")

	return p
}

func (p *parts) aCookieRedirectFixture(name string) *parts {
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
func (p *parts) aJsonEchoServer() *parts {
	p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		_ = json.NewDecoder(r.Body).Decode(&data)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"json": data})
	})
	return p
}

func (p *parts) aRestExtensionServer() *parts {
	p.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok from .rest"}`))
	})
	return p
}
