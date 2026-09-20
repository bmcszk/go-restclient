---
# go-restclient-2qlf
title: 'Directive: conditional @disabled !expr (variable truthiness)'
status: in-progress
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-20T15:46:49Z
parent: go-restclient-p988
blocked_by:
    - go-restclient-lyp1
---

Parent epic: execution control directives (@disabled/@loop/@sleep). Blocked by @disabled static skip.

## Requirement
`# @disabled !expr`: evaluate expr after variable substitution; if result is truthy ("true", non-zero number, non-empty - define exactly in docs), skip. Evaluation must NOT require a JS engine: variable substitution + literal truthiness only.

## Acceptance Criteria
- [ ] Truthiness rules defined in docs and tested (true/false/1/0/empty/non-empty)
- [ ] Unknown variable in expr -> request NOT silently skipped (error or explicit-false, pick + document)
- [ ] Fluent tests + docs
- [ ] - [ ] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)
