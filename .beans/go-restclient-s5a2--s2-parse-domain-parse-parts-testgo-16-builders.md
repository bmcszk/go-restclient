---
# go-restclient-s5a2
title: 'S2: parse domain → parse_parts_test.go (16 builders)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T11:20:03Z
updated_at: 2026-10-15T11:42:57Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-nwn6
---

## Scope
NEW parse_parts_test.go + parseState: parsingFile, parsingRequestAt, parsed* (16 builders) out of ext.
## Acceptance criteria
- [ ] parseState embedded; builders return *parts
- [ ] gates green, 270 tests, zero scenario diffs

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

## Proof of Work (orchestrator-verified)
- parseState created (parsedFile, parseErr + parse helpers), 16 builders moved to *parseState returning p.root; file parse_parts_test.go NEW
- root wiring now explicit per-domain in newParts (p.captureState.root/p.parseState.root = p) after promotion-ambiguity fix
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory, zero scenario diffs
- Residual: none.
