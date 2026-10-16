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
func (f *fileState) anExternalFile(name, content string) *parts {
	f.writeTempFile(name, content)
	f.httpFilePath = filepath.Join(f.baseDir, name)

	return f.parts
}

// aMissingFile points httpFilePath at a path that does not exist, for file-not-found tests.
func (f *fileState) aMissingFile(path string) *parts {
	p := f.parts
	p.httpFilePath = path

	return p
}

// writeTempFile writes content to a file named name inside the shared per-test base dir.
func (f *fileState) writeTempFile(name, content string) {
	p := f.parts
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, name), []byte(content), 0644))
}

// aHttpFileWithExternalFileEncoding writes the external file in the given encoding and a <@ request for it.
func (f *fileState) aHttpFileWithExternalFileEncoding(
	encodingName, externalFileName string, rawUTF8Content []byte, encoder transform.Transformer,
) *parts {
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
func (f *fileState) aHttpFileWithExternalFile(externalFileName, externalFileContent string) *parts {
	p := f.parts
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_with_vars.http")
}

// aHttpFileWithExternalFileStatic writes a short request body that points at the named
// external file via < (no @: NO variable substitution).
func (f *fileState) aHttpFileWithExternalFileStatic(externalFileName, externalFileContent string) *parts {
	p := f.parts
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_static.http")
}

const externalFilesFixtureDir = "test/data/external_files"

// aHttpFileWithExternalFileFixture loads the committed external-file fixture via <@.
func (f *fileState) aHttpFileWithExternalFileFixture(externalFileName string) *parts {
	p := f.parts
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFile(externalFileName, string(content))
}

// aHttpFileWithExternalFileStaticFixture is the no-substitution < counterpart.
func (f *fileState) aHttpFileWithExternalFileStaticFixture(externalFileName string) *parts {
	p := f.parts
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFileStatic(externalFileName, string(content))
}

// aDotEnvNamedFile writes content to <baseDir>/.env.<name> for --env-style scenarios.
func (e *envState) aDotEnvNamedFile(name, content string) *parts {
	p := e.parts
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env."+name), []byte(content), 0644))
	return p
}

// aClientWithEnvName builds the client with WithEnvName (named dotenv file).
func (e *envState) aClientWithEnvName(name string) *parts {
	p := e.parts
	return p.aClient(rc.WithEnvName(name))
}

// aJsonEchoServer accepts POST with Content-Type application/json, parses the body as JSON,
// echoes it back wrapped in {"json": ...}, and responds 200 OK with application/json.
func (s *cookiesState) aJsonEchoServer() *parts {
	p := s.parts

	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		_ = json.NewDecoder(r.Body).Decode(&data)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"json": data})
	})
}

// aRestExtensionServer serves GET with a 200 OK and a fixed JSON body.
func (s *cookiesState) aRestExtensionServer() *parts {
	p := s.parts

	return p.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok from .rest"}`))
	})
}

// parsedRequestMethod asserts the HTTP method of the parsed request at index i.
func (c *parseState) parsedRequestMethod(i int, method string) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(method, p.parsedFile.Requests[i].Method)

	return p
}

// parsedRequestRawURLIs asserts the raw (pre-substitution) URL of the parsed request
// at index i.
func (c *parseState) parsedRequestRawURLIs(i int, rawURL string) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(rawURL, p.parsedFile.Requests[i].RawURLString)

	return p
}

// executingRequestAt parses the .http file and executes only the request at the given index.
func (c *parseState) executingRequestAt(index int) *parts {
	p := c.parts

	p.require.NotNil(p.client)

	parsed, err := p.client.ParseFile(p.httpFilePath)
	p.require.NoError(err)

	resp, execErr := p.client.ExecuteRequest(context.Background(), parsed, index)
	p.responses = append(p.responses, resp)
	p.execErr = execErr

	return p
}
