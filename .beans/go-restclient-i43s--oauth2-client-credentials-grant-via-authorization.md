---
# go-restclient-i43s
title: 'OAuth2 client_credentials grant via Authorization: oauth2 header'
status: completed
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-21T20:24:20Z
parent: go-restclient-gqnq
---

Parent epic: OAuth2 non-interactive grants.

## Requirement
Client-credentials grant. Detection: request header `Authorization: oauth2 [client_credentials] <prefix>` (bare `oauth2 *** <prefix>` = client_credentials per httpyac default). Fetch token (form-encoded, grant_type=client_credentials), cache per prefix on client, apply as `Authorization: Bearer <token>`.

## Acceptance Criteria
- [ ] Happy path against canned token endpoint; correct form body + Content-Type asserted
- [ ] Cache: one token endpoint call even across multiple requests with same prefix (assert via requestCount)
- [ ] Missing prefix variables -> clear error naming the missing var
- [ ] Token endpoint 4xx/5xx -> error naming prefix + status
- [ ] Fluent tests + docs
- [ ] - [ ] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)

## PoW (2026-09-21)
- RED 29547b6: 5 fluent tests, 4 failing for the right reason (stub "token fetch not implemented"), independently re-run: failures="4"
- GREEN e621bf2: lookupVar reuse via directiveResolveContext, form-encoded POST via c.httpClient, 4k-bounded error body, per-prefix mutex cache; 264 tests 0 failures, golangci-lint 0 (independently re-run)
- PR #39 OPEN (feature/oauth2-grants), CI all-pass, head e621bf2b5d8738afe6e90d343f0db102f038869e
