---
# go-restclient-4r1g
title: 'C3: control merge — loop+disabled+disabled_cond+sleep → client_execute_control_test.go (832ln, 42 tests)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T12:34:45Z
updated_at: 2026-10-15T13:50:39Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-1sgh
---

## Triage (brainer endgame re-plan 2026-10-15, disk-verified)
Merge loop(149?)+disabled+disabled_cond+sleep → client_execute_control_test.go 832ln/42 tests.
## Acceptance criteria
- [ ] 4 → 1; 832ln; 42 tests
## Gates (every wave)
gofmt empty; build/vet 0; golangci-lint `0 issues.`; go test -count=1 both pkgs; inventory 270; zero semantic edits (moves-only + import dedup); CI watched to terminal on PR #84.
## Skills for coder
- ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
## Delegation
pi contract per pi-delegation skill (-t +codemode + scout recon; GIT JAIL; orchestrator re-gates, commits with Refs:, pushes).
Blocked-by: S5 (go-restclient-1sgh)

## Proof of Work (orchestrator-verified)
- loop(396)+disabled(93)+disabled_cond(194)+sleep(149) → client_execute_control_test.go (815ln < 1000); 3 files deleted
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory
- Residual: none.
