---
# go-restclient-v2w413m6
title: 'V2W4: fakerHeaderRule+serverReceivedFakerHeaders from client_execute_system_vars_test.go into capture_parts_test.go'
status: completed
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:27:15Z
parent: go-restclient-495b
---

Triage: brainer v2 plan 2026-10-15. FULL requirement context: clear separation, moves-only, no test-body edits, no //nolint/lint-config, ≤1000 ln.
## Scope
Move fakerHeaderRule type + serverReceivedFakerHeaders method (~35ln, client_execute_system_vars_test.go) into capture_parts_test.go (faker headers are captured-request domain). system_vars → ~805ln/27 tests; capture_parts → ~703ln.
## Acceptance
- fakerHeaderRule only in capture_parts_test.go; system_vars file zero parts decls; 298 parity; gates green; capture_parts <1000
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules

## Proof of Work (orchestrator-verified)
- fakerHeaderRule type + serverReceivedFakerHeaders method moved into capture_parts_test.go (699ln, 56 methods); client_execute_system_vars_test.go 27 tests, 0 parts decls (remaining fakerHeaderRule mentions are test-body usages of the moved type — same package, moves-only rule)
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, inventory 270+2Fuzz
- Residual: none.
