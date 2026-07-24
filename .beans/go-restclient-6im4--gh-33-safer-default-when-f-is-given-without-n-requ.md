---
# go-restclient-6im4
title: 'gh-33: Safer default when -f is given without -n (require -n or -i)'
status: completed
type: feature
priority: high
created_at: 2026-07-24T11:33:33Z
updated_at: 2026-07-24T12:26:53Z
parent: go-restclient-vidx
---

go-restclient-6im4  in-progress   feature  [high]                  
gh-33: Safer default when -f is given without -n (require -n or -i)
──────────────────────────────────────────────────                 
parent: go-restclient-vidx                                         
──────────────────────────────────────────────────                 
                                                                   

  ## Issue #33                                                                
                                                                              
  Running without -n or -i should require explicit flag.                      
                                                                              
  Tasks:                                                                      
                                                                              
  [ ] Change default behavior to require -n or -i                             
  [ ] Add --all flag for full-file execution                                  
  [ ] Update documentation

## Summary

Changed default behavior to require `-n`, `-i`, or `--all` when running requests:
- Without any selector, CLI now errors: "specify -n NAME, -i INDEX, or --all to run requests"
- Added `--all` flag for explicit full-file execution
- Updated all tests to use `--all` where appropriate

Changes:
- cmd/restclient/main.go: Added `--all` flag, refactored executeRequestsOrAll()
- cmd/restclient/main_test.go: Updated tests to use `--all`

Tests pass. Breaking change for users who relied on running all requests without `-n` or `-i`.
