---
# go-restclient-1sgh
title: 'S5: server domain + graphql fold → server_parts_test.go'
status: completed
type: task
priority: normal
created_at: 2026-10-15T11:20:03Z
updated_at: 2026-10-15T12:44:10Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-i1gt
---

go-restclient-1sgh  in-progress   task  [normal]       
S5: server domain + graphql fold → server_parts_test.go
──────────────────────────────────────────────────     
parent: go-restclient-495b                             
──────────────────────────────────────────────────     
                                                       

  ## Scope                                                                    
                                                                              
  NEW server_parts_test.go + serverState: aTLSServer, aTLSInsecureClient,     
  aCannedServer+cannedRoute+serveCannedRoute, anEchoServer,                   
  aMockTransportClient+mockRoundTripper, withServerAddressVars,               
  aGraphQLServer+graphqlResponse/graphqlError types; fields                   
  servers/serverURL/requestHits/activeServerVars; core aHttpServer moves here 
  (increments requestHits); DELETE fluent_parts_graphql_test.go.              
                                                                              
  ## Acceptance criteria                                                      
                                                                              
  [ ] serverState embedded; aHttpServer + graphql folded; graphql file deleted
  [ ] gates green, 270 tests, zero scenario diffs                             
                                                                              
  ## Triage (brainer completion plan 2026-10-15)                              
                                                                              
  Part of single-PR delivery (#84 → master, absorbs #83). Waves strictly      
  serial: every wave edits fluent_parts_test.go (struct embed + root wiring); 
  zero-scenario-diff gate needs stable base. Pattern to replicate = capture   
  pilot: anonymous embed, root *parts back-pointer, builders return p.root,   
  var                                                                         
  _ = []any{domainState{}} guard.                                             
                                                                              
  ## Gates (per wave)                                                         
                                                                              
  gofmt empty; build/vet 0; golangci-lint '0 issues.'; go test -count=1 green;
  Test inventory 270 no dupes; zero scenario-file diffs; CI watched to        
  terminal.                                                                   
                                                                              
  ## Skills for coder                                                         
                                                                              
  • ~/.agents/skills/go-dev                                                   
  • go-restclient-fluent-dsl skill (Hermes) — DSL layout table + hard rules   
  • docs/fluent-testing-guideline.md §6-12 (in-repo)                          
                                                                              
  ## Delegation                                                               
                                                                              
  pi contract (pi-delegation skill): GIT JAIL + -t +codemode + scout recon;   
  orchestrator re-gates + commits 'Refs: ' + pushes to reorg/parts-dsl-split.



## S5 execution report (coder, 2026-10-15)

**Scout list (server-domain builders):** ext: aTLSServer, aTLSInsecureClient, aCannedServer + cannedRoute + serveCannedRoute, anEchoServer, aMockTransportClient + mockRoundTripper, withServerAddressVars, serverReceivedFakerHeaders (+fakerHeaderRule type). Core fluent_parts_test.go: aHttpServer (increments requestHits). graphql file: aGraphQLServer + graphqlResponse/graphqlError.

**serverState fields:** root *parts, servers []*httptest.Server, serverURL string, requestHits atomic.Int64 (moved — requestCount reads it via promotion), activeServerVars map[string]any. Embedded anonymously in parts; `p.serverState.root = p` wired in newParts alongside all previous wires.

**Moved to serverState methods (byte-identical bodies, receiver transform + `p := s.root`):** aHttpServer (from core), aTLSServer, aTLSInsecureClient, aCannedServer, anEchoServer, withServerAddressVars, aMockTransportClient, aGraphQLServer; types cannedRoute, serveCannedRoute, graphqlResponse, graphqlError, mockRoundTripper + RoundTrip. fluent_parts_graphql_test.go deleted (0 test funcs in it).

**fakerHeaders decision:** serverReceivedFakerHeaders STAYS in ext — callers (client_execute_system_vars_test.go) assert faker sysvar content on captured requests, not server-domain concerns; fakerHeaderRule/urlHost stayed with it. mockRoundTripper moved with its builder per scope.

**Gates:** gofmt empty; build 0; vet 0; golangci-lint `0 issues.`; `go test ./... -count=1` ok both packages; root package 270 unique top-level tests (300 assertions incl. subtests, 0 fail), same as baseline. Byte-identity verified programmatically against HEAD blobs.

**git status:** only server_parts_test.go (A), fluent_parts_graphql_test.go (D), fluent_parts_ext_test.go (M), fluent_parts_test.go (M) + .beans/. Zero scenario-file diffs. Commit/push with orchestrator per delegation contract.

## Proof of Work (orchestrator-verified)
- serverState (servers, serverURL, requestHits, activeServerVars, mock types, cannedRoute/serveCannedRoute) embedded; server builders + core aHttpServer + graphql fold moved to *serverState; fluent_parts_graphql_test.go DELETED; fakerHeaders stays ext for S6 (scenario content assertions)
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory, zero scenario diffs
- Residual: none.
