---
# go-restclient-2ih6
title: 'C7: system_vars merge — system_vars+hresp_vars+system-vars → client_execute_system_vars_test.go (817ln, 27 tests)'
status: todo
type: task
priority: normal
created_at: 2026-10-15T12:34:45Z
updated_at: 2026-10-15T12:35:04Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-1sgh
---

## Triage (brainer endgame re-plan 2026-10-15, disk-verified)
Merge client_execute_system_vars(600)+multiple_consistency(51)+uuid_consistency(166)+system_vars+hresp_vars → client_execute_system_vars_test.go 817ln/27 tests.
## Acceptance criteria
- [ ] 5 → 1; 817ln; 27 tests
## Gates (every wave)
gofmt empty; build/vet 0; golangci-lint `0 issues.`; go test -count=1 both pkgs; inventory 270; zero semantic edits (moves-only + import dedup); CI watched to terminal on PR #84.
## Skills for coder
- ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
## Delegation
pi contract per pi-delegation skill (-t +codemode + scout recon; GIT JAIL; orchestrator re-gates, commits with Refs:, pushes).
Blocked-by: S5 (go-restclient-1sgh)
