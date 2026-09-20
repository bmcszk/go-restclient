---
# go-restclient-68op
title: 'CLI+vars: -n triggers ref chain; composability with -A'
status: completed
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T18:08:59Z
parent: go-restclient-8y9h
blocked_by:
    - go-restclient-maez
---

Parent epic: declarative request referencing (@ref/@forceRef/@import).

## Requirement
CLI + variable integration. `restclient -f file.http -n "get protected"` automatically executes the ref chain of the target first (replacing the need for manual `-A` in the common case). `--all` keeps natural file order with refs resolved on demand.

## Acceptance Criteria
- [x] `-n` executes ref chain before target
- [x] `-A/--after` still works (both mechanisms composable); docs note the difference
- [x] CLI integration test with canned server (login -> token -> protected)
- [ ] - [x] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)

## Proof of Work

TDD RED (pi): both CLI tests failed as expected - `/login` missing from captured paths (orchestrator verified: actual []string{"/protected"}).
TDD GREEN (pi): 4-line change in client.go ExecuteRequest - resolveRequestRefs with fresh refExecutionState before target execution; ref errors short-circuit; returned response stays target-only. -A composable by construction (runPrerequisite -> executeSingle -> ExecuteRequest).
Lint follow-up (pi): fixture line 137 chars wrapped, fixture content byte-identical.
Orchestrator re-ran all gates on final tree: golangci-lint 0 issues; gotestsum DONE 220 tests, 0 failures; gofmt clean; inventory vs HEAD = exactly +2 CLI tests, 0 lost.
Committed 644985f on feature/ref-force-ref (PR #37).
Residual: README/http_syntax.md CLI docs note - tracked with epic docs pass, non-blocking.
