---
# go-restclient-7a9s
title: 'C4: core merge — core+edgecases+baseurl+sameline+tls+init → client_execute_core_test.go (779ln, 29 tests)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T12:34:45Z
updated_at: 2026-10-15T14:00:48Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-1sgh
---

## Triage (brainer endgame re-plan 2026-10-15, disk-verified)
Merge core+edgecases+baseurl+sameline+tls+init → client_execute_core_test.go 779ln/29 tests.
## Acceptance criteria
- [ ] 6 → 1; 779ln; 29 tests
## Gates (every wave)
gofmt empty; build/vet 0; golangci-lint `0 issues.`; go test -count=1 both pkgs; inventory 270; zero semantic edits (moves-only + import dedup); CI watched to terminal on PR #84.
## Skills for coder
- ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
## Delegation
pi contract per pi-delegation skill (-t +codemode + scout recon; GIT JAIL; orchestrator re-gates, commits with Refs:, pushes).
Blocked-by: S5 (go-restclient-1sgh)

## Proof of Work (orchestrator-verified)
- core(436)+edgecases(108)+baseurl(113)+sameline(54)+tls(25)+client_init(98) → client_execute_core_test.go (803ln < 1000); 5 files deleted
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory
- Residual: none.
