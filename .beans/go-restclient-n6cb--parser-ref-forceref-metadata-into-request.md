---
# go-restclient-n6cb
title: 'Parser: @ref / @forceRef metadata into Request'
status: completed
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T14:42:52Z
parent: go-restclient-8y9h
---

Parent epic: declarative request referencing (@ref/@forceRef/@import).

## Requirement
Parser accepts `# @ref <name>` / `# @forceRef <name>` metadata lines, mirroring existing directive handling (`@name`, `@no-redirect`, `@timeout` in parser_state.go). Multiple refs per request; valid before/after request line; unknown ref names fail at parse time (or first execution) with a clear error.

## Acceptance Criteria
- [x] Refs stored on Request struct (ordered list, forceRef flag per entry)
- [x] Fluent parser tests: single, multiple, mixed ref/forceRef, malformed ref
- [x] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)

## Proof of Work

TDD RED (pi run 1): 6 tests in client_execute_refs_test.go verified failing: `req.Refs undefined (type *restclient.Request has no field or method Refs)` - confirmed by orchestrator.
TDD GREEN (pi run 2): all 6 pass. Orchestrator re-ran every gate on the final tree:
- go build ./... && go vet ./... && golangci-lint run ./... -> exit 0, `0 issues.`
- gotestsum --junitfile /tmp/refs_green2.xml -- -count=1 -cover ./... -> DONE 216 tests, 0 failures, coverage 82.6%
- Test-func inventory parity: 195 baseline == 195 after (diff empty)
- gofmt: touched files clean (pre-existing drift in untouched files left alone)

Committed f372bd3 on feature/ref-force-ref, PR #37.
Residual: CLI ref-chain trigger and @import are separate beans (go-restclient-68op, go-restclient-qe7j).
