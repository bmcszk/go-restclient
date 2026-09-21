---
# go-restclient-atek
title: 'Directive: @loop for N / for item of collection ($index, name0..nameN)'
status: completed
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-20T17:53:33Z
parent: go-restclient-p988
---

Parent epic: execution control directives (@disabled/@loop/@sleep).

## Requirement
`# @loop for N` (N literal or variable) and `# @loop for item of collectionVar` (collectionVar = variable holding JSON array). Each iteration: `$index` (0-based) and `item` available as variables; request re-executed; responses addressable as `name0`, `name1`, ... when request is named (matches httpyac). Unnamed looped requests just run N times.

## Acceptance Criteria
- [x] `for N` and `for item of var` both work; N<=0 and empty collection are no-ops (no response entries, no error)
- [x] `{{item.field}}` substituted per element (URL/headers/body/multipart), `$index` available
- [x] Non-array collection -> clear execution error; unknown collection var -> error naming it
- [x] Fluent tests (12; canned server counts calls); CLI smoke via existing suite; docs in epic docs pass
- [x] `make check` passes (golangci-lint 0 issues, 260 tests 0 failures, 3 consecutive green runs)

## Proof of Work (atek)

- TDD RED verified: 12 tests failed only on undefined LoopFor/LoopExpr/LoopCollection.
- GREEN took 3 rounds: round 1 left LoopZeroOrNegativeIsNoop failing (ExecuteRequest returned Skipped response for zero-loop — semantics bug: zero-loop is a no-op, NOT a skip) + runOneRequest complexity 13. Fixes: executeAndStoreRequest returns nil,nil for no-op loops; runRequestWithLoops extraction; resolveSpecialPlaceholder extraction in variables.go; snapshotLoopState in parser_loop.go; loop resolvers -> variables_loop.go, $random block -> variables_random.go (file-length limits).
- GREEN commit 4b9d069 on feature/execution-control (PR #38), commit-only-green.
- Orchestrator-verified: 260 tests 0 failures x3 consecutive runs (one timing flake observed and ruled out), golangci-lint 0 issues, build/vet clean, gofmt clean.
- Design notes: plain name addressing -> name0 (zero-based first iteration); loop bindings applied in URL/headers/body/multipart substitution.
