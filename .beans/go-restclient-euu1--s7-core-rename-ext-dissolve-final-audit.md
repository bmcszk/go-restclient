---
# go-restclient-euu1
title: 'S7: core rename + ext dissolve + final audit'
status: todo
type: task
priority: normal
created_at: 2026-10-15T11:20:04Z
updated_at: 2026-10-15T11:20:42Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-wlqn
---

## Scope
fluent_parts_test.go→core_parts_test.go (lifecycle+response asserts only); DELETE fluent_parts_ext_test.go (must be empty by now); sweep grep -n 'p\.T\.' *_parts_test.go = 0; re-verify file count + all <1000 ln; 270 tests.
## Acceptance criteria
- [ ] core_parts_test.go; ext deleted; QF1008 sweep clean
- [ ] audit: file count recorded, all files <1000, inventory 270

## Triage (brainer completion plan 2026-10-15)
Part of single-PR delivery (#84 → master, absorbs #83). Waves strictly serial: every wave edits fluent_parts_test.go (struct embed + root wiring); zero-scenario-diff gate needs stable base. Pattern to replicate = capture pilot: anonymous embed, root *parts back-pointer, builders return p.root, var _ = []any{domainState{}} guard.
## Gates (per wave)
gofmt empty; build/vet 0; golangci-lint '0 issues.'; go test -count=1 green; Test inventory 270 no dupes; zero scenario-file diffs; CI watched to terminal.
## Skills for coder
- ~/.agents/skills/go-dev
- go-restclient-fluent-dsl skill (Hermes) — DSL layout table + hard rules
- docs/fluent-testing-guideline.md §6-12 (in-repo)
## Delegation
pi contract (pi-delegation skill): GIT JAIL + -t +codemode + scout recon; orchestrator re-gates + commits 'Refs: <bean-id>' + pushes to reorg/parts-dsl-split.
