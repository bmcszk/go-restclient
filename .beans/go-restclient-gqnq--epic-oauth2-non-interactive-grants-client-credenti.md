---
# go-restclient-gqnq
title: 'Epic: OAuth2 non-interactive grants (client credentials + password)'
status: in-progress
type: feature
priority: normal
tags:
    - epic
    - httpyac-parity
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T14:16:10Z
---

## Goal

Built-in non-interactive OAuth2 token acquisition so E2E suites against real APIs need no hand-rolled token choreography.

## Triage / Context

- httpyac (docs 2026-09-19, https://httpyac.github.io/guide/variables.html): `Authorization: oauth2 *** <prefix>` triggers token fetch using `{{prefix}}_tokenEndpoint`, `_clientId`, `_clientSecret`, etc. Supports auth-code/PKCE/implicit/password/client-credentials/device-code.
- We have ZERO OAuth support (`$aadToken` regex exists in variables.go but is never substituted - recognized as dynamic placeholder only).
- Scope: **client_credentials + password grant only** - headless E2E cannot do browser flows. Keep-alive renewal and token exchange explicitly out of scope.

## Acceptance Criteria

- [ ] `Authorization: oauth2 client_credentials <prefix>` fetches token from `{{prefix}}_tokenEndpoint` with `{{prefix}}_clientId`/`_clientSecret` (form-encoded body)
- [ ] Token cached per prefix for client lifetime; header set as `Bearer <token>` on the request
- [ ] Password grant variant via `{{prefix}}_username`/`_password`
- [ ] `useAuthorizationHeader=false` sends client creds in the POST body (per RFC 6749)
- [ ] Token endpoint failure -> clear error naming the prefix
- [ ] Fluent tests with canned token endpoint + docs section in http_syntax.md
