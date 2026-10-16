package restclient_test

import (
	rc "github.com/bmcszk/go-restclient"
	"os"
	"path/filepath"
	"strings"
)

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
func (e *envParts) aDotEnvFile(content string) *envParts {
	p := e.parts
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env"), []byte(content), 0644))
	return e
}

// aDotEnvFileRemoved removes <baseDir>/.env if present (os.Remove, ignore not-exists) so a
// previous scenario's .env cannot leak into the next subtest.
func (e *envParts) aDotEnvFileRemoved() *envParts {
	p := e.parts
	_ = os.Remove(filepath.Join(p.baseDir, ".env"))
	return e
}

// withEnv sets an environment variable for the duration of the test.
func (e *envParts) withEnv(k, v string) *envParts {
	p := e.parts
	p.Setenv(k, v)

	return e
}

// aClientWithEnvironment builds the client with WithEnvironment unless env is empty.
func (e *envParts) aClientWithEnvironment(env string) *envParts {
	p := e.parts
	var client *rc.Client
	var err error
	if env != "" {
		client, err = rc.NewClient(rc.WithEnvironment(env))
	} else {
		client, err = rc.NewClient()
	}
	p.require.NoError(err)
	p.client = client
	return e
}

// anEnvJsonFile writes a committed env-json template with {{SERVER_URL}} resolved.
func (e *envParts) anEnvJsonFile(fileName, templatePath, serverURL string) *envParts {
	p := e.parts
	content, err := os.ReadFile(templatePath)
	p.require.NoError(err)
	resolved := strings.ReplaceAll(string(content), "{{SERVER_URL}}", serverURL)
	path := filepath.Join(p.baseDir, fileName)
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0600))
	return e
}

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
