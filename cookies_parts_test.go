package restclient_test

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"text/template"

	rc "github.com/bmcszk/go-restclient"
)

// cookiesState holds the cookie/redirect domain state of the fluent test DSL: the
// cookie round-trip flip flag set by aCookieTestServer's /check-cookie handler and
// read by cookieWasStored / cookieWasNotStored.
type cookiesState struct {
	// root points back at the enclosing parts so the cookies builders can return
	// the fluent chain; parts reads it by field promotion as p.root as well.
	root *parts
	// cookieCheck flips true when the cookie server receives its test cookie back.
	cookieCheck bool
}

var _ = []any{
	cookiesState{},
}

// aFreshCookieCheck resets the cookie check flag before a scenario runs.
func (s *cookiesState) aFreshCookieCheck() *parts {
	p := s.root
	p.cookieCheck = false

	return p
}

// aCookieTestServer starts the cookie test server: /set-cookie sets test-cookie=test-value,
// /check-cookie flips cookiesState.cookieCheck when it receives the cookie back.
func (s *cookiesState) aCookieTestServer() *parts {
	p := s.root

	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set-cookie":
			http.SetCookie(w, &http.Cookie{Name: "test-cookie", Value: "test-value"})
			w.WriteHeader(http.StatusOK)
		case "/check-cookie":
			if cookie, err := r.Cookie("test-cookie"); err == nil && cookie.Value == "test-value" {
				s.cookieCheck = true
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// aRedirectTestServer starts the redirect test server: /redirect issues a 302 to /target,
// /target answers 200 with body "Target page".
func (s *cookiesState) aRedirectTestServer() *parts {
	p := s.root

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
func (s *cookiesState) aCookieJarClient() *parts {
	p := s.root
	jar, err := cookiejar.New(nil)
	p.require.NoError(err)

	return p.aClient(rc.WithVars(p.activeServerVars), rc.WithHTTPClient(&http.Client{Jar: jar}))
}

// aNoRedirectClient rebuilds the client with an HTTP client that refuses to follow redirects.
func (s *cookiesState) aNoRedirectClient() *parts {
	p := s.root
	noRedirectHTTPClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return p.aClient(rc.WithVars(p.activeServerVars), rc.WithHTTPClient(noRedirectHTTPClient))
}

// cookieWasStored asserts the cookie server received its cookie back.
func (s *cookiesState) cookieWasStored() *parts {
	p := s.root
	p.assert.True(s.cookieCheck, "cookie check assertion failed")

	return p
}

// cookieWasNotStored asserts the cookie server did not receive the cookie back.
func (s *cookiesState) cookieWasNotStored() *parts {
	p := s.root
	p.assert.False(s.cookieCheck, "cookie check assertion failed")

	return p
}

// aJsonEchoServer accepts POST with Content-Type application/json, parses the body as JSON,
// echoes it back wrapped in {"json": ...}, and responds 200 OK with application/json.
func (s *cookiesState) aJsonEchoServer() *parts {
	p := s.root

	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		_ = json.NewDecoder(r.Body).Decode(&data)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"json": data})
	})
}

// aRestExtensionServer serves GET with a 200 OK and a fixed JSON body.
func (s *cookiesState) aRestExtensionServer() *parts {
	p := s.root

	return p.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok from .rest"}`))
	})
}

// aCookieRedirectFixture renders a committed cookie/redirect template into baseDir.
func (s *cookiesState) aCookieRedirectFixture(name string) *parts {
	p := s.root
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
