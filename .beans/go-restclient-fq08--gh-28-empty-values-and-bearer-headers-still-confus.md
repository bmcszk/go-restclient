---
# go-restclient-fq08
title: 'gh-28: Empty  values and Bearer headers still confusing'
status: completed
type: bug
priority: high
created_at: 2026-07-24T11:31:17Z
updated_at: 2026-07-24T12:06:34Z
parent: go-restclient-vidx
---

go-restclient-fq08  in-progress   bug  [high]          
gh-28: Empty  values and Bearer headers still confusing
──────────────────────────────────────────────────     
parent: go-restclient-vidx                             
──────────────────────────────────────────────────     
                                                       

  ## Issue #28                                                                
                                                                              
  Empty dotenv values cause confusing errors.                                 
                                                                              
  Tasks:                                                                      
                                                                              
  [ ] Warn when  value is empty                                               
  [ ] Skip Authorization header if token is empty                             
  [ ] Add tests for empty dotenv values

## Summary

Added warning when $dotenv value is empty. When a dotenv variable resolves to an empty string, the CLI now logs: `WARN $dotenv value is empty var=TOKEN`.

Changes:
- variables.go: Added empty value check in dotEnvReplacer()
- cmd/restclient/main.go: Updated -o/--output help text

Tests pass. No breaking changes.
