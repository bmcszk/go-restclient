---
# go-restclient-lyp1
title: 'Directive: @disabled static skip with Skipped response state'
status: in-progress
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-20T13:47:24Z
parent: go-restclient-p988
---

Parent epic: execution control directives (@disabled/@loop/@sleep).

## Requirement
`# @disabled` (and `// @disabled`) statically skips the request. Skipped requests produce a response entry with a Skipped flag (new field on Response or Error sentinel - pick one, document). No HTTP call made.

## Acceptance Criteria
- [ ] Parser stores disabled flag; executor skips without HTTP call
- [ ] Skipped visible in responses slice + CLI output (e.g. `SKIP` line), not treated as failure
- [ ] Fluent tests + docs
- [ ] - [ ] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)
