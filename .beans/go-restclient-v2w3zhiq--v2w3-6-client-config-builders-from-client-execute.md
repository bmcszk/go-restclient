---
# go-restclient-v2w3zhiq
title: 'V2W3: 6 client-config builders from client_execute_core_test.go into parts_test.go (base)'
status: todo
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:11:28Z
parent: go-restclient-495b
---

Triage: brainer v2 plan 2026-10-15. FULL requirement context: clear separation, base parts struct = most basic builders (REQ2), moves-only, no test-body edits, no //nolint/lint-config.
## Scope
Move 6 client-config builder methods (aClientWithDefaults, clientBaseURLIs, clientBaseURLIsEmpty, clientDefaultHeaderIs, clientDefaultHeadersEmpty, clientExists; client_execute_core_test.go L717-769) into parts_test.go (base). client_execute_core_test.go → ~751ln/29 tests, parts_test.go ~303ln.
## Acceptance
- 6 builders on p *parts live in parts_test.go; core file zero client-config builders; 298 parity; gates green; both files <1000
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
