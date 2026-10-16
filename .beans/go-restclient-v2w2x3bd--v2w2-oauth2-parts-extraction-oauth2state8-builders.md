---
# go-restclient-v2w2x3bd
title: 'V2W2: oauth2 parts extraction — oauth2State+8 builders out of client_oauth2_test.go into NEW oauth2_parts_test.go'
status: completed
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:18:46Z
parent: go-restclient-495b
---

go-restclient-v2w2x3bd  in-progress   task  [normal]                                                             
V2W2: oauth2 parts extraction — oauth2State+8 builders out of client_oauth2_test.go into NEW oauth2_parts_test.go
──────────────────────────────────────────────────                                                               
parent: go-restclient-495b                                                                                       
──────────────────────────────────────────────────                                                               
                                                                                                                 

  Triage: brainer v2 plan 2026-10-15. FULL requirement context: clear *_test. 
  go                                                                          
  (tests only) vs *_parts_test.go (parts only) separation; ≤1000 ln; moves-   
  only; no test-body edits; no //nolint; no lint-config changes.              
                                                                              
  ## Scope                                                                    
                                                                              
  Move oauth2State + 8 oauth2 builder methods (client_oauth2_test.go L14-114) 
  into NEW oauth2_parts_test.go. 12 Test funcs stay (~293ln). Imports         
  recomputed. oauth2State = ONLY oauth2 fields+methods (REQ2).                
                                                                              
  ## Acceptance                                                               
                                                                              
  • oauth2_parts_test.go NEW zero Test funcs; client_oauth2_test.go zero parts
  decls; 298 parity; gates green                                              
                                                                              
  ## Skills for coder                                                         
                                                                              
  ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules


## V2W2 result
Moved decls: `type oauth2State`, its `var _ = []any{...}` assertion, 8 builder methods (`anOAuth2TokenServerWithStatus`, `aClientWithOAuth2Vars`, `aClientWithOAuth2ProgrammaticVars`, `clientWithProgrammaticVars`, `aClientWithOAuth2PasswordVars`, `aClientWithOAuth2PasswordVarsAuthHeaderUnset`, `aClientWithOAuth2PasswordVarsMissingPassword`, `aClientWithOAuth2VarsMissingSecret`) → `oauth2_parts_test.go` (package `restclient_test`, mirrors cookies pattern, receivers/root back-pointer unchanged). `client_oauth2_test.go` = 12 Test funcs (288 ln, imports recomputed: base64/fmt/net/http/testing; `rc` import dropped — only builders used it).

Gates: gofmt -l empty · go build 0 · go vet 0 · golangci-lint `0 issues.` · `go test ./... -count=1` ok (root 0.555s, cmd 16.566s) · inventory 270 Test + 2 Fuzz. git status: client_oauth2_test.go (M), oauth2_parts_test.go (new), .beans/ (M). Parity: moves-only, zero test-body edits, files ≤1000 ln (288/110).

## Proof of Work (orchestrator-verified)
- oauth2State + 8 builders moved to NEW oauth2_parts_test.go (110ln, 0 Test funcs); client_oauth2_test.go now 12 tests only (0 parts decls, 288ln)
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, inventory 270+2Fuzz
- Residual: none.
