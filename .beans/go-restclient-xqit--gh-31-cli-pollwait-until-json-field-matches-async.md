---
# go-restclient-xqit
title: 'gh-31: CLI poll/wait until JSON field matches (async API)'
status: scrapped
type: feature
priority: normal
created_at: 2026-07-24T11:32:51Z
updated_at: 2026-07-24T12:11:30Z
parent: go-restclient-vidx
---

go-restclient-xqit  todo   feature  [normal]             
gh-31: CLI poll/wait until JSON field matches (async API)
──────────────────────────────────────────────────       
parent: go-restclient-vidx                               
──────────────────────────────────────────────────       
                                                         

  ## Issue #31                                                                
                                                                              
  Async API submissions need poll/wait functionality.                         
                                                                              
  Tasks:                                                                      
                                                                              
  [ ] Add --wait flag for polling                                             
  [ ] Support JSON field matching                                             
  [ ] Add timeout option                                                      
  [ ] Document async API workflow

## Reasons for Scrapping

YAGNI. The user can already use shell scripts to poll for async APIs. The poll/wait feature is a convenience feature that can be added later if needed.

The existing workflow is:
```bash
#!/bin/bash
# Poll for submission status
while true; do
  status=$(restclient -f redeliveries.http -n getSubmissionStatus -o jsonpath status)
  if [ "$status" = "completed" ] || [ "$status" = "failed" ]; then
    echo "Status: $status"
    break
  fi
  sleep 2
done
```
