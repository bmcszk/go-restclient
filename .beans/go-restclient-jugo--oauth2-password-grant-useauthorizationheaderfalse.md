---
# go-restclient-jugo
title: OAuth2 password grant + useAuthorizationHeader=false
status: todo
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T14:16:10Z
parent: go-restclient-gqnq
blocked_by:
    - go-restclient-i43s
---

Parent epic: OAuth2 non-interactive grants. Blocked by client-credentials grant (shares token plumbing).

## Requirement
Password grant: `Authorization: oauth2 password <prefix>` with `{{prefix}}_username`/`_password`. Same token endpoint plumbing, cache, and error handling as client_credentials.

## Acceptance Criteria
- [ ] grant_type=password form body asserted (username, password, client_id, client_secret)
- [ ] `{{prefix}}_useAuthorizationHeader=false` puts client creds in body instead of Basic header
- [ ] Cache + error paths covered by fluent tests
- [ ] Docs
- [ ] - [ ] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)
