---
# go-restclient-euu1
title: 'S7: core rename + ext dissolve + final audit'
status: completed
type: task
priority: normal
created_at: 2026-10-15T11:20:04Z
updated_at: 2026-10-15T13:37:35Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-wlqn
---

## Triage (brainer endgame re-plan 2026-10-15, disk-verified)
Rename fluent_parts_test.go → parts_test.go (core, 268ln, lifecycle+response asserts only); fluent_parts_ext_test.go must be EMPTY by now → DELETE it (261ln moved via S1-S5). Final fluent_* sweep: `ls fluent_parts*_test.go` empty.
## Acceptance criteria
- [ ] parts_test.go exists; ext deleted; zero fluent_* files
- [ ] gates green, 270 tests
## Gates
gofmt/build/vet 0; lint `0 issues.`; go test -count=1; inventory 270; moves-only; CI terminal on #84.
## Skills for coder
- ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
## Delegation
pi contract per pi-delegation skill.

## Proof of Work (orchestrator-verified)
- ext dissolved: urlHost→client_execute_vars, fakerHeaders+fakerHeaderRule→client_execute_system_vars, 6 client* builders→client_init; fluent_parts_ext_test.go DELETED
- fluent_parts_test.go → parts_test.go (pure rename, similarity 100%)
- fluent sweep: ZERO fluent_* files; only docs ref left (S8)
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory
- Residual: C1-C7 consolidation waves + S8 docs.
