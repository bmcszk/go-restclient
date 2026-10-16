package restclient_test

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"

	rc "github.com/bmcszk/go-restclient"
)

// aFixtureCopy copies a committed fixture into baseDir, applying text replacements.
func (f *fileParts) aFixtureCopy(requestPath, destName string, replacements map[string]string) *parts {
	p := f.parts
	content, err := os.ReadFile(requestPath)
	p.require.NoError(err)
	result := string(content)
	for old, newVal := range replacements {
		result = strings.ReplaceAll(result, old, newVal)
	}
	path := filepath.Join(p.baseDir, destName)
	p.require.NoError(os.WriteFile(path, []byte(result), 0644))
	p.httpFilePath = path
	return p
}

// aClientWithEnvironment builds the client with WithEnvironment unless env is empty.
func (e *envParts) aClientWithEnvironment(env string) *parts {
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
	return p
}

// aHttpFileFromTemplateWithData renders a committed request fixture template with the given
// template data ([[ ]] delimiters) and stores the processed file path for execution.
func (f *fileParts) aHttpFileFromTemplateWithData(templateName string, data any) *parts {
	p := f.parts
	tmplContent, err := os.ReadFile(filepath.Join(requestFilesDir, templateName))
	p.require.NoError(err)

	tmpl, err := template.New(templateName).Delims("[[", "]]").Parse(string(tmplContent))
	p.require.NoError(err)

	path := filepath.Join(p.baseDir, templateName)

	file, err := os.Create(path)
	p.require.NoError(err)
	p.require.NoError(tmpl.Execute(file, data))
	p.require.NoError(file.Close())

	p.httpFilePath = path

	return p
}

// anEnvJsonFile writes a committed env-json template with {{SERVER_URL}} resolved.
func (e *envParts) anEnvJsonFile(fileName, templatePath, serverURL string) *parts {
	p := e.parts
	content, err := os.ReadFile(templatePath)
	p.require.NoError(err)
	resolved := strings.ReplaceAll(string(content), "{{SERVER_URL}}", serverURL)
	path := filepath.Join(p.baseDir, fileName)
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0600))
	return p
}

// responsesValidateAgainst validates responses against an explicit .hresp path.
func (p *parts) responsesValidateAgainst(expectedPath string) *parts {
	p.require.NotNil(p.client)
	p.assert.NoError(p.client.ValidateResponses(expectedPath, p.responses...))

	return p
}
