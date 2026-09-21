---
# go-restclient-jugo
title: OAuth2 password grant + useAuthorizationHeader=false
status: completed
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-21T20:24:20Z
parent: go-restclient-gqnq
blocked_by:
    - go-restclient-i43s
---

Parent epic: OAuth2 non-interactive grants. Blocked by client-credentials grant (shares token plumbing).

## Requirement
Password grant: `Authorization: oauth2 password <prefix>` with `{{prefix}}_username`/`_password`. Same token endpoint plumbing, cache, and error handling as client_credentials.

## Acceptance Criteria
- [x] grant_type=password form body asserted (username, password, client_id, client_secret)
- [x] `{{prefix}}_useAuthorizationHeader=false` puts client creds in body instead of Basic header
- [x] Cache + error paths covered by fluent tests
- [x] Docs
- [x] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)

## Proof of work

RED: 4 failing tests (password form body, useAuthorizationHeader=false/true
split, missing-password error) — RED commit on
feature/issue-fixes-45-49 before GREEN.
GREEN: password grant + useAuthorizationHeader, commit a0e6700; docs in
docs/http_syntax.md (OAuth2 Password Grant section).
Gates verified independently: 279 tests 0 failures, golangci-lint 0,
build/vet clean, gofmt clean.
Residual: epic go-restclient-gqnq closure pending PR merge + user review.
