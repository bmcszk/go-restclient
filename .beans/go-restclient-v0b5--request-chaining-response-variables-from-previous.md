---
# go-restclient-v0b5
title: 'Request chaining: response variables from previous requests'
status: completed
type: feature
priority: high
created_at: 2026-07-23T15:58:07Z
updated_at: 2026-07-23T16:36:04Z
---

go-restclient-v0b5  todo   feature  [high]                 
Request chaining: response variables from previous requests
──────────────────────────────────────────────────         
                                                           

  go-restclient-v0b5  todo   feature  [high]               Request chaining:  
  response variables from previous requests                                   
  ──────────────────────────────────────────────────                          
                                                                              
  ## Problem                                                                  
                                                                              
  Users need auth request then use access_token from response in subsequent   
  requests.                                                                   
                                                                              
  ## Research: How Both Clients Do It                                         
                                                                              
  ### VS Code REST Client — Request Variables (no scripts)                    
                                                                              
  Named requests with # @name login. Responses auto-stored. Reference:        
  {{login.response.body.token}} — JSON property                               
  {{login.response.body.data[0].id}} — array index {{login.response.headers.X-
  Custom}} — header {{login.response.status}} — status code No scripting      
  required.                                                                   
                                                                              
  ### JetBrains — Response Handler Scripts                                    
                                                                              
  | {% client.global.set("auth_token", response.body.access_token); %} Then   
  use                                                                       | 
  {{auth_token}} in subsequent requests. Requires JS.                         
                                                                              
  ## Proposed: VS Code-style implicit response references (Phase 1)           
                                                                              
  After each request executes, store response keyed by name. Before subsequent
  requests, inject {{name.response.body.X}} etc. No scripts needed. Covers 90%
  of use cases.                                                               
                                                                              
  ## Beans                                                                    
                                                                              
  [ ] Store response map in ParsedFile after execution                      [ 
  ] Add response reference regex patterns to variables.go                 [ ] 
  Implement JSON path resolution (dot + array index)                    [ ]   
  Inject response references before variable substitution               [ ]   
  Unit tests for response variable resolution                           [ ]   
  Integration test: auth flow with token chaining                       [ ]   
  Update docs/http_syntax.md                                            [ ]   
  make check passes                                                           
                                                                              
  ## Sub-tasks                                                                
                                                                              
  [ ] go-restclient-xxxx: Store response map in ParsedFile                    
  [ ] go-restclient-xxxx: Response reference regex + JSON path resolution     
  [ ] go-restclient-xxxx: Inject response refs in ExecuteFile loop            
  [ ] go-restclient-xxxx: Unit tests                                          
  [ ] go-restclient-xxxx: Integration test: auth token flow                   
  [ ] go-restclient-xxxx: Update docs

## Summary of Changes\n\n### New file: `response_refs.go`\n- Response reference regex pattern\n- `resolveResponseReference()` — resolves `{{name.response.body.X}}`, `{{name.response.headers.X}}`, `{{name.response.status}}`\n- `resolveResponseBody()`, `resolveResponseHeader()`\n- JSON path resolution: `resolveJSONPath()`, `parseJSONPathSegments()`, `walkJSONPath()`\n\n### Modified: `request.go`\n- Added `ResponseMap map[string]*Response` to `ParsedFile`\n\n### Modified: `client.go`\n- Initialize `ResponseMap` in `ParseFile` and `ExecuteFile`\n- Store responses after each execution via `storeResponse()`\n- Thread `responseMap` through variable substitution chain\n\n### Modified: `variables.go`\n- Added `resolveContext` struct (replaces 9-parameter function)\n- Response reference check in `resolveVariablePlaceholder()`\n\n### New file: `test/response_refs.go`\n- 7 tests: status code, body field, header value, nested body, array index, missing name, whole body\n\n### Updated: `docs/http_syntax.md`\n- Full syntax table and examples for response references\n\n### Usage\n```http\n### login\nPOST https://api.example.com/auth/login\nContent-Type: application/json\n\n{\"username\":\"admin\",\"password\":\"secret\"}\n\n### protected\nGET https://api.example.com/protected\nAuthorization: Bearer {{login.response.body.access_token}}\n```
