---
# go-restclient-9ysx
title: 'C1: validator merge A — general+body+json → validator_test.go (742ln, 41 tests)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T12:34:45Z
updated_at: 2026-10-15T13:42:37Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-wlqn
---

go-restclient-9ysx  in-progress   task  [normal]                               
C1: validator merge A — general+body+json → validator_test.go (742ln, 41 tests)
──────────────────────────────────────────────────                             
parent: go-restclient-495b                                                     
──────────────────────────────────────────────────                             
                                                                               

  ## Triage (brainer endgame re-plan 2026-10-15, disk-verified)               
                                                                              
  Merge validator_general(536)+validator_body(142)+json_validator(64) →       
  validator_test.go 742ln/41 tests.                                           
                                                                              
  ## Acceptance criteria                                                      
                                                                              
  [ ] 3 files → 1; 742ln < 1000; 41 tests counted                             
                                                                              
  ## Gates (every wave)                                                       
                                                                              
  gofmt empty; build/vet 0; golangci-lint 0 issues.; go test -count=1 both    
  pkgs;                                                                       
  inventory 270; zero semantic edits (moves-only + import dedup); CI watched  
  to                                                                          
  terminal on PR #84.                                                         
                                                                              
  ## Skills for coder                                                         
                                                                              
  • ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules                    
                                                                              
  ## Delegation                                                               
                                                                              
  pi contract per pi-delegation skill (-t +codemode + scout recon; GIT JAIL;  
  orchestrator re-gates, commits with Refs:, pushes). Blocked-by: S6 (go-     
  restclient-wlqn)



## Wave C1 execution report (merged)
- Files: validator_body_test.go (142 L) + json_validator_test.go (64 L) appended into validator_general_test.go → renamed via git mv to validator_test.go (733 lines, <1000 cap).
- Imports deduped (net/http + testing); test bodies and builders verbatim; collision pre-check clean (Body*/JSON* names unique).
- One gofmt fix: extra blank line at body→json seam (whitespace-only hunk).
- Gates: gofmt empty ✓, go build 0 ✓, go vet 0 ✓, golangci-lint "0 issues." ✓, go test ./... -count=1 ok ✓, test count root pkg = 270 ✓ (cmd 28, workspace total 313 incl. subtests).
- git status: RM validator_general_test.go→validator_test.go, D validator_body_test.go, D json_validator_test.go, pre-existing .beans/ mod only.

## Proof of Work (orchestrator-verified)
- validator_general(536)+validator_body(142)+json_validator(64) → validator_test.go (733ln < 1000); 2 source files deleted
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory
- Residual: none.
