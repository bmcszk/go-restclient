---
# go-restclient-kkp9
title: 'RED phase: tests for conditional @disabled !expr'
status: completed
type: task
priority: normal
created_at: 2026-09-20T15:50:39Z
updated_at: 2026-09-20T16:50:15Z
---

## Goal
Write failing tests for `# @disabled !expr` (variable truthiness skip), as RED phase for bean go-restclient-2qlf. No production changes, no commits.

## Subtasks
- [x] Add fixtures under test/data/http_request_files/
- [x] Add parsedRequestDisabledExpr DSL accessor in fluent_parts_ext_test.go
- [x] Write client_execute_disabled_cond_test.go with 10 top-level Test funcs (split per guideline)
- [x] Run gates, capture RED output
- [x] Confirm no production files touched

## RED criteria (per contract)
EXPECTED RED from: undefined symbols (DisabledExpr field on Request), parse handling. Assertion failures when symbols defined = test bug.

## Notes
- Branch: feature/execution-control (HEAD aedb191), tree clean
- Will not commit; orchestrator handles lifecycle

## Summary of Changes

Created RED-phase tests for `# @disabled !expr` (variable truthiness skip). No production code touched.

### Fixtures (test/data/http_request_files/)
- `disabled_cond_flagvar.http` — `# @disabled !{{flag}}` over `GET [[.ServerURL]]/conditional`; parameterized via `WithVars` for the truthy/falsy/zero/one/empty/nonempty/case tests
- `disabled_cond_missing.http` — `# @disabled !{{missing}}` over `GET [[.ServerURL]]/conditional`; no var ever defined; used by `UnknownVariableErrors`

### Test file (client_execute_disabled_cond_test.go)
10 top-level Test funcs (split per fluent-testing-guideline §6 — no t.Run wrappers):
- `TestParseFile_ConditionalDisabledWithBang_TogglesPlainDisabled` — inline `# @disabled !true`; asserts parse OK, `Disabled == false`, `DisabledExpr == "true"`
- `TestExecuteFile_ConditionalDisabled_TruthyExprSkips` — `flag="true"` → skip, 0 hits, response[0].Skipped
- `TestExecuteFile_ConditionalDisabled_FalsyExprRuns` — `flag="false"` → run, 1 hit
- `TestExecuteFile_ConditionalDisabled_ZeroIsFalsy` — `flag=0` → run
- `TestExecuteFile_ConditionalDisabled_NumberOneIsTruthy` — `flag=1` → skip
- `TestExecuteFile_ConditionalDisabled_EmptyStringRuns` — `flag=""` → run
- `TestExecuteFile_ConditionalDisabled_NonEmptySkips` — `flag="anything"` → skip
- `TestExecuteFile_ConditionalDisabled_UnknownVariableErrors` — `{{missing}}` → error containing "undefined variable" + "missing"
- `TestExecuteFile_ConditionalDisabled_CaseInsensitiveTrue` — `flag="TRUE"` → skip
- `TestExecuteFile_ConditionalDisabled_CaseInsensitiveFalse` — `flag="False"` → run

### DSL accessor (fluent_parts_ext_test.go)
- `parsedRequestDisabledExpr(i, want string)` — sibling to `parsedRequestDisabled`; reads `Requests[i].DisabledExpr`

### Variable mechanism chosen
Programmatic vars via `rc.WithVars(map[string]any{...})` because the parser stores the raw expr text and runtime substitutes + evaluates — no need for per-fixture in-file `@var = ...` lines. URL substitution uses `[[.ServerURL]]` via the established `aHttpFileFromTemplate` path (same as `disabled_then_enabled.http`).

## Proof of Work

```
$ gofmt -l client_execute_disabled_cond_test.go fluent_parts_ext_test.go
(empty)

$ go build ./...
(no output — success; production code unchanged)

$ go vet ./...
# github.com/bmcszk/go-restclient_test
# [github.com/bmcszk/go-restclient_test]
vet: ./fluent_parts_ext_test.go:278:48: p.parsedFile.Requests[i].DisabledExpr undefined (type *restclient.Request has no field or method DisabledExpr)

$ go test -run 'ConditionalDisabled' ./... 2>&1 | tail -12
# github.com/bmcszk/go-restclient_test [github.com/bmcszk/go-restclient.test]
./fluent_parts_ext_test.go:278:48: p.parsedFile.Requests[i].DisabledExpr undefined (type *restclient.Request has no field or method DisabledExpr)
FAIL	github.com/bmcszk/go-restclient [build failed]
ok  	github.com/bmcszk/go-restclient/cmd/restclient	(cached) [no tests to run]
FAIL

$ git diff --stat -- '*.go' ':!*_test.go'
(empty — no production .go touched)

$ git status --short
 M fluent_parts_ext_test.go                                (new accessor only)
?? .beans/go-restclient-kkp9--...                          (bean)
?? client_execute_disabled_cond_test.go                    (tests)
?? test/data/http_request_files/disabled_cond_flagvar.http (fixture)
?? test/data/http_request_files/disabled_cond_missing.http (fixture)

$ git log --oneline -1
aedb191 chore(beans): close go-restclient-lyp1 (@disabled shipped in 7a38ab1), claim 2qlf
(unchanged — no commits made)
```

RED confirmed: undefined `DisabledExpr` field — exactly the "EXPECTED RED = undefined symbols like DisabledExpr" class per the contract. After green, the field exists, the build resolves, and the 10 new tests + 5 existing guards all run.

Residual: none. Handing back to orchestrator for the GREEN pass (add `DisabledExpr` to `Request`, update `handleDisabledDirective` to accept `!expr`, evaluate at runtime in `runOneRequest` before the disabled-skip branch, produce a runtime error when an undefined variable is referenced).
