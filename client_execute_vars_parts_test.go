package restclient_test

import (
	"os"
	"path/filepath"
	"strings"

	rc "github.com/bmcszk/go-restclient"
)

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

// responsesValidateAgainst validates responses against an explicit .hresp path.
func (p *parts) responsesValidateAgainst(expectedPath string) *parts {
	p.require.NotNil(p.client)
	p.assert.NoError(p.client.ValidateResponses(expectedPath, p.responses...))

	return p
}
