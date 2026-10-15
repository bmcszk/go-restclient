package restclient_test

import (
	"os"
	"path/filepath"
	"strings"

	rc "github.com/bmcszk/go-restclient"
)

// envState holds the env-domain state of the fluent test DSL: dotenv files,
// named dotenv files and environment-backed client builds carry no extra
// per-test state, so only the fluent-chain back pointer lives here.
type envState struct {
	// root points back at the enclosing parts so the env builders can return
	// the fluent chain; parts reads it by field promotion as p.root as well.
	root *parts
}

var _ = []any{
	envState{},
}

// aDotEnvFile writes content to <baseDir>/.env. The parser resolves .env relative to the
// so this works for aTempDir-based .http files. No-op safety: writing empty content is allowed.
func (e *envState) aDotEnvFile(content string) *parts {
	p := e.root
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env"), []byte(content), 0644))
	return p
}

// aDotEnvFileRemoved removes <baseDir>/.env if present (os.Remove, ignore not-exists) so a
// previous scenario's .env cannot leak into the next subtest.
func (e *envState) aDotEnvFileRemoved() *parts {
	p := e.root
	_ = os.Remove(filepath.Join(p.baseDir, ".env"))
	return p
}

// aDotEnvNamedFile writes content to <baseDir>/.env.<name> for --env-style scenarios.
func (e *envState) aDotEnvNamedFile(name, content string) *parts {
	p := e.root
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env."+name), []byte(content), 0644))
	return p
}

// aClientWithEnvName builds the client with WithEnvName (named dotenv file).
func (e *envState) aClientWithEnvName(name string) *parts {
	p := e.root
	return p.aClient(rc.WithEnvName(name))
}

// anEnvJsonFile writes a committed env-json template with {{SERVER_URL}} resolved.
func (e *envState) anEnvJsonFile(fileName, templatePath, serverURL string) *parts {
	p := e.root
	content, err := os.ReadFile(templatePath)
	p.require.NoError(err)
	resolved := strings.ReplaceAll(string(content), "{{SERVER_URL}}", serverURL)
	path := filepath.Join(p.baseDir, fileName)
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0600))
	return p
}

// aClientWithEnvironment builds the client with WithEnvironment unless env is empty.
func (e *envState) aClientWithEnvironment(env string) *parts {
	p := e.root
	var client *rc.Client
	var err error
	if env != "" {
		client, err = rc.NewClient(rc.WithEnvironment(env))
	} else {
		client, err = rc.NewClient()
	}
	p.require.NoError(err)
	p.client = client
	return p
}

// withEnv sets an environment variable for the duration of the test.
func (e *envState) withEnv(k, v string) *parts {
	p := e.root
	p.Setenv(k, v)

	return p
}
