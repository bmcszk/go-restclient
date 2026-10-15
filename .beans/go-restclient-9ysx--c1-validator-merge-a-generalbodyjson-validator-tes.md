---
# go-restclient-9ysx
title: 'C1: validator merge A — general+body+json → validator_test.go (742ln, 41 tests)'
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
Merge validator_general(536)+validator_body(142)+json_validator(64) → validator_test.go 742ln/41 tests.
## Acceptance criteria
- [ ] 3 files → 1; 742ln < 1000; 41 tests counted
## Gates (every wave)
gofmt empty; build/vet 0; golangci-lint `0 issues.`; go test -count=1 both pkgs; inventory 270; zero semantic edits (moves-only + import dedup); CI watched to terminal on PR #84.
## Skills for coder
- ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
## Delegation
pi contract per pi-delegation skill (-t +codemode + scout recon; GIT JAIL; orchestrator re-gates, commits with Refs:, pushes).
Blocked-by: S6 (go-restclient-wlqn)
