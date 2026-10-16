package restclient_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"golang.org/x/text/transform"
)

const externalFilesFixtureDir = "test/data/external_files"

const requestFilesDir = "test/data/http_request_files"

// fileParts holds the file/fixture domain state of the fluent test DSL: the path
// of the .http file under test that the file and fixture builders write and the
// executeFile / parse builders read back.
type fileParts struct {
	// httpFilePath holds the path of the .http file under test.
	httpFilePath string
}

// newFileParts returns the given/when/then fixture for the file/fixture-domain fluent DSL.
func newFileParts(t *testing.T) (given, when, then *testFixture) {
	t.Helper()

	return newParts(t)
}

// aTemplateFixture renders a committed test/data/<dir> template with [[ ]] delimiters.
func (p *testFixture) aTemplateFixture(dir, name string, data any) *testFixture {
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

// aRequestFixtureAbs points the DSL at a repo-root-relative request fixture path.
func (p *testFixture) aRequestFixtureAbs(name string) *testFixture {
	p.httpFilePath = name

	return p
}

// aHttpFile writes the given .http content, replacing {{server}} with the test server URL.
func (p *testFixture) aHttpFile(content string) *testFixture {
	resolved := strings.ReplaceAll(content, "{{server}}", p.serverURL)
	path := filepath.Join(p.baseDir, "requests.http")
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	p.httpFilePath = path

	return p
}

// aHttpFileFromTemplate renders a committed request fixture template with the test server
// URL ([[.ServerURL]] delimiters) and stores the processed file path for execution.
// serverURL may be empty for parse-only fixtures; the template still renders (URL stays relative).
func (p *testFixture) aHttpFileFromTemplate(templateName string) *testFixture {
	return p.aHttpFileFromTemplateWithData(templateName, struct{ ServerURL string }{ServerURL: p.serverURL})
}

// aRequestFixture points the DSL at a committed request fixture file without templating.
func (p *testFixture) aRequestFixture(name string) *testFixture {
	p.httpFilePath = filepath.Join(requestFilesDir, name)

	return p
}

// aFormattedRequestFixture renders a committed %s-placeholder fixture with fmt.Sprintf.
func (p *testFixture) aFormattedRequestFixture(name string, substitutions ...string) *testFixture {
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

// anUploadsFixtureCopy copies the multipart fixture with server URL and < paths resolved.
func (p *testFixture) anUploadsFixtureCopy() *testFixture {
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
func (p *testFixture) aFixtureCopy(requestPath, destName string, replacements map[string]string) *testFixture {
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
func (p *testFixture) aHttpFileFromTemplateWithData(templateName string, data any) *testFixture {
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

// anExternalFile writes the request file under test verbatim to <baseDir>/name.
func (p *testFixture) anExternalFile(name, content string) *testFixture {
	p.writeTempFile(name, content)
	p.httpFilePath = filepath.Join(p.baseDir, name)

	return p
}

// aMissingFile points httpFilePath at a path that does not exist, for file-not-found tests.
func (p *testFixture) aMissingFile(path string) *testFixture {
	p.httpFilePath = path

	return p
}

// writeTempFile writes content to a file named name inside the shared per-test base dir.
func (p *testFixture) writeTempFile(name, content string) {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, name), []byte(content), 0644))
}

// aHttpFileWithExternalFileEncoding writes the external file in the given encoding and a <@ request for it.
func (p *testFixture) aHttpFileWithExternalFileEncoding(
	encodingName, externalFileName string, rawUTF8Content []byte, encoder transform.Transformer,
) *testFixture {
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
func (p *testFixture) aHttpFileWithExternalFile(externalFileName, externalFileContent string) *testFixture {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_with_vars.http")
}

// aHttpFileWithExternalFileStatic writes a short request body that points at the named
// external file via < (no @: NO variable substitution).
func (p *testFixture) aHttpFileWithExternalFileStatic(externalFileName, externalFileContent string) *testFixture {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_static.http")
}

// aHttpFileWithExternalFileFixture loads the committed external-file fixture via <@.
func (p *testFixture) aHttpFileWithExternalFileFixture(externalFileName string) *testFixture {
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFile(externalFileName, string(content))
}

// aHttpFileWithExternalFileStaticFixture is the no-substitution < counterpart.
func (p *testFixture) aHttpFileWithExternalFileStaticFixture(externalFileName string) *testFixture {
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFileStatic(externalFileName, string(content))
}

// executeFile runs every request in the .http file.
func (p *testFixture) executeFile() *testFixture {
	p.require.NotNil(p.client)
	p.responses, p.execErr = p.client.ExecuteFile(context.Background(), p.httpFilePath)

	return p
}

// expectedResponseFile writes the given .hresp content as the expected responses file.
func (p *testFixture) expectedResponseFile(content string) *testFixture {
	path := filepath.Join(p.baseDir, "expected.hresp")
	p.require.NoError(os.WriteFile(path, []byte(content), 0644))
	p.expectedFilePath = path

	return p
}
