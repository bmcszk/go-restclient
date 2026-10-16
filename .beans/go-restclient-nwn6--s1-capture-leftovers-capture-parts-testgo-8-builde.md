---
# go-restclient-nwn6
title: 'S1: capture leftovers → capture_parts_test.go (~8 builders)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T11:20:03Z
updated_at: 2026-10-15T11:31:15Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-75ci
---

go-restclient-nwn6  in-progress   task  [normal]           
S1: capture leftovers → capture_parts_test.go (~8 builders)
──────────────────────────────────────────────────         
parent: go-restclient-495b                                 
──────────────────────────────────────────────────         
                                                           

  ## Scope                                                                    
                                                                              
  Source: fluent_parts_ext_test.go. Move ~8 capture-family builders           
  (serverReceivedMethodAndPath/JSONBody/HeaderValue, requestRawURLContains +  
  kin) to captureState in capture_parts_test.go.                              
                                                                              
  ## Acceptance criteria                                                      
                                                                              
  [ ] builders live on *captureState, return *parts                           
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



## S1 execution (2026-10-15, pi contract)
- Scout: subagent launch failed (host pkg missing @earendil-works/chord deps); recon done in-session instead.
- Moved 9 builders ext→captureState (block ext L656–738 was contiguous): clientRequestInterceptorCaptures, clientSentNoHeaders, serverReceivedMethodAndPath, serverReceivedJSONBody, serverReceivedHeaderValue, requestRawURLContains, capturedRequestCount, requestSkipped, requestExecuted. Method-only transform: receiver→*captureState, +`p := c.root`; bodies/docs byte-identical; zero scenario files touched.
- requestSkipped/requestExecuted read p.responses (record of executions) → kept in capture file per precedent (allResponseCodesAre already there); capture state owns recorded requests+bodies+execution record, parse reads ParsedFile.
- Imports: ext keeps io/json (server helpers still use them); capture file gained io. No guard edits needed (unexported type embedded, no unused imports).
- Gates: gofmt empty; go build 0; go vet 0; golangci-lint `0 issues.`; go test ./... -count=1 all ok (pkg 0.605s, cli 24.9s); Test inventory 270; porcelain = only capture_parts_test.go (M), fluent_parts_ext_test.go (M), .beans/ untracked.
- Note: 2 orphaned doc-comment fragments in capture_parts_test.go (serverReceivedHeaderNotContains/FieldCountIs) predate the pilot — dead-method leftovers from PR #35 round 3; no lint impact, left untouched (minimal diff).

## Proof of Work (orchestrator-verified)
- pi wave S1 moved 9 builders (serverReceivedMethodAndPath/JSONBody/HeaderValue, serverReceivedLatin1BodyIs, requestRawURLContains+kin, requestSkipped, requestExecuted) to *captureState, returning p.root; serverReceivedFakerHeaders stays ext (scenario-coupled)
- gates re-run by orchestrator: gofmt empty, build/vet 0, golangci-lint `0 issues.`, go test -count=1 both pkgs ok, Test inventory 270, zero scenario-file diffs (git status: only capture_parts + ext + .beans)
- Residual: none.
