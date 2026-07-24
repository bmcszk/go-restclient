---
# go-restclient-rexz
title: 'gh-29: --after should work with  bearer tokens'
status: scrapped
type: feature
priority: high
created_at: 2026-07-24T11:32:05Z
updated_at: 2026-07-24T12:10:50Z
parent: go-restclient-vidx
---

go-restclient-rexz  in-progress   feature  [high] 
gh-29: --after should work with  bearer tokens    
──────────────────────────────────────────────────
parent: go-restclient-vidx                        
──────────────────────────────────────────────────
                                                  

  ## Issue #29                                                                
                                                                              
  --after flag doesn't work well with dotenv-based auth files.                
                                                                              
  Tasks:                                                                      
                                                                              
  [ ] Support dotenv variables in prerequisite requests                       
  [ ] Allow chaining auth token from .env file                                
  [ ] Add integration test for OAuth flow

## Reasons for Scrapping

This is a documentation issue, not a code issue. The response chaining feature already works for this use case. Users should use response references instead of dotenv variables for chaining:

```http
### authorize
POST {{base_url}}/oauth2/token
...

### listRedeliveries
GET {{base_url}}/notification/redeliveries
Authorization: Bearer {{authorize.response.body.access_token}}
```

The dotenv variables are loaded from the `.env` file, not from response references. This is by design.

**Action:** Document the correct pattern in README.md
