---
# go-restclient-3hy1
title: 'PR #37 review round 1: comment hygiene sweep + remove shape-test bean'
status: completed
type: task
priority: normal
created_at: 2026-09-19T19:03:26Z
updated_at: 2026-09-19T19:13:27Z
---

Apply binding rule (one-line-or-omit) to comments in PR #37 diff. Files: client_execute_refs_test.go, parser_state.go, client.go, request.go. Plus git rm .beans/go-restclient-f50t--shape-test.md. Gates: build/vet/lint/gotestsum/gofmt.

## Acceptance criteria
- [x] All listed test godocs shrunk/deleted per spec
- [x] parser_state.go field/function comments one-liner per spec
- [x] client.go function godocs one-liner per spec
- [x] request.go Imported comment one-liner
- [x] .beans/go-restclient-f50t--shape-test.md removed via `git rm`
- [x] No test functions added or removed (grep test names before/after: identical, 223 tests)
- [x] Gates green: go build/vet/golangci-lint/gotestsum/gofmt

## Summary of Changes
Comment-only sweep across 4 Go files + git rm of a shape-test bean. No logic, no test-function changes.

- client_execute_refs_test.go: deleted 2-line helper godoc on parsedRequestRefs (incl. stale "RED:" TDD artifact); deleted test godocs on ParserRecordsRefAndForceRef, RefRunsReferencedBeforeReferencing, RefCachedWithinRun, ForceRefReruns, CycleFails, UnknownRefFails, TransitiveChain, CrossFileRefAndVars, MissingFileFails, ImportedRequestRefCached (names self-explanatory); shrank RefResponseUsableInReferencingRequest to one line.
- parser_state.go: importedParsedFiles field 4-line comment → 1 line; handleImportDirective 7-line godoc → 1 line; handleRefDirective 2-line → 1 line; finalize 3-line comment → 1 line.
- client.go: runOneRequest 2-line godoc → 1 line; resolveRequestRefs 3-line → 1 line. Kept 1-line "@import'd requests run only when invoked via @ref/@forceRef" inline (earns its place).
- request.go: Imported 3-line comment → 1 line. RequestRef godoc untouched (one line, exported).
- .beans/go-restclient-f50t--shape-test.md: `git rm` (was scrapped shape-probe bean; beans of that kind must not ship).

## Proof of Work
- go build ./... → OK
- go vet ./... → OK (exit 0)
- golangci-lint run ./... → "0 issues."
- gotestsum --junitfile /tmp/round1.xml -- -count=1 ./... → DONE 223 tests, 0 failures (pkg . 164ms, cmd/restclient 12.699s)
- gofmt -l client.go parser_state.go request.go client_execute_refs_test.go → empty output
- Test-name inventory diff (git stash / grep '^func Test' / diff) → identical, 223 tests
- git diff master -- '*.go' grep '^+.*//' → every added comment is single-line

Residual: none.
