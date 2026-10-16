package restclient_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/text/transform"
)

const externalFilesFixtureDir = "test/data/external_files"

const requestFilesDir = "test/data/http_request_files"

// fileParts holds the file/fixture domain state of the fluent test DSL: the path
// of the .http file under test that the file and fixture builders write and the
// executeFile / parse builders read back.
type fileParts struct {
	*parts
	// httpFilePath holds the path of the .http file under test.
	httpFilePath string
}

var _ = []any{
	fileParts{},
}

// aTemplateFixture renders a committed test/data/<dir> template with [[ ]] delimiters.
func (f *fileParts) aTemplateFixture(dir, name string, data any) *fileParts {
	tmplContent, err := os.ReadFile(filepath.Join("test", "data", dir, name))
	f.require.NoError(err)

	tmpl, err := template.New(name).Delims("[[", "]]").Parse(string(tmplContent))
	f.require.NoError(err)

	path := filepath.Join(f.baseDir, name)

	file, err := os.Create(path)
	f.require.NoError(err)
	f.require.NoError(tmpl.Execute(file, data))
	f.require.NoError(file.Close())

	f.httpFilePath = path

	return f
}

// aRequestFixtureAbs points the DSL at a repo-root-relative request fixture path.
func (f *fileParts) aRequestFixtureAbs(name string) *fileParts {
	f.httpFilePath = name

	return f
}

// aHttpFile writes the given .http content, replacing {{server}} with the test server URL.
func (f *fileParts) aHttpFile(content string) *fileParts {
	resolved := strings.ReplaceAll(content, "{{server}}", f.serverURL)
	path := filepath.Join(f.baseDir, "requests.http")
	f.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	f.httpFilePath = path

	return f
}

// aHttpFileFromTemplate renders a committed request fixture template with the test server
// URL ([[.ServerURL]] delimiters) and stores the processed file path for execution.
// serverURL may be empty for parse-only fixtures; the template still renders (URL stays relative).
func (f *fileParts) aHttpFileFromTemplate(templateName string) *fileParts {
	return f.aHttpFileFromTemplateWithData(templateName, struct{ ServerURL string }{ServerURL: f.serverURL})
}

// aRequestFixture points the DSL at a committed request fixture file without templating.
func (f *fileParts) aRequestFixture(name string) *fileParts {
	f.httpFilePath = filepath.Join(requestFilesDir, name)

	return f
}

// aFormattedRequestFixture renders a committed %s-placeholder fixture with fmt.Sprintf.
func (f *fileParts) aFormattedRequestFixture(name string, substitutions ...string) *fileParts {
	baseContent, err := os.ReadFile(filepath.Join(requestFilesDir, name))
	if err != nil {
		baseContent, err = os.ReadFile(filepath.Join("test", "data", "execute_file_ignore_empty_blocks", name))
		f.require.NoError(err)
	}

	args := make([]any, len(substitutions))
	for i, s := range substitutions {
		args[i] = s
	}

	return f.aHttpFile(fmt.Sprintf(string(baseContent), args...))
}

// anUploadsFixtureCopy copies the multipart fixture with server URL and < paths resolved.
func (f *fileParts) anUploadsFixtureCopy() *fileParts {
	f.require.NotEmpty(f.serverURL)

	content, err := os.ReadFile(filepath.Join("test", "data", "http_request_files", "multipart_file_uploads.http"))
	f.require.NoError(err)

	resolved := strings.ReplaceAll(string(content), "< ./test/data/request_body/", "< test/data/request_body/")
	resolved = strings.ReplaceAll(resolved, "[[.ServerURL]]", f.serverURL)
	path := filepath.Join(f.baseDir, "multipart_file_uploads.http")

	f.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	f.httpFilePath = path

	return f
}

// aFixtureCopy copies a committed fixture into baseDir, applying text replacements.
func (f *fileParts) aFixtureCopy(requestPath, destName string, replacements map[string]string) *fileParts {
	content, err := os.ReadFile(requestPath)
	f.require.NoError(err)
	result := string(content)
	for old, newVal := range replacements {
		result = strings.ReplaceAll(result, old, newVal)
	}
	path := filepath.Join(f.baseDir, destName)
	f.require.NoError(os.WriteFile(path, []byte(result), 0644))
	f.httpFilePath = path
	return f
}

// aHttpFileFromTemplateWithData renders a committed request fixture template with the given
// template data ([[ ]] delimiters) and stores the processed file path for execution.
func (f *fileParts) aHttpFileFromTemplateWithData(templateName string, data any) *fileParts {
	tmplContent, err := os.ReadFile(filepath.Join(requestFilesDir, templateName))
	f.require.NoError(err)

	tmpl, err := template.New(templateName).Delims("[[", "]]").Parse(string(tmplContent))
	f.require.NoError(err)

	path := filepath.Join(f.baseDir, templateName)

	file, err := os.Create(path)
	f.require.NoError(err)
	f.require.NoError(tmpl.Execute(file, data))
	f.require.NoError(file.Close())

	f.httpFilePath = path

	return f
}

// anExternalFile writes the request file under test verbatim to <baseDir>/name.
func (f *fileParts) anExternalFile(name, content string) *fileParts {
	f.writeTempFile(name, content)
	f.httpFilePath = filepath.Join(f.baseDir, name)

	return f
}

// aMissingFile points httpFilePath at a path that does not exist, for file-not-found tests.
func (f *fileParts) aMissingFile(path string) *fileParts {
	f.httpFilePath = path

	return f
}

// writeTempFile writes content to a file named name inside the shared per-test base dir.
func (f *fileParts) writeTempFile(name, content string) {
	f.require.NoError(os.WriteFile(filepath.Join(f.baseDir, name), []byte(content), 0644))
}

// aHttpFileWithExternalFileEncoding writes the external file in the given encoding and a <@ request for it.
func (f *fileParts) aHttpFileWithExternalFileEncoding(
	encodingName, externalFileName string, rawUTF8Content []byte, encoder transform.Transformer,
) *fileParts {
	var encoded []byte
	if encoder != nil {
		var err error
		encoded, _, err = transform.Bytes(encoder, rawUTF8Content)
		f.require.NoError(err)
	} else {
		encoded = rawUTF8Content
	}

	f.require.NoError(os.WriteFile(filepath.Join(f.baseDir, externalFileName), encoded, 0644))

	return f.aHttpFileFromTemplateWithData("external_file_encoding.http",
		struct {
			ServerURL    string
			EncodingName string
			ExternalName string
		}{f.serverURL, encodingName, externalFileName})
}

// aHttpFileWithExternalFile writes a short request body that points at the named external
// file via <@, and stores the given JSON content in the external file.
func (f *fileParts) aHttpFileWithExternalFile(externalFileName, externalFileContent string) *fileParts {
	f.require.NoError(os.WriteFile(filepath.Join(f.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return f.aHttpFileFromTemplate("external_file_with_vars.http")
}

// aHttpFileWithExternalFileStatic writes a short request body that points at the named
// external file via < (no @: NO variable substitution).
func (f *fileParts) aHttpFileWithExternalFileStatic(externalFileName, externalFileContent string) *fileParts {
	f.require.NoError(os.WriteFile(filepath.Join(f.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return f.aHttpFileFromTemplate("external_file_static.http")
}

// aHttpFileWithExternalFileFixture loads the committed external-file fixture via <@.
func (f *fileParts) aHttpFileWithExternalFileFixture(externalFileName string) *fileParts {
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	f.require.NoError(err)

	return f.aHttpFileWithExternalFile(externalFileName, string(content))
}

// aHttpFileWithExternalFileStaticFixture is the no-substitution < counterpart.
func (f *fileParts) aHttpFileWithExternalFileStaticFixture(externalFileName string) *fileParts {
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	f.require.NoError(err)

	return f.aHttpFileWithExternalFileStatic(externalFileName, string(content))
}
