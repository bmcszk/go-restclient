---
# go-restclient-r1hr
title: 'CI: make check failing in workflow run 30082352529'
status: completed
type: bug
priority: high
created_at: 2026-07-24T09:33:34Z
updated_at: 2026-07-24T09:58:47Z
---

go-restclient-r1hr  completed   bug  [high]       
CI: make check failing in workflow run 30082352529
──────────────────────────────────────────────────
                                                  

  GitHub Actions workflow run https://github.com/bmcszk/go-                   
  restclient/actions/runs/30082352529 is failing.                             
                                                                              
  Tasks:                                                                      
                                                                              
  [ ] Check workflow logs for errors                                          
  [ ] Fix any lint or test failures                                           
  [ ] Ensure CI passes on feature/cli branch

## Summary

Fixed all lint and test issues:
- Removed duplicate workflow file (`go_test.yml`)
- Fixed `kong.Parse` return value not checked
- Refactored `run` function - extracted `setupClient`
- Refactored `findRequestIndex` - extracted `findByName` and `findByIndex`
- Fixed line length issues in `executeRequests` and `executeSingle`
- Removed unused `executeAll` function
- Fixed test expectation for mutually exclusive flags

All 231 tests pass, 0 lint issues.
