---
# go-restclient-awnn
title: 'C2: validator merge B — placeholders+headers → validator_placeholders_test.go (672ln, 36 tests)'
status: todo
type: task
priority: normal
created_at: 2026-10-15T12:34:45Z
updated_at: 2026-10-15T12:35:04Z
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
