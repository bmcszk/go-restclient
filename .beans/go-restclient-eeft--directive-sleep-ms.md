---
# go-restclient-eeft
title: 'Directive: @sleep <ms>'
status: todo
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T14:16:10Z
parent: go-restclient-p988
---

Parent epic: execution control directives (@disabled/@loop/@sleep).

## Requirement
`# @sleep <ms>`: executor waits given milliseconds before sending the request (after refs resolve, before loop iteration send). Non-negative integer; invalid value -> parse error.

## Acceptance Criteria
- [ ] Sleep honored in library and CLI; 0 valid no-op
- [ ] Fluent test with timing assertion (loose bound)
- [ ] Docs
- [ ] - [ ] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)
