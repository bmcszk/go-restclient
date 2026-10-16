package restclient_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	rc "github.com/bmcszk/go-restclient"
)

// aDotEnvNamedFile writes content to <baseDir>/.env.<name> for --env-style scenarios.
func (e *envParts) aDotEnvNamedFile(name, content string) *envParts {
	p := e.parts
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env."+name), []byte(content), 0644))
	return e
}

// aClientWithEnvName builds the client with WithEnvName (named dotenv file).
func (e *envParts) aClientWithEnvName(name string) *envParts {
	p := e.parts
	p.aClient(rc.WithEnvName(name))
	return e
}

// aJsonEchoServer accepts POST with Content-Type application/json, parses the body as JSON,
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
