---
# go-restclient-tcas
title: Add ParseFile, request selection, and .hresp assertions to CLI
status: completed
type: feature
priority: normal
created_at: 2026-07-23T15:12:29Z
updated_at: 2026-07-23T15:32:26Z
---

go-restclient-tcas  in-progress   feature  [normal]           
Add ParseFile, request selection, and .hresp assertions to CLI
──────────────────────────────────────────────────            
                                                              

  go-restclient-tcas  in-progress   feature  [normal]         Add ParseFile,  
  request selection, and .hresp assertions to CLI                             
  ──────────────────────────────────────────────────                          
                                                                              
  ## Requirements                                                             
                                                                              
  1. Library: Expose ParseFile (parse without execute) so CLI can inspect     
  requests                                                                    
  2. CLI flags: -n  and -i  to run only one request                           
  3. CLI flag: -e <expected.hresp> for asserting responses                    
  4. Exit code: non-zero if any assertion fails                               
                                                                              
  ## Beans                                                                    
                                                                              
  [ ] Add ParseFile method to library                                       [ 
  ] Add -n, -i, -e flags to CLI                                           [ ] 
  Add                                                                         
  unit tests for ParseFile                                          [ ] Add   
  CLI integration tests                                             [ ] make  
  check passes                                                                
                                                                              
  ## Done                                                                     
                                                                              
  [x] Add ParseFile method to library                                         
  [x] Add ExecuteRequest method to library                                    
  [x] Add createTestFileFromString test helper                                
  [x] 9 unit tests passing

## Done
- [x] Add `ParseFile` method to library
- [x] Add `ExecuteRequest` method to library
- [x] Add `-n`, `-i`, `-e` flags to CLI
- [x] Add unit tests for `ParseFile` and `ExecuteRequest`
- [x] Add CLI integration tests (12 tests)
- [x] `make check` passes (213 tests, 0 lint issues)

## Summary of Changes

**Library** (`client.go`):
- `ParseFile(path) (*ParsedFile, error)` — parse without execute
- `ExecuteRequest(ctx, parsedFile, index) (*Response, error)` — run a single request by index
- Renamed private `executeRequest` → `doHTTPRequest` to avoid naming confusion

**CLI** (`cmd/restclient/main.go`):
- `-n <name>` — run only the named request (case-insensitive)
- `-i <index>` — run only the request at 0-based index
- `-e <expected.hresp>` — assert responses against expected-response file
- `-n` and `-i` are mutually exclusive
- Exit codes: 0 success, 1 error/assertion failure, 2 usage error

**Tests**:
- 9 unit tests: ParseFile (5) + ExecuteRequest (4)
- 12 CLI integration tests: flag parsing, name/index selection, assertions
- Total: 213 tests, 0 lint issues
