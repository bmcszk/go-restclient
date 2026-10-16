package restclient_test

import (
	rc "github.com/bmcszk/go-restclient"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type envParts struct {
	// dotenvFiles lists dotenv files the env builders created (for assertions/cleanup).
	dotenvFiles []string
}

// newEnvParts returns the given/when/then fixture for env-domain builders.
func newEnvParts(t *testing.T) (given, when, then *testFixture) {
	t.Helper()
	return newParts(t)
}

// aDotEnvFile writes content to <baseDir>/.env. The parser resolves .env relative to the
// so this works for aTempDir-based .http files. No-op safety: writing empty content is allowed.
func (p *testFixture) aDotEnvFile(content string) *testFixture {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env"), []byte(content), 0644))
	p.dotenvFiles = append(p.dotenvFiles, filepath.Join(p.baseDir, ".env"))
	return p
}

// aDotEnvFileRemoved removes <baseDir>/.env if present (os.Remove, ignore not-exists) so a
// previous scenario's .env cannot leak into the next subtest.
func (p *testFixture) aDotEnvFileRemoved() *testFixture {
	_ = os.Remove(filepath.Join(p.baseDir, ".env"))
	return p
}

// withEnv sets an environment variable for the duration of the test.
func (p *testFixture) withEnv(k, v string) *testFixture {
	p.Setenv(k, v)

	return p
}

// aClientWithEnvironment builds the client with WithEnvironment unless env is empty.
func (p *testFixture) aClientWithEnvironment(env string) *testFixture {
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

// anEnvJsonFile writes a committed env-json template with {{SERVER_URL}} resolved.
func (p *testFixture) anEnvJsonFile(fileName, templatePath, serverURL string) *testFixture {
	content, err := os.ReadFile(templatePath)
	p.require.NoError(err)
	resolved := strings.ReplaceAll(string(content), "{{SERVER_URL}}", serverURL)
	path := filepath.Join(p.baseDir, fileName)
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0600))
	return p
}

// aDotEnvNamedFile writes content to <baseDir>/.env.<name> for --env-style scenarios.
func (p *testFixture) aDotEnvNamedFile(name, content string) *testFixture {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env."+name), []byte(content), 0644))
	return p
}

// aClientWithEnvName builds the client with WithEnvName (named dotenv file).
func (p *testFixture) aClientWithEnvName(name string) *testFixture {
	p.aClient(rc.WithEnvName(name))
	return p
}

// aJsonEchoServer accepts POST with Content-Type application/json, parses the body as JSON,
