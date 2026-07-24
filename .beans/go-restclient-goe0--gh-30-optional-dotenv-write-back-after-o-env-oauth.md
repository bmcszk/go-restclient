---
# go-restclient-goe0
title: 'gh-30: Optional dotenv write-back after -o env / OAuth prerequisites'
status: scrapped
type: feature
priority: normal
created_at: 2026-07-24T11:32:30Z
updated_at: 2026-07-24T12:11:14Z
parent: go-restclient-vidx
---

go-restclient-goe0  todo   feature  [normal]                        
gh-30: Optional dotenv write-back after -o env / OAuth prerequisites
──────────────────────────────────────────────────                  
parent: go-restclient-vidx                                          
──────────────────────────────────────────────────                  
                                                                    

  ## Issue #30                                                                
                                                                              
  Token persistence still needs custom bash script.                           
                                                                              
  Tasks:                                                                      
                                                                              
  [ ] Add dotenv write-back support                                           
  [ ] Allow saving OAuth tokens to .env                                       
  [ ] Document workflow for token persistence

## Reasons for Scrapping

YAGNI. The user can already use response chaining to pass tokens between requests. The dotenv write-back is a convenience feature that can be added later if needed.

The existing workflow is:
1. Use response chaining to pass tokens between requests
2. Use `-o env` to extract values for scripting

Example:
```http
### authorize
POST https://api.example.com/oauth2/token
Content-Type: application/json

{"client_id": "xxx", "client_secret": "yyy"}

### listRedeliveries
GET https://api.example.com/notification/redeliveries
Authorization: Bearer {{authorize.response.body.access_token}}
```

Or for scripting:
```bash
restclient -f redeliveries.http -n authorize -o env access_token
```
