---
# go-restclient-atek
title: 'Directive: @loop for N / for item of collection ($index, name0..nameN)'
status: todo
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T14:16:10Z
parent: go-restclient-p988
---

Parent epic: execution control directives (@disabled/@loop/@sleep).

## Requirement
`# @loop for N` (N literal or variable) and `# @loop for item of collectionVar` (collectionVar = variable holding JSON array). Each iteration: `$index` (0-based) and `item` available as variables; request re-executed; responses addressable as `name0`, `name1`, ... when request is named (matches httpyac). Unnamed looped requests just run N times.

## Acceptance Criteria
- [ ] `for N` and `for item of var` both work; N<=0 and empty collection are no-ops (not errors)
- [ ] `{{item.field}}` works when items are JSON objects
- [ ] Non-array collection variable -> clear error
- [ ] Fluent tests (canned server counts calls), CLI smoke, docs
- [ ] - [ ] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)
