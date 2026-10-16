---
# go-restclient-awnn
title: 'C2: validator merge B — placeholders+headers → validator_placeholders_test.go (672ln, 36 tests)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T12:34:45Z
updated_at: 2026-10-15T13:46:48Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-wlqn
---

## Triage (brainer endgame re-plan 2026-10-15, disk-verified)
Merge validator_placeholders(466)+validator_headers(206) → validator_placeholders_test.go 672ln/36 tests.
## Acceptance criteria
- [ ] 2 → 1; 672ln; 36 tests
## Gates (every wave)
gofmt empty; build/vet 0; golangci-lint `0 issues.`; go test -count=1 both pkgs; inventory 270; zero semantic edits (moves-only + import dedup); CI watched to terminal on PR #84.
## Skills for coder
- ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
## Delegation
pi contract per pi-delegation skill (-t +codemode + scout recon; GIT JAIL; orchestrator re-gates, commits with Refs:, pushes).
Blocked-by: S6 (go-restclient-wlqn)

## Proof of Work (orchestrator-verified)
- validator_placeholders(466)+validator_headers(206) → validator_placeholders_test.go (666ln < 1000); validator_headers_test.go deleted
- pi process was killed post-edit pre-report (EXIT:143) — work itself complete: gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory
- Residual: none.
