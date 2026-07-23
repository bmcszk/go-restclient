package main_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildBinary compiles the restclient binary and returns its path.
func buildBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "restclient")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = filepath.Join(findModuleRoot(t), "cmd", "restclient")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(out))
	return binary
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root")
		}
		dir = parent
	}
}

// writeTestFile writes content to a temp .http file.
func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// startMockServer starts a test HTTP server.
func startMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func runBinary(t *testing.T, binary string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run binary: %v", err)
		}
	}
	return string(out), exitCode
}

func TestCLI_NoArgs(t *testing.T) {
	binary := buildBinary(t)
	out, code := runBinary(t, binary)
	assert.Equal(t, 2, code)
	assert.Contains(t, out, "-f <file> is required")
}

func TestCLI_MissingFile(t *testing.T) {
	binary := buildBinary(t)
	out, code := runBinary(t, binary, "-f", "/nonexistent/file.http")
	assert.Equal(t, 1, code)
	assert.Contains(t, out, "error")
}

func TestCLI_SingleRequest(t *testing.T) {
	server := startMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "hello")
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("GET %s/health\n", server.URL))

	out, code := runBinary(t, binary, "-f", filePath)
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "200 OK")
	assert.Contains(t, out, "hello")
}

func TestCLI_SelectByName(t *testing.T) {
	var paths []string
	server := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("### get user\nGET %s/a\n###\n### get order\nGET %s/b\n",
			server.URL, server.URL))

	out, code := runBinary(t, binary, "-f", filePath, "-n", "get order")
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "200 OK")
	assert.Equal(t, []string{"/b"}, paths, "only the named request should execute")
}

func TestCLI_SelectByIndex(t *testing.T) {
	var paths []string
	server := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("GET %s/first\n###\nGET %s/second\n###\nGET %s/third\n",
			server.URL, server.URL, server.URL))

	out, code := runBinary(t, binary, "-f", filePath, "-i", "1")
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "200 OK")
	assert.Equal(t, []string{"/second"}, paths)
}

func TestCLI_NameNotFound(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		"GET http://example.com/a\n")

	out, code := runBinary(t, binary, "-f", filePath, "-n", "nonexistent")
	assert.Equal(t, 1, code)
	assert.Contains(t, out, "not found")
}

func TestCLI_IndexOutOfRange(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		"GET http://example.com/a\n")

	out, code := runBinary(t, binary, "-f", filePath, "-i", "5")
	assert.Equal(t, 1, code)
	assert.Contains(t, out, "out of range")
}

func TestCLI_NameAndIndexMutuallyExclusive(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		"GET http://example.com/a\n")

	out, code := runBinary(t, binary, "-f", filePath, "-n", "foo", "-i", "0")
	assert.Equal(t, 2, code)
	assert.Contains(t, out, "mutually exclusive")
}

func TestCLI_ExpectedPass(t *testing.T) {
	server := startMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Custom", "yes")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "body-ok")
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("GET %s/check\n", server.URL))

	expectedPath := writeTestFile(t, dir, "expected.hresp",
		"HTTP/1.1 200 OK\nX-Custom: yes\n\nbody-ok\n")

	out, code := runBinary(t, binary, "-f", filePath, "-e", expectedPath)
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "200 OK")
}

func TestCLI_ExpectedFail(t *testing.T) {
	server := startMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "actual-body")
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("GET %s/check\n", server.URL))

	expectedPath := writeTestFile(t, dir, "expected.hresp",
		"HTTP/1.1 200 OK\n\nexpected-body\n")

	out, code := runBinary(t, binary, "-f", filePath, "-e", expectedPath)
	assert.Equal(t, 1, code)
	assert.Contains(t, out, "error")
}

func TestCLI_SelectByNameWithExpected(t *testing.T) {
	server := startMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, "created")
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("### skip this\nGET %s/other\n###\n### create item\nPOST %s/item\n",
			server.URL, server.URL))

	expectedPath := writeTestFile(t, dir, "expected.hresp",
		"HTTP/1.1 201 Created\n\ncreated\n")

	out, code := runBinary(t, binary, "-f", filePath, "-n", "create item", "-e", expectedPath)
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "201 Created")
}

func TestCLI_NegativeIndex(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		"GET http://example.com/a\n")

	out, code := runBinary(t, binary, "-f", filePath, "-i", "-1")
	assert.Equal(t, 1, code)
	assert.Contains(t, out, "out of range")
}

func TestCLI_List(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		"### login\nGET http://example.com/login\n###\n### logout\nGET http://example.com/logout\n")

	out, code := runBinary(t, binary, "-f", filePath, "--list")
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "0  login")
	assert.Contains(t, out, "1  logout")
}

func TestCLI_ListUnnamed(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		"GET http://example.com/a\n###\nGET http://example.com/b\n")

	out, code := runBinary(t, binary, "-f", filePath, "--list")
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "0  (unnamed)")
	assert.Contains(t, out, "1  (unnamed)")
}

func TestCLI_ListMissingFile(t *testing.T) {
	binary := buildBinary(t)
	out, code := runBinary(t, binary, "--list")
	assert.Equal(t, 2, code)
	assert.Contains(t, out, "-f <file> is required")
}

func TestCLI_FailOnError_4xx(t *testing.T) {
	server := startMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, "not found")
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("GET %s/missing\n", server.URL))

	// Without --fail-on-error: exits 0
	_, code := runBinary(t, binary, "-f", filePath)
	assert.Equal(t, 0, code)

	// With --fail-on-error: exits 1
	_, code = runBinary(t, binary, "-f", filePath, "--fail-on-error")
	assert.Equal(t, 1, code)
}

func TestCLI_FailOnError_Success(t *testing.T) {
	server := startMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("GET %s/ok\n", server.URL))

	// With --fail-on-error but 200 response: exits 0
	_, code := runBinary(t, binary, "-f", filePath, "--fail-on-error")
	assert.Equal(t, 0, code)
}

func TestCLI_DefineFlag(t *testing.T) {
	server := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("GET %s/protected\nAuthorization: Bearer {{token}}\n", server.URL))

	// With -D token=test-token
	_, code := runBinary(t, binary, "-f", filePath, "-D", "token=test-token")
	assert.Equal(t, 0, code)
}

func TestCLI_DefineFlagLong(t *testing.T) {
	server := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer another" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("GET %s/protected\nAuthorization: Bearer {{token}}\n", server.URL))

	// With --define token=another
	_, code := runBinary(t, binary, "-f", filePath, "--define", "token=another")
	assert.Equal(t, 0, code)
}

func TestCLI_DefineFlagMultiple(t *testing.T) {
	server := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "key123" || r.Header.Get("X-Env") != "staging" {
			http.Error(w, "bad headers", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http",
		fmt.Sprintf("GET %s/protected\nX-API-Key: {{api_key}}\nX-Env: {{env}}\n", server.URL))

	// Multiple -D flags
	_, code := runBinary(t, binary, "-f", filePath, "-D", "api_key=key123", "-D", "env=staging")
	assert.Equal(t, 0, code)
}

func TestCLI_DefineFlagInvalid(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()
	filePath := writeTestFile(t, dir, "test.http", "GET http://example.com\n")

	_, code := runBinary(t, binary, "-f", filePath, "-D", "invalid")
	assert.Equal(t, 1, code)
}
