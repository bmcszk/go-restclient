---
# go-restclient-4m54
title: 'C5: refs merge — execute_refs+response_refs → client_execute_refs_test.go (422ln, 18 tests)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T12:34:45Z
updated_at: 2026-10-15T14:04:41Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-1sgh
---

## Triage (brainer endgame re-plan 2026-10-15, disk-verified)
Merge execute_refs(215)+response_refs(207) → client_execute_refs_test.go 422ln/18 tests.
## Acceptance criteria
- [ ] 2 → 1; 422ln; 18 tests
## Gates (every wave)
gofmt empty; build/vet 0; golangci-lint `0 issues.`; go test -count=1 both pkgs; inventory 270; zero semantic edits (moves-only + import dedup); CI watched to terminal on PR #84.
## Skills for coder
- ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
## Delegation
pi contract per pi-delegation skill (-t +codemode + scout recon; GIT JAIL; orchestrator re-gates, commits with Refs:, pushes).
Blocked-by: S5 (go-restclient-1sgh)

## Proof of Work (orchestrator-verified)
- response_refs(207) merged into client_execute_refs_test.go (416ln); response_refs_test.go deleted
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory
- Residual: none.
