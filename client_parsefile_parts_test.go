package restclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	rc "github.com/bmcszk/go-restclient"
	"golang.org/x/text/transform"
)

// anExternalFile writes the request file under test verbatim to <baseDir>/name.
func (f *fileParts) anExternalFile(name, content string) *fileParts {
	f.writeTempFile(name, content)
	f.httpFilePath = filepath.Join(f.baseDir, name)

	return f
}

// aMissingFile points httpFilePath at a path that does not exist, for file-not-found tests.
func (f *fileParts) aMissingFile(path string) *fileParts {
	p := f.parts
	p.httpFilePath = path

	return f
}

// writeTempFile writes content to a file named name inside the shared per-test base dir.
func (f *fileParts) writeTempFile(name, content string) {
	p := f.parts
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, name), []byte(content), 0644))
}

// aHttpFileWithExternalFileEncoding writes the external file in the given encoding and a <@ request for it.
func (f *fileParts) aHttpFileWithExternalFileEncoding(
	encodingName, externalFileName string, rawUTF8Content []byte, encoder transform.Transformer,
) *fileParts {
	p := f.parts
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
func (f *fileParts) aHttpFileWithExternalFile(externalFileName, externalFileContent string) *fileParts {
	p := f.parts
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_with_vars.http")
}

// aHttpFileWithExternalFileStatic writes a short request body that points at the named
// external file via < (no @: NO variable substitution).
func (f *fileParts) aHttpFileWithExternalFileStatic(externalFileName, externalFileContent string) *fileParts {
	p := f.parts
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_static.http")
}

const externalFilesFixtureDir = "test/data/external_files"

// aHttpFileWithExternalFileFixture loads the committed external-file fixture via <@.
func (f *fileParts) aHttpFileWithExternalFileFixture(externalFileName string) *fileParts {
	p := f.parts
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFile(externalFileName, string(content))
}

// aHttpFileWithExternalFileStaticFixture is the no-substitution < counterpart.
func (f *fileParts) aHttpFileWithExternalFileStaticFixture(externalFileName string) *fileParts {
	p := f.parts
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFileStatic(externalFileName, string(content))
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
// echoes it back wrapped in {"json": ...}, and responds 200 OK with application/json.
func (s *cookiesParts) aJsonEchoServer() *cookiesParts {
	p := s.parts

	p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		_ = json.NewDecoder(r.Body).Decode(&data)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"json": data})
	})
	return s
}

// aRestExtensionServer serves GET with a 200 OK and a fixed JSON body.
func (s *cookiesParts) aRestExtensionServer() *cookiesParts {
	p := s.parts

	p.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok from .rest"}`))
	})
	return s
}

// parsedRequestMethod asserts the HTTP method of the parsed request at index i.
func (c *parseParts) parsedRequestMethod(i int, method string) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(method, p.parsedFile.Requests[i].Method)

	return c
}

// parsedRequestRawURLIs asserts the raw (pre-substitution) URL of the parsed request
// at index i.
func (c *parseParts) parsedRequestRawURLIs(i int, rawURL string) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(rawURL, p.parsedFile.Requests[i].RawURLString)

	return c
}

// executingRequestAt parses the .http file and executes only the request at the given index.
func (c *parseParts) executingRequestAt(index int) *parseParts {
	p := c.parts

	p.require.NotNil(p.client)

	parsed, err := p.client.ParseFile(p.httpFilePath)
	p.require.NoError(err)

	resp, execErr := p.client.ExecuteRequest(context.Background(), parsed, index)
	p.responses = append(p.responses, resp)
	p.execErr = execErr

	return c
}
