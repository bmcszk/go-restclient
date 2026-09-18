---
# go-restclient-t3k3
title: 'FLUENT-TESTS: rewrite all tests to fluent Given/When/Then DSL'
status: todo
type: feature
priority: high
created_at: 2026-09-18T14:54:32Z
updated_at: 2026-09-18T15:03:35Z
---

# Goal

Rewrite the entire test suite to a fluent Given/When/Then DSL. Kill the two-layer style
(root `*_test.go` thin wrappers calling `test.Run*` helpers). Single layer: fluent tests
in package `test` under `test/`.

Style source: `go-integration-tests` skill -> `references/fluent-testing-guideline.md` (MANDATORY)
+ user-specified API (see DSL surface).

# Binding style rules

```go
func TestExecuteFile_SingleRequest(t *testing.T) {
    given, when, then := newParts(t)

    given.
        aHttpServer(returning(200, "user data").forPath("/users")).and().
        aHttpFile(`GET {{server}}/users`)

    when.
        executeFile()

    then.
        noError().and().
        responseCode(200).and().
        responseContains("user data")
}
```

- `parts` struct: embeds `*testing.T`, holds `require *require.Assertions` + ALL test state
  (serverURL, servers, httpFilePath, expectedPath, client, responses, execErr, validationErr, requestCount).
- `newParts(t) (*parts, *parts, *parts)` returns given/when/then = same `*parts` 3x.
- `func (p *parts) and() *parts { return p }`; `.and()` ALWAYS at end of line, never start.
- ONE given chain, ONE when chain, ONE then chain per test. No direct helper calls in test funcs,
  no local variables for test state. Helpers allowed only INSIDE fluent method bodies.
- Declare only the section variables used (`_, when, then := newParts(t)` when no setup).

# DSL surface (binding)

Given:
- `aHttpServer(h http.HandlerFunc)` — httptest.Server; wraps h to count requests; t.Cleanup close.
  Helper builders allowed: `returning(code, body)` / `returning(code, body).forPath(p)` / raw func.
- `aHttpFile(content string)` — writes temp `.http`; `{{server}}` substituted with serverURL
  (requires aHttpServer first when used).
- `aClient(opts ...rc.ClientOption)`
- `withProgrammaticVars(map[string]any)` / `withEnv(k, v string)` (t.Setenv)
- `expectedResponseFile(content string)` — temp `.hresp` for ValidateResponses

When:
- `executeFile()` — client.ExecuteFile(ctx, p.httpFilePath) -> p.responses, p.execErr
- `sendingRequest(name string)` — ParseFile, find request by Name, ExecuteRequest; single-response
- `validateResponses()` — client.ValidateResponses(p.expectedPath, p.responses...)

Then (apply to LAST response unless `responseAt(i)` sets cursor):
- `responseAt(i int)`
- `noError()` — execErr nil AND every response.Error nil
- `errorContains(texts ...string)` / `errorCount(n)` — on execErr (multierror-aware)
- `responseCode(code int)` / `responseContains(s)` / `responseNotContains(s)` / `responseBodyIs(s)` / `responseHeader(k, v)`
- `requestCount(n)` — server hit count
- `validationSucceeds()` / `validationFails(count int, texts ...string)` — on validationErr (multierror-aware)

# DSL extension policy (Tasks 2-3)

The DSL surface above is the binding MINIMUM. Migration work may ADD fluent methods when
parity requires them (e.g. responseMatchesRegexp, responseJSONEq, request-body capture via
a capturing server helper, time-window assertions) — following the same naming/return
conventions. Migration may NOT bypass the fluent pattern (no direct helper calls in tests,
no local test-state variables).

# Layout target state

- DSL core: `test/fluent_parts_test.go` (package test).
- All `Test*` funcs live in `test/*_test.go` (package test), names preserved (TestNewClient,
  TestExecuteFile_*, TestValidateResponses_*, ...).
- Root `*_test.go` wrappers DELETED. `test/Run*` helpers DELETED as files migrate.
- Fixtures: prefer inline `aHttpFile` content; keep fixture files only for big/binary cases.
- Behavior parity: every current test case keeps equivalent assertions (per-file mapping).

# Acceptance criteria

- [ ] DSL core in test/ with full surface above
- [ ] `grep -rE 'test\.Run[A-Z]' .` -> 0 hits; root `*_test.go` gone
- [ ] `go test ./...` green with test count >= baseline 231 junit testcases (gotestsum, includes subtests)
- [ ] `make check` green (lint + tests)
- [ ] Every Test func follows given/when/then shape

## Proof of Work

Baseline (master, e2d1eaf): `gotestsum --junitfile ... -- -cover ./...` -> DONE 231 tests,
PASS (root pkg 81.2% coverage; test pkg = helpers only, zero Test funcs).
Branch: feature/fluent-tests.
