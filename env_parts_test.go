package restclient_test

import (
	rc "github.com/bmcszk/go-restclient"
	"os"
	"path/filepath"
	"strings"
)

// aDotEnvFile writes content to <baseDir>/.env. The parser resolves .env relative to the
// so this works for aTempDir-based .http files. No-op safety: writing empty content is allowed.
func (p *parts) aDotEnvFile(content string) *parts {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env"), []byte(content), 0644))
	p.dotenvFiles = append(p.dotenvFiles, filepath.Join(p.baseDir, ".env"))
	return p
}

func (p *parts) aDotEnvFileRemoved() *parts {
	_ = os.Remove(filepath.Join(p.baseDir, ".env"))
	return p
}

func (p *parts) withEnv(k, v string) *parts {
	p.Setenv(k, v)

	return p
}

func (p *parts) aClientWithEnvironment(env string) *parts {
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
func (p *parts) anEnvJsonFile(fileName, templatePath, serverURL string) *parts {
	content, err := os.ReadFile(templatePath)
	p.require.NoError(err)
	resolved := strings.ReplaceAll(string(content), "{{SERVER_URL}}", serverURL)
	path := filepath.Join(p.baseDir, fileName)
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0600))
	return p
}

func (p *parts) aDotEnvNamedFile(name, content string) *parts {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env."+name), []byte(content), 0644))
	return p
}

func (p *parts) aClientWithEnvName(name string) *parts {
	p.aClient(rc.WithEnvName(name))
	return p
}

// aJsonEchoServer accepts POST with Content-Type application/json, parses the body as JSON,
