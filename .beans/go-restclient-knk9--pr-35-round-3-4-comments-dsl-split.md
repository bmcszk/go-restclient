---
# go-restclient-knk9
title: 'PR #35 round 3 — 4 comments + DSL split'
status: completed
type: task
priority: normal
created_at: 2026-09-19T12:19:15Z
updated_at: 2026-09-19T13:08:31Z
---

## Scope

Address PR #35 round 3 review comments and generalizations.

## Comments

- [x] C1: Move graphqlResponse/graphqlError + writeGraphQLResponse helper + 7 inline GraphQL mock servers into DSL (`aGraphQLServer`)
- [x] C2: Shrink 6-line migration godoc in validator_options_test.go; grep+sweep other 5+ line migration-history godocs
- [x] C3: Convert multi-line (3+) inline external-file content to fixture files (test_vars.json, test_static.json); extend DSL
- [x] C4: Split DSL into domain files (general/validate/graphql). fluent_parts_test.go = core only; fluent_parts_ext_test.go = general; fluent_parts_validator_test.go = validator; NEW fluent_parts_graphql_test.go

## Generalization

- [x] Other 3+ line inline external-file content → fixtures (none beyond C3 hit)
- [x] Other 4+ line migration-history godoc → shrink (also shrunk the legacy-asserts block in client_execute_inplace_vars_test.go)
- [x] Other GraphQL-like mock infra duplicated in test bodies → DSL (no other graphql mocks)

## Gates

- go build ./... → 0
- go vet ./... → 0
- golangci-lint run ./... → 0
- gotestsum -- -count=1 -cover ./... → 0 failures
- gofmt clean on touched files

## Summary of Changes

DSL split (6 files, all <1000 lines):

| File                              | Lines |
| --------------------------------- | ----- |
| fluent_parts_test.go              |   253 |
| fluent_parts_ext_test.go          |   621 |
| fluent_parts_capture_test.go      |   466 |
| fluent_parts_validator_test.go    |   204 |
| fluent_parts_graphql_test.go      |    34 |
| fluent_parts_cookies_redirects_test.go | 107 |

- fluent_parts_test.go — core: parts struct, newParts, and(), lifecycle (aHttpServer/aClient/executeFile/validateResponses/responseAt/current), base assertions (responseCode/Contains/BodyIs/Header/Values/Empty/Count/HasNoError/HasError, noError, errorContains, requestCount), client config snapshot.
- fluent_parts_ext_test.go — general extensions: fixtures/templates/external files, env vars, capture/tracking, serverReceived* family (base 3), canned/echo servers, faker rules, aHttpFile* helpers, mock transport, client config assertions.
- fluent_parts_capture_test.go — capture/tracking/JSON-path utilities + request body assertions + tracking timestamp assertions + serverReceived header helpers + body matchers.
- fluent_parts_validator_test.go — validator DSL + validateResponses/validationSucceeds/validationFails + expected response fixture/file setters + responsesValidateAgainst[Fixture].
- fluent_parts_graphql_test.go — aGraphQLServer + graphqlResponse/graphqlError.
- fluent_parts_cookies_redirects_test.go — cookie/redirect/.rest/jsonEcho DSL.

C1 (GraphQL DSL): fluent_parts_graphql_test.go with `aGraphQLServer(responses ...graphqlResponse)` + graphqlResponse/graphqlError types. writeGraphQLResponse helper deleted. All 7 inline mocks in client_execute_graphql_test.go (single + batch) converted to the new DSL call.

C2 (godoc trim): shrunk the 6-line godoc above TestValidateResponsesWithOptions_OutOfRangeIndex and the 5-line "legacy asserts" godoc above TestExecuteFile_InPlace_VariableDefinedBySystemVariable to 1 line. Round 4 sweep: ~30 DSL godocs across all 6 fluent_parts_*_test.go files trimmed to 1 line; split-banner comment removed from fluent_parts_ext_test.go.

C3 (fixtures): test/data/external_files/test_vars.json and test_static.json fixtures. Added `aHttpFileWithExternalFileFixture` / `aHttpFileWithExternalFileStaticFixture` DSL methods. Two 4-line inline JSON literals in client_execute_external_file_test.go converted to fixture calls.

C4 (DSL split): 6 files, see table above.

Round 4 cleanup (linter hygiene):
- Deleted all 8 `var _ = []any{...}` DSL-symbol appeasement blocks repo-wide (fluent_parts_*.go plus validator_general_test.go).
- Deleted `var _ = time.Time{}` appeasement block in client_execute_vars_test.go. The unused-linter now guards directly without these appeasements.
- Deleted 5 dead DSL methods that became zero-callers once the var-blocks were removed: serverReceivedHeaderMatchesRegexp, serverReceivedHeaderNotContains, serverReceivedHeaderFieldCountIs, serverReceivedHeaderContains, anExternalFileBytes.
- Unified fixtures directory: test/fixtures/ dissolved into test/data/ via `git mv` (expected.hresp → test/data/http_response_files/, test_chaining.http + test_multi.http → test/data/http_request_files/). docs/cli_test_report.md paths updated.

## Proof of Work

Final gate run after round 4 (var-block + dead-method deletions + fixtures unification):

```
go build ./...                                          → 0
go vet ./...                                            → 0
golangci-lint run ./...                                 → 0 issues
gofmt -l <touched files>                                → (empty)
gotestsum -- -count=1 -cover ./...                      → 210 tests, 0 failures (cov ~82%)
```

DSL file sizes (final tree):

| File                              | Lines |
| --------------------------------- | ----- |
| fluent_parts_test.go              |   253 |
| fluent_parts_ext_test.go          |   621 |
| fluent_parts_capture_test.go      |   466 |
| fluent_parts_validator_test.go    |   204 |
| fluent_parts_graphql_test.go      |    34 |
| fluent_parts_cookies_redirects_test.go | 107 |

All files < 1000 lines (project AGENTS.md limit).

Deletion evidence:
- `grep -rn "var _ = \[\]any{" --include="*.go" .` → 0 matches
- `grep -n "var _ = time.Time{}" client_execute_vars_test.go` → 0 matches
- dead methods (serverReceivedHeaderMatchesRegexp, serverReceivedHeaderNotContains, serverReceivedHeaderFieldCountIs, serverReceivedHeaderContains, anExternalFileBytes): `grep -rn "func.*<name>"` → 0 matches each
- `ls test/fixtures/` → No such file or directory; all fixtures under test/data/{http_request_files,http_response_files,external_files,...}

Status: changes UNCOMMITTED in working tree (per orchestrator instruction).

Residual: none.
