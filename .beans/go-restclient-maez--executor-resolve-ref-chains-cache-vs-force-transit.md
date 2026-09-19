---
# go-restclient-maez
title: 'Executor: resolve ref chains (cache vs force), transitive + cycle-safe'
status: todo
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T14:16:10Z
parent: go-restclient-8y9h
blocked_by:
    - go-restclient-n6cb
---

Parent epic: declarative request referencing (@ref/@forceRef/@import).

## Requirement
Execution engine resolves refs before executing the referencing request. @ref: run once per ExecuteFile run, cache response, reuse. @forceRef: re-execute every time. Ref chain must be transitive (a refs b, b refs c) and cycle-safe (detect + error).

## Acceptance Criteria
- [ ] Transitive resolution + cycle detection with clear error
- [ ] @ref caches within one run; @forceRef re-executes (assert via requestCount in fluent DSL)
- [ ] Referenced response reachable as `{{name.response.body.x}}` in the referencing request
- [ ] Fluent tests: chain, cache-reuse, forceRef re-run, cycle error
- [ ] - [ ] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)
