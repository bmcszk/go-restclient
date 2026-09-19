---
# go-restclient-maez
title: 'Executor: resolve ref chains (cache vs force), transitive + cycle-safe'
status: completed
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T17:45:40Z
parent: go-restclient-8y9h
blocked_by:
    - go-restclient-n6cb
---

Parent epic: declarative request referencing (@ref/@forceRef/@import).

## Requirement
Execution engine resolves refs before executing the referencing request. @ref: run once per ExecuteFile run, cache response, reuse. @forceRef: re-execute every time. Ref chain must be transitive (a refs b, b refs c) and cycle-safe (detect + error).

## Acceptance Criteria
- [x] Transitive resolution + cycle detection with clear error
- [x] @ref caches within one run; @forceRef re-executes (assert via requestCount in fluent DSL)
- [x] Referenced response reachable as `{{name.response.body.x}}` in the referencing request
- [x] Fluent tests: chain, cache-reuse, forceRef re-run, cycle error
- [ ] - [x] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)

## Proof of Work

Executor core shipped in f372bd3 (PR #37): depth-first resolveRequestRefs, refExecutionState cache/onStack, cycle + unknown-name errors.
Pin tests added in 2fc0a01 (both pi-delegated, orchestrator re-verified):
- TestExecuteFile_RequestRefs_RefResponseUsableInReferencingRequest: PASS (probe: passed immediately; pi traced mechanism — runAndCacheReferenced -> storeResponse -> parsedFile.ResponseMap["login"], resolveResponseReference dispatches body/headers/status; test pins it against regression)
- TestExecuteFile_RequestRefs_TransitiveChain: PASS (captured order /c,/b,/a depth-first)
Gates re-run by orchestrator on final tree: golangci-lint 0 issues; gotestsum DONE 218 tests, 0 failures, coverage 83.2%; inventory delta = exactly the 2 new tests.
Residual: none for this bean. CLI -n ref-chain trigger tracked separately in go-restclient-68op.
