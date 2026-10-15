package restclient_test

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	rc "github.com/bmcszk/go-restclient"
)

// aFixtureCopy copies a committed fixture into baseDir, applying text replacements.
func (f *fileState) aFixtureCopy(requestPath, destName string, replacements map[string]string) *parts {
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

// firstTrackedValueIsFloatInRange asserts the first value of the active bucket parses as
// a float within [lowest, highest].
func (c *captureState) firstTrackedValueIsFloatInRange(lowest, highest float64) *parts {
	p := c.parts
	values := p.activeBucket()
	p.require.NotEmpty(values)

	number, err := strconv.ParseFloat(values[0], 64)
	p.require.NoError(err, "value %q should be a float", values[0])
	p.assert.GreaterOrEqual(number, lowest)
	p.assert.LessOrEqual(number, highest)

	return p
}

func (c *captureState) allTrackedValuesMatchRegexp(pattern string) *parts {
	p := c.parts
	re := regexp.MustCompile(pattern)
	for _, value := range p.activeBucket() {
		p.assert.True(re.MatchString(value), "value %q should match %q", value, pattern)
	}

	return p
}

func (c *captureState) capturedRequestPathMatchesRegexp(index int, pattern string) *parts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedRequests[index].URL.Path)

	return p
}

// serverReceivedJSONBody asserts the request body at the given server-hit index equals the
// expected JSON regardless of key order or whitespace.
func (c *captureState) serverReceivedJSONBody(index int, expectedJSON string) *parts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)

	body, err := io.ReadAll(p.capturedRequests[index].Body)
	p.require.NoError(err)
	p.assert.JSONEq(expectedJSON, string(body))

	return p
}

func (c *captureState) requestHeadersMatchCaptured(keys ...string) *parts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedRequests), 0)

	for _, key := range keys {
		p.assert.Equal(p.capturedRequests[0].Header.Get(key), current.Request.Headers.Get(key), "header %q", key)
	}

	return p
}

func (c *captureState) requestPathMatchesCapturedPath() *parts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedRequests), 0)
	p.assert.Equal(p.capturedRequests[0].URL.Path, current.Request.URL.Path)

	return p
}

// requestRawBodyMatchesCapturedBody asserts the request RawBody equals the first captured body.
func (c *captureState) requestRawBodyMatchesCapturedBody() *parts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedBodies), 0)
	p.assert.Equal(p.capturedBodies[0], current.Request.RawBody)

	return p
}

// serverReceivedHostIs asserts p.capturedRequests[index].Host equals host.
func (c *captureState) serverReceivedHostIs(index int, host string) *parts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(host, p.capturedRequests[index].Host)

	return p
}

// aClientWithEnvironment builds the client with WithEnvironment unless env is empty.
func (e *envState) aClientWithEnvironment(env string) *parts {
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
func (f *fileState) aHttpFileFromTemplateWithData(templateName string, data any) *parts {
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
func (e *envState) anEnvJsonFile(fileName, templatePath, serverURL string) *parts {
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
