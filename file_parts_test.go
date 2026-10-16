package restclient_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/transform"
)

const externalFilesFixtureDir = "test/data/external_files"

const requestFilesDir = "test/data/http_request_files"

type fileParts struct {
	parts
}

func newFileParts(t *testing.T) (given, when, then *fileParts) {
	t.Helper()

	f := &fileParts{parts: parts{
		T:       t,
		require: require.New(t),
		assert:  assert.New(t),
		baseDir: t.TempDir(),
	}}

	return f, f, f
}

func (p *parts) aTemplateFixture(dir, name string, data any) *parts {
	tmplContent, err := os.ReadFile(filepath.Join("test", "data", dir, name))
	p.require.NoError(err)

	tmpl, err := template.New(name).Delims("[[", "]]").Parse(string(tmplContent))
	p.require.NoError(err)

	path := filepath.Join(p.baseDir, name)

	file, err := os.Create(path)
	p.require.NoError(err)
	p.require.NoError(tmpl.Execute(file, data))
	p.require.NoError(file.Close())

	p.httpFilePath = path

	return p
}

func (p *parts) aRequestFixtureAbs(name string) *parts {
	p.httpFilePath = name

	return p
}

func (p *parts) aHttpFile(content string) *parts {
	resolved := strings.ReplaceAll(content, "{{server}}", p.serverURL)
	path := filepath.Join(p.baseDir, "requests.http")
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	p.httpFilePath = path

	return p
}

// aHttpFileFromTemplate renders a committed request fixture template with the test server
// URL ([[.ServerURL]] delimiters) and stores the processed file path for execution.
// serverURL may be empty for parse-only fixtures; the template still renders (URL stays relative).
func (p *parts) aHttpFileFromTemplate(templateName string) *parts {
	return p.aHttpFileFromTemplateWithData(templateName, struct{ ServerURL string }{ServerURL: p.serverURL})
}

func (p *parts) aRequestFixture(name string) *parts {
	p.httpFilePath = filepath.Join(requestFilesDir, name)

	return p
}

func (p *parts) aFormattedRequestFixture(name string, substitutions ...string) *parts {
	baseContent, err := os.ReadFile(filepath.Join(requestFilesDir, name))
	if err != nil {
		baseContent, err = os.ReadFile(filepath.Join("test", "data", "execute_file_ignore_empty_blocks", name))
		p.require.NoError(err)
	}

	args := make([]any, len(substitutions))
	for i, s := range substitutions {
		args[i] = s
	}

	return p.aHttpFile(fmt.Sprintf(string(baseContent), args...))
}

func (p *parts) anUploadsFixtureCopy() *parts {
	p.require.NotEmpty(p.serverURL)

	content, err := os.ReadFile(filepath.Join("test", "data", "http_request_files", "multipart_file_uploads.http"))
	p.require.NoError(err)

	resolved := strings.ReplaceAll(string(content), "< ./test/data/request_body/", "< test/data/request_body/")
	resolved = strings.ReplaceAll(resolved, "[[.ServerURL]]", p.serverURL)
	path := filepath.Join(p.baseDir, "multipart_file_uploads.http")

	p.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	p.httpFilePath = path

	return p
}

// aFixtureCopy copies a committed fixture into baseDir, applying text replacements.
func (p *parts) aFixtureCopy(requestPath, destName string, replacements map[string]string) *parts {
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

// aHttpFileFromTemplateWithData renders a committed request fixture template with the given
// template data ([[ ]] delimiters) and stores the processed file path for execution.
func (p *parts) aHttpFileFromTemplateWithData(templateName string, data any) *parts {
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

func (p *parts) anExternalFile(name, content string) *parts {
	p.writeTempFile(name, content)
	p.httpFilePath = filepath.Join(p.baseDir, name)

	return p
}

func (p *parts) aMissingFile(path string) *parts {
	p.httpFilePath = path

	return p
}

func (p *parts) writeTempFile(name, content string) {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, name), []byte(content), 0644))
}

func (p *parts) aHttpFileWithExternalFileEncoding(
	encodingName, externalFileName string, rawUTF8Content []byte, encoder transform.Transformer,
) *parts {
	var encoded []byte
	if encoder != nil {
		var err error
		encoded, _, err = transform.Bytes(encoder, rawUTF8Content)
		p.require.NoError(err)
	} else {
		encoded = rawUTF8Content
	}

	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), encoded, 0644))

	return p.aHttpFileFromTemplateWithData("external_file_encoding.http",
		struct {
			ServerURL    string
			EncodingName string
			ExternalName string
		}{p.serverURL, encodingName, externalFileName})
}

// aHttpFileWithExternalFile writes a short request body that points at the named external
// file via <@, and stores the given JSON content in the external file.
func (p *parts) aHttpFileWithExternalFile(externalFileName, externalFileContent string) *parts {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_with_vars.http")
}

// aHttpFileWithExternalFileStatic writes a short request body that points at the named
// external file via < (no @: NO variable substitution).
func (p *parts) aHttpFileWithExternalFileStatic(externalFileName, externalFileContent string) *parts {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_static.http")
}

// aHttpFileWithExternalFileFixture loads the committed external-file fixture via <@.
func (p *parts) aHttpFileWithExternalFileFixture(externalFileName string) *parts {
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFile(externalFileName, string(content))
}

func (p *parts) aHttpFileWithExternalFileStaticFixture(externalFileName string) *parts {
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFileStatic(externalFileName, string(content))
}

func (p *parts) executeFile() *parts {
	p.require.NotNil(p.client)
	p.responses, p.execErr = p.client.ExecuteFile(context.Background(), p.httpFilePath)

	return p
}

func (p *parts) expectedResponseFile(content string) *parts {
	path := filepath.Join(p.baseDir, "expected.hresp")
	p.require.NoError(os.WriteFile(path, []byte(content), 0644))
	p.expectedFilePath = path

	return p
}
