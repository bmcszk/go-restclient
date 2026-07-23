---
# go-restclient-ubw6
title: Store response map in ParsedFile after execution
status: completed
type: task
priority: high
created_at: 2026-07-23T16:00:43Z
updated_at: 2026-07-23T16:05:05Z
parent: go-restclient-v0b5
---

go-restclient-ubw6  in-progress   task  [high]    
Store response map in ParsedFile after execution  
──────────────────────────────────────────────────
parent: go-restclient-v0b5                        
──────────────────────────────────────────────────

## Done
- [x] Added `ResponseMap map[string]*Response` to `ParsedFile`
- [x] Initialized map in `ExecuteFile` before loop
- [x] Stored response after execution keyed by `restClientReq.Name`
- [x] Compiles clean
