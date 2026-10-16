---
# go-restclient-v2w5gx68
title: 'V2W5: merge client_script_test.go (4 tests) into client_parsefile_test.go; delete script file'
status: completed
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:29:58Z
parent: go-restclient-495b
---

go-restclient-v2w5gx68  in-progress   task  [normal]                                         
V2W5: merge client_script_test.go (4 tests) into client_parsefile_test.go; delete script file
──────────────────────────────────────────────────                                           
parent: go-restclient-495b                                                                   
──────────────────────────────────────────────────                                           
                                                                                             

  Triage: brainer v2 plan 2026-10-15. FULL requirement context: fewer files   
  (REQ4), clear separation, moves-only, no test-body edits, no //nolint/lint- 
  config, ≤1000 ln.                                                           
                                                                              
  ## Scope                                                                    
                                                                              
  Append client_script_test.go's 4 Test funcs verbatim into                   
  client_parsefile_test.go (same parse/execute-file domain), dedupe imports,  
  DELETE client_script_test.go. Result ~606ln/27 tests.                       
                                                                              
  ## Acceptance                                                               
                                                                              
  • client_script_test.go gone; parsefile 27 tests <1000; 298 parity; gates   
  green                                                                       
                                                                              
  ## Skills for coder                                                         
                                                                              
  ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules

## Result (V2W5 done)
- client_parsefile_test.go: 595 ln, 27 Test funcs (23 + 4 verbatim script tests). client_script_test.go deleted.
- Moves-only: git diff = +110/-0 in parsefile (script 119 ln − 9 dup package/import lines = 110 appended lines), 0 test-body edits.
- Gates: gofmt empty; go build 0; go vet 0; golangci-lint `0 issues.`; go test ./... -count=1 ok (root 0.606s, cli 11.727s).
- Inventory: 298 Test + 2 Fuzz = 300 (matches acceptance 298 parity; contract's "270" stale pre-W4 count).
- git status: only client_parsefile_test.go (M), client_script_test.go (D), .beans/ (pre-existing tracker edit).

## Proof of Work (orchestrator-verified)
- client_script_test.go (4 tests) merged into client_parsefile_test.go (596ln, 27 tests); script file DELETED
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, inventory 270+2Fuzz
- Residual: none.
