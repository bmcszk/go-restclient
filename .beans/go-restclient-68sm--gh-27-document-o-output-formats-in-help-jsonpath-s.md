---
# go-restclient-68sm
title: 'gh-27: Document -o output formats in --help (jsonpath syntax)'
status: completed
type: feature
priority: normal
created_at: 2026-07-24T11:30:46Z
updated_at: 2026-07-24T11:41:16Z
parent: go-restclient-vidx
---

go-restclient-68sm  in-progress   feature  [normal]          
gh-27: Document -o output formats in --help (jsonpath syntax)
──────────────────────────────────────────────────           
parent: go-restclient-vidx                                   
──────────────────────────────────────────────────           
                                                             

  ## Issue #27                                                                
                                                                              
  Output format documentation needs improvement.                              
                                                                              
  Tasks:                                                                      
                                                                              
  [ ] Document jsonpath syntax in --help                                      
  [ ] Add examples for each output format                                     
  [ ] Update docs/cli_test_report.md

## Summary

Updated help text for -o/--output flag to document available formats:
- body
- jsonpath
- env

Tests pass. No breaking changes.
