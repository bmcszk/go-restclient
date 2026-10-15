---
# go-restclient-v2w413m6
title: 'V2W4: fakerHeaderRule+serverReceivedFakerHeaders from client_execute_system_vars_test.go into capture_parts_test.go'
status: todo
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:11:28Z
parent: go-restclient-495b
---

Triage: brainer v2 plan 2026-10-15. FULL requirement context: clear separation, moves-only, no test-body edits, no //nolint/lint-config, ≤1000 ln.
## Scope
Move fakerHeaderRule type + serverReceivedFakerHeaders method (~35ln, client_execute_system_vars_test.go) into capture_parts_test.go (faker headers are captured-request domain). system_vars → ~805ln/27 tests; capture_parts → ~703ln.
## Acceptance
- fakerHeaderRule only in capture_parts_test.go; system_vars file zero parts decls; 298 parity; gates green; capture_parts <1000
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
