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

// cookiesParts holds the cookie/redirect domain state of the fluent test DSL: the
// cookie round-trip flip flag set by aCookieTestServer's /check-cookie handler and
// read by cookieWasStored / cookieWasNotStored.
type cookiesParts struct {
	*parts
	// cookieCheck flips true when the cookie server receives its test cookie back.
	cookieCheck bool
}

var _ = []any{
	cookiesParts{},
}

// aFreshCookieCheck resets the cookie check flag before a scenario runs.
func (s *cookiesParts) aFreshCookieCheck() *cookiesParts {
	p := s.parts
	p.cookieCheck = false

	return s
}

// aCookieTestServer starts the cookie test server: /set-cookie sets test-cookie=test-value,
// /check-cookie flips cookiesParts.cookieCheck when it receives the cookie back.
func (s *cookiesParts) aCookieTestServer() *cookiesParts {
	p := s.parts

	p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
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
	return s
}

// aRedirectTestServer starts the redirect test server: /redirect issues a 302 to /target,
// /target answers 200 with body "Target page".
func (s *cookiesParts) aRedirectTestServer() *cookiesParts {
	p := s.parts

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
	return s
}

// aCookieJarClient rebuilds the client with a fresh cookie jar.
func (s *cookiesParts) aCookieJarClient() *cookiesParts {
	p := s.parts
	jar, err := cookiejar.New(nil)
	p.require.NoError(err)

	p.aClient(rc.WithVars(p.activeServerVars), rc.WithHTTPClient(&http.Client{Jar: jar}))

	return s
}

// aNoRedirectClient rebuilds the client with an HTTP client that refuses to follow redirects.
func (s *cookiesParts) aNoRedirectClient() *cookiesParts {
	p := s.parts
	noRedirectHTTPClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	p.aClient(rc.WithVars(p.activeServerVars), rc.WithHTTPClient(noRedirectHTTPClient))

	return s
}

// cookieWasStored asserts the cookie server received its cookie back.
func (s *cookiesParts) cookieWasStored() *cookiesParts {
	p := s.parts
	p.assert.True(s.cookieCheck, "cookie check assertion failed")

	return s
}

// cookieWasNotStored asserts the cookie server did not receive the cookie back.
func (s *cookiesParts) cookieWasNotStored() *cookiesParts {
	p := s.parts
	p.assert.False(s.cookieCheck, "cookie check assertion failed")

	return s
}

// aCookieRedirectFixture renders a committed cookie/redirect template into baseDir.
func (s *cookiesParts) aCookieRedirectFixture(name string) *cookiesParts {
	p := s.parts
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

	return s
}

// echoes it back wrapped in {"json": ...}, and responds 200 OK with application/json.
func (s *cookiesParts) aJsonEchoServer() *cookiesParts {
	p := s.parts

	p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		_ = json.NewDecoder(r.Body).Decode(&data)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"json": data})
	})
	return s
}

// aRestExtensionServer serves GET with a 200 OK and a fixed JSON body.
func (s *cookiesParts) aRestExtensionServer() *cookiesParts {
	p := s.parts

	p.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok from .rest"}`))
	})
	return s
}
