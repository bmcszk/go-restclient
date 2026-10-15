package restclient_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/text/transform"
)

// fileState holds the file/fixture domain state of the fluent test DSL: the path
// of the .http file under test that the file and fixture builders write and the
// executeFile / parse builders read back.
type fileState struct {
	// root points back at the enclosing parts so the file builders can return
	// the fluent chain; parts reads it by field promotion as p.root as well.
	root *parts
	// httpFilePath holds the path of the .http file under test.
	httpFilePath string
}

var _ = []any{
	fileState{},
}

const requestFilesDir = "test/data/http_request_files"

// anExternalFile writes the request file under test verbatim to <baseDir>/name.
func (f *fileState) anExternalFile(name, content string) *parts {
	p := f.root
	p.writeTempFile(name, content)
	p.httpFilePath = filepath.Join(p.baseDir, name)

	return p
}

// aFormattedRequestFixture renders a committed %s-placeholder fixture with fmt.Sprintf.
func (f *fileState) aFormattedRequestFixture(name string, substitutions ...string) *parts {
	p := f.root
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

// aTemplateFixture renders a committed test/data/<dir> template with [[ ]] delimiters.
func (f *fileState) aTemplateFixture(dir, name string, data any) *parts {
	p := f.root
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

// aMissingFile points httpFilePath at a path that does not exist, for file-not-found tests.
func (f *fileState) aMissingFile(path string) *parts {
	p := f.root
	p.httpFilePath = path

	return p
}

// aFixtureCopy copies a committed fixture into baseDir, applying text replacements.
func (f *fileState) aFixtureCopy(requestPath, destName string, replacements map[string]string) *parts {
	p := f.root
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

// aRequestFixtureAbs points the DSL at a repo-root-relative request fixture path.
func (f *fileState) aRequestFixtureAbs(name string) *parts {
	p := f.root
	p.httpFilePath = name

	return p
}

// writeTempFile writes content to a file named name inside the shared per-test base dir.
func (f *fileState) writeTempFile(name, content string) {
	p := f.root
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, name), []byte(content), 0644))
}

// aHttpFileWithExternalFileEncoding writes the external file in the given encoding and a <@ request for it.
func (f *fileState) aHttpFileWithExternalFileEncoding(
	encodingName, externalFileName string, rawUTF8Content []byte, encoder transform.Transformer,
) *parts {
	p := f.root
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
func (f *fileState) aHttpFileWithExternalFile(externalFileName, externalFileContent string) *parts {
	p := f.root
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_with_vars.http")
}

// aHttpFileWithExternalFileStatic writes a short request body that points at the named
// external file via < (no @: NO variable substitution).
func (f *fileState) aHttpFileWithExternalFileStatic(externalFileName, externalFileContent string) *parts {
	p := f.root
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_static.http")
}

const externalFilesFixtureDir = "test/data/external_files"

// aHttpFileWithExternalFileFixture loads the committed external-file fixture via <@.
func (f *fileState) aHttpFileWithExternalFileFixture(externalFileName string) *parts {
	p := f.root
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFile(externalFileName, string(content))
}

// aHttpFileWithExternalFileStaticFixture is the no-substitution < counterpart.
func (f *fileState) aHttpFileWithExternalFileStaticFixture(externalFileName string) *parts {
	p := f.root
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFileStatic(externalFileName, string(content))
}

// aHttpFile writes the given .http content, replacing {{server}} with the test server URL.
func (f *fileState) aHttpFile(content string) *parts {
	p := f.root
	resolved := strings.ReplaceAll(content, "{{server}}", p.serverURL)
	path := filepath.Join(p.baseDir, "requests.http")
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	p.httpFilePath = path

	return p
}

// aHttpFileFromTemplate renders a committed request fixture template with the test server
// URL ([[.ServerURL]] delimiters) and stores the processed file path for execution.
// serverURL may be empty for parse-only fixtures; the template still renders (URL stays relative).
func (f *fileState) aHttpFileFromTemplate(templateName string) *parts {
	p := f.root
	return p.aHttpFileFromTemplateWithData(templateName, struct{ ServerURL string }{ServerURL: p.serverURL})
}

// aHttpFileFromTemplateWithData renders a committed request fixture template with the given
// template data ([[ ]] delimiters) and stores the processed file path for execution.
func (f *fileState) aHttpFileFromTemplateWithData(templateName string, data any) *parts {
	p := f.root
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

// aRequestFixture points the DSL at a committed request fixture file without templating.
func (f *fileState) aRequestFixture(name string) *parts {
	p := f.root
	p.httpFilePath = filepath.Join(requestFilesDir, name)

	return p
}

// anUploadsFixtureCopy copies the multipart fixture with server URL and < paths resolved.
func (f *fileState) anUploadsFixtureCopy() *parts {
	p := f.root
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
