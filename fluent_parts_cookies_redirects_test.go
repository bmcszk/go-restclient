package restclient_test

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"

	rc "github.com/bmcszk/go-restclient"
)

// Cookie / redirect / .rest extension DSL methods, grouped because they are
// driven by the cookies_redirects_test.go and external_file_test.go scenarios.

// aFreshCookieCheck resets the cookie check flag before a scenario runs.
func (p *parts) aFreshCookieCheck() *parts {
	p.cookieCheck = false

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

// aJsonEchoServer accepts POST with Content-Type application/json, parses the body as JSON,
// echoes it back wrapped in {"json": ...}, and responds 200 OK with application/json.
func (p *parts) aJsonEchoServer() *parts {
	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		_ = json.NewDecoder(r.Body).Decode(&data)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"json": data})
	})
}

// aRestExtensionServer serves GET with a 200 OK and a fixed JSON body.
func (p *parts) aRestExtensionServer() *parts {
	return p.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok from .rest"}`))
	})
}
