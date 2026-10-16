---
# go-restclient-v2w3zhiq
title: 'V2W3: 6 client-config builders from client_execute_core_test.go into parts_test.go (base)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:23:12Z
parent: go-restclient-495b
---

go-restclient-v2w3zhiq  in-progress   task  [normal]                                     
V2W3: 6 client-config builders from client_execute_core_test.go into parts_test.go (base)
──────────────────────────────────────────────────                                       
parent: go-restclient-495b                                                               
──────────────────────────────────────────────────                                       
                                                                                         

  Triage: brainer v2 plan 2026-10-15. FULL requirement context: clear         
  separation, base parts struct = most basic builders (REQ2), moves-only, no  
  test-body edits, no //nolint/lint-config.                                   
                                                                              
  ## Scope                                                                    
                                                                              
  Move 6 client-config builder methods (aClientWithDefaults, clientBaseURLIs, 
  clientBaseURLIsEmpty, clientDefaultHeaderIs, clientDefaultHeadersEmpty,     
  clientExists; client_execute_core_test.go L717-769) into parts_test.go      
  (base). client_execute_core_test.go → ~751ln/29 tests, parts_test.go ~303ln.
                                                                              
  ## Acceptance                                                               
                                                                              
  • 6 builders on p *parts live in parts_test.go; core file zero client-config
  builders; 298 parity; gates green; both files <1000                         
                                                                              
  ## Skills for coder                                                         
                                                                              
  ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules



## V2W3 execution report (2026-10-15)
Moved verbatim (byte-identical) 6 client-config builder methods on p *parts from client_execute_core_test.go (L717–769) into parts_test.go (appended after responseTLSVersionMatches, source order kept): aClientWithDefaults, clientBaseURLIs, clientBaseURLIsEmpty, clientDefaultHeaderIs, clientExists, clientDefaultHeadersEmpty.
Imports recomputed: dropped "time" from client_execute_core_test.go; added "net/http" to parts_test.go.
No test bodies touched; zero nolint; no lint-config edits; files 749 / 303 lines (≤1000).
Gates: gofmt clean; go build ./... 0 errors; go vet 0; golangci-lint "0 issues."; go test ./... -count=1 ok (root + cmd/restclient); inventory 270 Test + 2 Fuzz in root package.
git status: M client_execute_core_test.go, M parts_test.go, M .beans/ (expected).

## Proof of Work (orchestrator-verified)
- 6 client-config builders moved into parts_test.go (base, 26 methods now); client_execute_core_test.go 29 tests, 0 parts methods
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, inventory 270+2Fuzz
- Residual: none.
