package main_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestCLI_NoArgs(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenABuiltBinary()

	when.
		whenRunningNoArgs()

	then.
		thenExitCodeIs(80).and().
		thenOutputContains("missing flags: --file=STRING")
}

func TestCLI_MissingFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenABuiltBinary()

	when.
		whenRunningWithArgs("-f", "/nonexistent/file.http")

	then.
		thenExitCodeIs(1).and().
		thenOutputContains("error")
}

func TestCLI_SingleRequest(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "hello")
		}).and().
		givenAnHttpFile(fmt.Sprintf("GET %s/health\n", given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningAll()

	then.
		thenExitCodeIs(0).and().
		thenOutputContains("200 OK").and().
		thenOutputContains("hello")
}

func TestCLI_SelectByName(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		givenAnHttpFile(fmt.Sprintf("### get user\nGET %s/a\n###\n### get order\nGET %s/b\n",
			given.serverURL, given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningNamed("get order")

	then.
		thenExitCodeIs(0).and().
		thenOutputContains("200 OK").and().
		thenServerGotPaths("/b")
}

func TestCLI_SelectByIndex(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		givenAnHttpFile(fmt.Sprintf("GET %s/first\n###\nGET %s/second\n###\nGET %s/third\n",
			given.serverURL, given.serverURL, given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningByIndex("1")

	then.
		thenExitCodeIs(0).and().
		thenOutputContains("200 OK").and().
		thenServerGotPaths("/second")
}

func TestCLI_NameNotFound(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAnHttpFile("GET http://example.com/a\n").and().
		givenABuiltBinary()

	when.
		whenRunningNamed("nonexistent")

	then.
		thenExitCodeIs(1).and().
		thenOutputContains("not found")
}

func TestCLI_IndexOutOfRange(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAnHttpFile("GET http://example.com/a\n").and().
		givenABuiltBinary()

	when.
		whenRunningByIndex("5")

	then.
		thenExitCodeIs(1).and().
		thenOutputContains("out of range")
}

func TestCLI_NameAndIndexMutuallyExclusive(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAnHttpFile("GET http://example.com/a\n").and().
		givenABuiltBinary()

	when.
		whenRunningWithArgs("-f", given.httpFile, "-n", "foo", "-i", "0")

	then.
		thenExitCodeIs(1).and().
		thenOutputContains("mutually exclusive")
}

func TestCLI_ExpectedPass(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Custom", "yes")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "body-ok")
		}).and().
		givenAnHttpFile(fmt.Sprintf("GET %s/check\n", given.serverURL)).and().
		givenAnExpectedFile("HTTP/1.1 200 OK\nX-Custom: yes\n\nbody-ok\n").and().
		givenABuiltBinary()

	when.
		whenRunningAllWithExpected()

	then.
		thenExitCodeIs(0).and().
		thenOutputContains("200 OK")
}

func TestCLI_ExpectedFail(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "actual-body")
		}).and().
		givenAnHttpFile(fmt.Sprintf("GET %s/check\n", given.serverURL)).and().
		givenAnExpectedFile("HTTP/1.1 200 OK\n\nexpected-body\n").and().
		givenABuiltBinary()

	when.
		whenRunningAllWithExpected()

	then.
		thenExitCodeIs(1).and().
		thenOutputContains("error")
}

func TestCLI_SelectByNameWithExpected(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, "created")
		}).and().
		givenAnHttpFile(fmt.Sprintf("### skip this\nGET %s/other\n###\n### create item\nPOST %s/item\n",
			given.serverURL, given.serverURL)).and().
		givenAnExpectedFile("HTTP/1.1 201 Created\n\ncreated\n").and().
		givenABuiltBinary()

	when.
		whenRunningNamedWithExpected("create item")

	then.
		thenExitCodeIs(0).and().
		thenOutputContains("201 Created")
}

func TestCLI_NegativeIndex(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAnHttpFile("GET http://example.com/a\n").and().
		givenABuiltBinary()

	when.
		whenRunningByIndexLong("-1")

	then.
		thenExitCodeIs(1).and().
		thenOutputContains("out of range")
}

func TestCLI_List(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAnHttpFile("### login\nGET http://example.com/login\n###" +
			"\n### logout\nGET http://example.com/logout\n").and().
		givenABuiltBinary()

	when.
		whenRunningList()

	then.
		thenExitCodeIs(0).and().
		thenOutputContains("0  login").and().
		thenOutputContains("1  logout")
}

func TestCLI_ListUnnamed(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAnHttpFile("GET http://example.com/a\n###\nGET http://example.com/b\n").and().
		givenABuiltBinary()

	when.
		whenRunningList()

	then.
		thenExitCodeIs(0).and().
		thenOutputContains("0  (unnamed)").and().
		thenOutputContains("1  (unnamed)")
}

func TestCLI_ListMissingFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenABuiltBinary()

	when.
		whenRunningListWithoutFile()

	then.
		thenExitCodeIs(80).and().
		thenOutputContains("missing flags: --file=STRING")
}

func TestCLI_FailOnError_4xx(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, "not found")
		}).and().
		givenAnHttpFile(fmt.Sprintf("GET %s/missing\n", given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningAll()

	then.
		thenExitCodeIs(0)

	when.
		whenRunningAllWithFailOnError()

	then.
		thenExitCodeIs(1)
}

func TestCLI_FailOnError_Success(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		givenAnHttpFile(fmt.Sprintf("GET %s/ok\n", given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningAllWithFailOnError()

	then.
		thenExitCodeIs(0)
}

func TestCLI_DefineFlag(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(bearerChecker("test-token")).and().
		givenAnHttpFile(fmt.Sprintf("GET %s/protected\nAuthorization: Bearer {{token}}\n", given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningAllWithDefine("token=test-token")

	then.
		thenExitCodeIs(0)
}

func TestCLI_DefineFlagLong(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(bearerChecker("another")).and().
		givenAnHttpFile(fmt.Sprintf("GET %s/protected\nAuthorization: Bearer {{token}}\n", given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningWithArgs("-f", given.httpFile, "--all", "--define", "token=another")

	then.
		thenExitCodeIs(0)
}

func TestCLI_DefineFlagMultiple(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(apiKeyEnvChecker()).and().
		givenAnHttpFile(fmt.Sprintf("GET %s/protected\nX-API-Key: {{api_key}}\nX-Env: {{env}}\n",
			given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningAllWithDefine("api_key=key123", "env=staging")

	then.
		thenExitCodeIs(0)
}

func TestCLI_EnvFlagRunsNamedDotenv(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(keyEchoHandler).and().
		givenADotenvFile(".env", "KEY=default\n").and().
		givenADotenvFile(".env.staging", "KEY=staging\n").and().
		givenAnHttpFile(fmt.Sprintf("GET %s/echo\nX-Key: {{$dotenv KEY}}\n", given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningAllWithEnv("staging")

	then.
		thenExitCodeIs(0).and().
		thenServerGotHeaderOnPath("/echo", "X-Key", "staging").and().
		thenOutputContains("staging")
}

func TestCLI_EnvFlagAbsentKeepsDefaultDotenv(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(keyEchoHandler).and().
		givenADotenvFile(".env", "KEY=default\n").and().
		givenADotenvFile(".env.staging", "KEY=staging\n").and().
		givenAnHttpFile(fmt.Sprintf("GET %s/echo\nX-Key: {{$dotenv KEY}}\n", given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningAll()

	then.
		thenExitCodeIs(0).and().
		thenServerGotHeaderOnPath("/echo", "X-Key", "default").and().
		thenOutputContains("default")
}

func TestCLI_NameSelectRunsRefChain(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		givenAnHttpFile(fmt.Sprintf("### get protected\n# @ref login\nGET %s/protected\n###"+
			"\n### login\nPOST %s/login\n",
			given.serverURL, given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningNamed("get protected")

	then.
		thenExitCodeIs(0).and().
		thenServerGotPaths("/login", "/protected")
}

func TestCLI_AfterComposableWithRefChain(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		givenAnHttpFile(fmt.Sprintf("### get protected\n# @ref login\nGET %s/protected\n"+
			"###\n### login\nPOST %s/login\n###\n### health\nGET %s/health\n",
			given.serverURL, given.serverURL, given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningNamedWithAfter("get protected", "health")

	then.
		thenExitCodeIs(0).and().
		thenServerGotPaths("/health", "/login", "/protected")
}

func TestCLI_DefineFlagInvalid(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAnHttpFile("GET http://example.com\n").and().
		givenABuiltBinary()

	when.
		whenRunningAllWithDefine("invalid")

	then.
		thenExitCodeIs(1)
}

func TestCLI_VersionFlag(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenABuiltBinary()

	when.
		whenRunningVersionFlag("--version")

	then.
		thenExitCodeIs(0).and().
		thenOutputTrimmedIsNotEmpty().and().
		thenOutputNotContains("missing flags")
}

func TestCLI_VersionShortFlag(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenABuiltBinary()

	when.
		whenRunningVersionFlag("-V")

	then.
		thenExitCodeIs(0).and().
		thenOutputTrimmedIsNotEmpty().and().
		thenOutputNotContains("missing flags")
}

func TestCLI_DisabledPrintsSkipLine(t *testing.T) {
	given, when, then := newParts(t)

	given.
		givenAMockServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		givenAnHttpFile(fmt.Sprintf("### disabled\n# @disabled\nGET %s/skipped\n###\nGET %s/ran\n",
			given.serverURL, given.serverURL)).and().
		givenABuiltBinary()

	when.
		whenRunningAll()

	then.
		thenExitCodeIs(0).and().
		thenOutputContains("SKIP").and().
		thenServerGotPaths("/ran")
}
