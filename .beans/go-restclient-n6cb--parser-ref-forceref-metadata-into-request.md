---
# go-restclient-n6cb
title: 'Parser: @ref / @forceRef metadata into Request'
status: todo
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T14:16:10Z
parent: go-restclient-8y9h
---

Parent epic: declarative request referencing (@ref/@forceRef/@import).

## Requirement
Parser accepts `# @ref <name>` / `# @forceRef <name>` metadata lines, mirroring existing directive handling (`@name`, `@no-redirect`, `@timeout` in parser_state.go). Multiple refs per request; valid before/after request line; unknown ref names fail at parse time (or first execution) with a clear error.

## Acceptance Criteria
- [ ] Refs stored on Request struct (ordered list, forceRef flag per entry)
- [ ] Fluent parser tests: single, multiple, mixed ref/forceRef, malformed ref
- [ ] - [ ] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)
