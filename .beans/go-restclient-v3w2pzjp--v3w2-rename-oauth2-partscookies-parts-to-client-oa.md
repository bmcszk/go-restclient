---
# go-restclient-v3w2pzjp
title: 'V3W2: rename oauth2_parts/cookies_parts to client_oauth2_parts_test.go/client_cookies_redirects_parts_test.go (+2 PF methods to parsefile pair)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T15:55:30Z
updated_at: 2026-10-15T16:19:20Z
parent: go-restclient-495b
---

go-restclient-v3w2pzjp  in-progress   task  [normal]                                                                                           
V3W2: rename oauth2_parts/cookies_parts to client_oauth2_parts_test.go/client_cookies_redirects_parts_test.go (+2 PF methods to parsefile pair)
──────────────────────────────────────────────────                                                                                             
parent: go-restclient-495b                                                                                                                     
──────────────────────────────────────────────────                                                                                             
                                                                                                                                               

  FULL REQUIREMENTS: same as epic v3 (3-file pairs, parts file ONLY serves its
  pair, anonymous embedding already done in V3W1). Moves-only; no test-body   
  edits; no //nolint/lint-config; ≤1000 ln.                                   
                                                                              
  ## Scope                                                                    
                                                                              
  • git mv oauth2_parts_test.go → client_oauth2_parts_test.go (pairs          
  client_oauth2_test.go; impl client_oauth2.go)                               
  • git mv cookies_parts_test.go → client_cookies_redirects_parts_test.go     
  (pairs client_cookies_redirects_test.go); move its 2 parsefile-used methods 
  (aHttpFile-family? verify by grep which 2 methods client_parsefile_test.go  
  uses) OUT into a temp home (they go to client_parsefile_parts_test.go in    
  V3W3 — if V3W3 runs after, put them in parts_test.go temporarily? NO: do    
  V3W3's parsefile-pair creation is separate wave; here just move the 2       
  methods into client_parsefile_test.go-adjacent... brainer plan says cookies 
  pair −2 PF methods +withServerAddressVars; verify with grep what parsefile  
  actually uses from cookies file and move those methods to parts_test.go base
  temporarily if their final home doesn't exist yet)                          
  • withServerAddressVars (from server domain) → into cookies pair if         
  client_cookies_redirects_test.go uses it, else base                         
                                                                              
  ## Gates                                                                    
                                                                              
  gofmt/build/vet/lint 0 issues./test -count=1; 270+2 parity; renames show    
  R100                                                                        
                                                                              
  ## Skills for coder                                                         
                                                                              
  ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules



## V3W2 execution notes (2026-10-15)
Grep matrix (users outside *_parts_test.go):
- all 8 oauth2State methods -> client_oauth2_test.go only ✓ (R100 rename, zero content change)
- aJsonEchoServer, aRestExtensionServer -> client_parsefile_test.go only → moved to base parts_test.go (with encoding/json import)
- remaining 8 cookiesState methods -> client_cookies_redirects_test.go only ✓
- aDotEnvFile/aDotEnvFileRemoved were already in env_parts_test.go (multi-file users: client_execute_vars/system_vars/parsefile) — untouched this wave per contract context
- withServerAddressVars: sole test-user client_cookies_redirects_test.go (4x) → moved from server_parts_test.go into client_cookies_redirects_parts_test.go (on *serverState receiver, single-user rule); net/url import moved with it
Gates: gofmt empty; build 0; vet 0; lint 0 issues.; go test ./... -count=1 ok (both packages); parity via go test -json: 300 top-level RUNs = 298 Test + 2 Fuzz (+13 fuzz seed sub-runs), 0 fails; HEAD-vs-worktree Test/Fuzz func sets identical (270 root Test + 28 CLI Test + 2 Fuzz).
Renames: git diff --cached -M100% --name-status → R100 oauth2_parts_test.go → client_oauth2_parts_test.go; R100 cookies_parts_test.go → client_cookies_redirects_parts_test.go. Status: renamed files + M server_parts_test.go + M parts_test.go + .beans only.

## Proof of Work (orchestrator-verified)
- R100 renames: oauth2_parts→client_oauth2_parts_test.go; cookies_parts→client_cookies_redirects_parts_test.go
- withServerAddressVars moved server pair → cookies pair (single-user); multi-file methods moved to base
- zero orphan methods in cookies pair (pair-usage grep verified); gates: gofmt/build/vet/lint 0, tests ok, 270+2
- Residual: user-flagged aRestExtensionServer/aJsonEchoServer still in base — V3W3 will place them in client_parsefile_parts_test.go (single-user rule)
