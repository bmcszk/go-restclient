---
# go-restclient-qe7j
title: 'Parser+executor: @import ./file.http (cross-file refs + file-global vars)'
status: todo
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T14:16:10Z
parent: go-restclient-8y9h
blocked_by:
    - go-restclient-maez
---

Parent epic: declarative request referencing (@ref/@forceRef/@import).

## Requirement
`# @import ./other.http` at file top: parse imported file first; its file-global variables and named requests become referenceable (@ref/@forceRef + `{{name.response.*}}`) from the importing file. httpyac semantics: only file-global-scope variables are imported.

## Acceptance Criteria
- [ ] Imported file parsed once; its named requests resolvable via @ref/@forceRef
- [ ] File-global vars from imported file visible in importing file (existing var precedence preserved: programmatic > env > file)
- [ ] Missing import file -> clear error
- [ ] Fluent tests + docs section in http_syntax.md
- [ ] - [ ] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)
