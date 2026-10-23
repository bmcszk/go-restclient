---
# go-restclient-zicv
title: 'C6: file/env merge — parsefile+external_file+dotenv (496ln) + script+parser_script (124ln)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T12:34:45Z
updated_at: 2026-10-15T14:11:12Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-1sgh
---

## Triage (brainer endgame re-plan 2026-10-15, disk-verified)
parsefile(166)+external_file(240)+dotenv(90) → client_parsefile_test.go 496ln; script(63)+parser_script(61) → client_script_test.go 124ln.
## Acceptance criteria
- [ ] 5 → 2; 496ln + 124ln; 23+4 tests
## Gates (every wave)
gofmt empty; build/vet 0; golangci-lint `0 issues.`; go test -count=1 both pkgs; inventory 270; zero semantic edits (moves-only + import dedup); CI watched to terminal on PR #84.
## Skills for coder
- ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
## Delegation
pi contract per pi-delegation skill (-t +codemode + scout recon; GIT JAIL; orchestrator re-gates, commits with Refs:, pushes).
Blocked-by: S5 (go-restclient-1sgh)

## Proof of Work (orchestrator-verified)
- external_file(240)+dotenv(90) merged into client_parsefile_test.go (485ln); parser_script(61) merged into client_script_test.go (119ln); 3 files deleted
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory
- Residual: none.
