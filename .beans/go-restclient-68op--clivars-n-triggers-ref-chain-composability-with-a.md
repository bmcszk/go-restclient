---
# go-restclient-68op
title: 'CLI+vars: -n triggers ref chain; composability with -A'
status: in-progress
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T17:45:40Z
parent: go-restclient-8y9h
blocked_by:
    - go-restclient-maez
---

Parent epic: declarative request referencing (@ref/@forceRef/@import).

## Requirement
CLI + variable integration. `restclient -f file.http -n "get protected"` automatically executes the ref chain of the target first (replacing the need for manual `-A` in the common case). `--all` keeps natural file order with refs resolved on demand.

## Acceptance Criteria
- [ ] `-n` executes ref chain before target
- [ ] `-A/--after` still works (both mechanisms composable); docs note the difference
- [ ] CLI integration test with canned server (login -> token -> protected)
- [ ] - [ ] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)
