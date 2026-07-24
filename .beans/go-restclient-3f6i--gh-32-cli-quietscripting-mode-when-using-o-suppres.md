---
# go-restclient-3f6i
title: 'gh-32: CLI quiet/scripting mode when using -o (suppress status)'
status: scrapped
type: feature
priority: normal
created_at: 2026-07-24T11:33:12Z
updated_at: 2026-07-24T12:13:48Z
parent: go-restclient-vidx
---

go-restclient-3f6i  in-progress   feature  [normal]            
gh-32: CLI quiet/scripting mode when using -o (suppress status)
──────────────────────────────────────────────────             
parent: go-restclient-vidx                                     
──────────────────────────────────────────────────             
                                                               

  ## Issue #32                                                                
                                                                              
  Accidental full-file run without -n causes issues.                          
                                                                              
  Tasks:                                                                      
                                                                              
  [ ] Add --quiet flag for scripting                                          
  [ ] Suppress warnings when -o is used                                       
  [ ] Add --all flag for explicit full-file run

## Reasons for Scrapping

Current behavior is already correct for scripting:
- `-o body` → stdout is body only
- `-o jsonpath` → stdout is extracted value only
- `-o env` → stdout is key=value only
- Warnings go to stderr
- Status/headers only shown when `-o` is NOT set

```bash
# This already works correctly:
token=$(restclient -f api.http -n authorize -o body 2>/dev/null)
```

YAGNI on `--quiet` flag.
