package main_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cliParts is the shared state for the CLI fluent Given/When/Then test DSL.
type cliParts struct {
	*testing.T
	require      *require.Assertions
	assert       *assert.Assertions
	binaryPath   string
	workDir      string
	httpFile     string
	expectedFile string
	servers      []*httptest.Server
	serverURL    string
	// capturedPaths holds the request paths the mock server received.
	capturedPaths []string
	// capturedHeaders stores request header values keyed by "path:key".
	capturedHeaders map[string]string
	output          string
	exitCode        int
}

// newParts returns the given, when and then entry points of the CLI DSL.
func newParts(t *testing.T) (given, when, then *cliParts) {
	t.Helper()

	p := &cliParts{
		T:               t,
		require:         require.New(t),
		assert:          assert.New(t),
		capturedHeaders: make(map[string]string),
	}

	return p, p, p
}

func (p *cliParts) and() *cliParts { return p }

// givenABuiltBinary compiles the restclient binary under test.
func (p *cliParts) givenABuiltBinary() *cliParts {
	binary := filepath.Join(p.TempDir(), "restclient")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = filepath.Join(p.moduleRoot(), "cmd", "restclient")
	out, err := cmd.CombinedOutput()
	p.require.NoError(err, "build failed: %s", string(out))
	p.binaryPath = binary

	return p
}

// moduleRoot walks up from the CLI package dir to the go.mod directory.
func (p *cliParts) moduleRoot() string {
	dir, err := os.Getwd()
	p.require.NoError(err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			p.Fatal("could not find module root")
		}
		dir = parent
	}
}

// givenAMockServer starts a test HTTP server recording paths and headers.
func (p *cliParts) givenAMockServer(h http.HandlerFunc) *cliParts {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.capturedPaths = append(p.capturedPaths, r.URL.Path)
		for key, values := range r.Header {
			p.capturedHeaders[r.URL.Path+":"+key] = values[0]
		}
		h(w, r)
	}))
	p.servers = append(p.servers, srv)
	p.serverURL = srv.URL
	p.Cleanup(srv.Close)

	return p
}

// givenAnHttpFile writes the .http file content into the working directory.
func (p *cliParts) givenAnHttpFile(content string) *cliParts {
	p.httpFile = p.writeFile("test.http", content)

	return p
}

// givenAnExpectedFile writes the expected-response file into the working dir.
func (p *cliParts) givenAnExpectedFile(content string) *cliParts {
	p.expectedFile = p.writeFile("expected.hresp", content)

	return p
}

// givenADotenvFile writes a dotenv file with the given name and content.
func (p *cliParts) givenADotenvFile(name, content string) *cliParts {
	p.writeFile(name, content)

	return p
}

// whenRunningWithArgs runs the binary and captures output and exit code.
func (p *cliParts) whenRunningWithArgs(args ...string) *cliParts {
	cmd := exec.Command(p.binaryPath, args...)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			p.Fatalf("failed to run binary: %v", err)
		}
	}
	p.output = string(out)
	p.exitCode = exitCode

	return p
}

// whenRunningAll runs the binary with -f file --all.
func (p *cliParts) whenRunningAll() *cliParts {
	return p.whenRunningWithArgs("-f", p.httpFile, "--all")
}

// whenRunningAllWithDefine runs -f file --all with one -D per define.
func (p *cliParts) whenRunningAllWithDefine(defines ...string) *cliParts {
	args := []string{"-f", p.httpFile, "--all"}
	for _, d := range defines {
		args = append(args, "-D", d)
	}

	return p.whenRunningWithArgs(args...)
}

// whenRunningAllWithEnv runs -f file --all --env name.
func (p *cliParts) whenRunningAllWithEnv(envName string) *cliParts {
	return p.whenRunningWithArgs("-f", p.httpFile, "--all", "--env", envName)
}

// whenRunningAllWithFailOnError runs -f file --all --fail-on-error.
func (p *cliParts) whenRunningAllWithFailOnError() *cliParts {
	return p.whenRunningWithArgs("-f", p.httpFile, "--all", "--fail-on-error")
}

// whenRunningAllWithExpected runs -f file --all -e expected.hresp.
func (p *cliParts) whenRunningAllWithExpected() *cliParts {
	return p.whenRunningWithArgs("-f", p.httpFile, "--all", "-e", p.expectedFile)
}

// whenRunningNamed runs the binary with -f file -n name.
func (p *cliParts) whenRunningNamed(name string) *cliParts {
	return p.whenRunningWithArgs("-f", p.httpFile, "-n", name)
}

// whenRunningNamedWithExpected runs -f file -n name -e expected.hresp.
func (p *cliParts) whenRunningNamedWithExpected(name string) *cliParts {
	return p.whenRunningWithArgs("-f", p.httpFile, "-n", name, "-e", p.expectedFile)
}

// whenRunningNamedWithAfter runs -f file -n name -A after.
func (p *cliParts) whenRunningNamedWithAfter(name, after string) *cliParts {
	return p.whenRunningWithArgs("-f", p.httpFile, "-n", name, "-A", after)
}

// whenRunningByIndex runs -f file -i index.
func (p *cliParts) whenRunningByIndex(index string) *cliParts {
	return p.whenRunningWithArgs("-f", p.httpFile, "-i", index)
}

// whenRunningByIndexLong runs -f file --index=<index>.
func (p *cliParts) whenRunningByIndexLong(index string) *cliParts {
	return p.whenRunningWithArgs("-f", p.httpFile, "--index="+index)
}

// whenRunningList runs -f file --list.
func (p *cliParts) whenRunningList() *cliParts {
	return p.whenRunningWithArgs("-f", p.httpFile, "--list")
}

// whenRunningListWithoutFile runs the binary with only --list.
func (p *cliParts) whenRunningListWithoutFile() *cliParts {
	return p.whenRunningWithArgs("--list")
}

// whenRunningNoArgs runs the binary without any arguments.
func (p *cliParts) whenRunningNoArgs() *cliParts {
	return p.whenRunningWithArgs()
}

// whenRunningVersionFlag runs the binary with the given version flag.
func (p *cliParts) whenRunningVersionFlag(flag string) *cliParts {
	return p.whenRunningWithArgs(flag)
}

// thenExitCodeIs asserts the exit code.
func (p *cliParts) thenExitCodeIs(code int) *cliParts {
	p.require.Equal(code, p.exitCode)

	return p
}

// thenOutputContains asserts the combined output mentions s.
func (p *cliParts) thenOutputContains(s string) *cliParts {
	p.require.Contains(p.output, s)

	return p
}

// thenOutputNotContains asserts the combined output does not mention s.
func (p *cliParts) thenOutputNotContains(s string) *cliParts {
	p.require.NotContains(p.output, s)

	return p
}

// thenOutputTrimmedIsNotEmpty asserts trimmed output is non-empty.
func (p *cliParts) thenOutputTrimmedIsNotEmpty() *cliParts {
	p.assert.NotEmpty(strings.TrimSpace(p.output))

	return p
}

// thenServerGotPaths asserts the server saw exactly the given request paths.
func (p *cliParts) thenServerGotPaths(want ...string) *cliParts {
	p.require.Equal(want, p.capturedPaths)

	return p
}

// thenServerGotHeaderOnPath asserts the header value for the recorded path.
func (p *cliParts) thenServerGotHeaderOnPath(path, key, want string) *cliParts {
	p.require.Equal(want, p.capturedHeaders[path+":"+key])

	return p
}

// writeFile writes content into the working directory and returns the path.
func (p *cliParts) writeFile(name, content string) string {
	p.ensureWorkDir()
	path := filepath.Join(p.workDir, name)
	p.require.NoError(os.WriteFile(path, []byte(content), 0o644))

	return path
}

// ensureWorkDir lazily creates the shared working directory.
func (p *cliParts) ensureWorkDir() {
	if p.workDir == "" {
		p.workDir = p.TempDir()
	}
}

// bearerChecker returns a handler requiring Authorization: Bearer <token>.
func bearerChecker(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	}
}

// apiKeyEnvChecker returns a handler requiring X-API-Key and X-Env headers.
func apiKeyEnvChecker() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "key123" || r.Header.Get("X-Env") != "staging" {
			http.Error(w, "bad headers", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	}
}

// keyEchoHandler responds with the request's X-Key header value.
func keyEchoHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprint(w, r.Header.Get("X-Key"))
}
