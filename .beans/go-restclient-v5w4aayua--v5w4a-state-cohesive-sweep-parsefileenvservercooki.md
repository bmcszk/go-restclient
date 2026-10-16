---
# go-restclient-v5w4aayua
title: 'V5W4a: state-cohesive sweep — parse/file/env/server/cookies/oauth2 struct+all methods into their own parts files'
status: completed
type: task
priority: normal
created_at: 2026-10-16T13:20:16Z
updated_at: 2026-10-16T14:32:32Z
parent: go-restclient-495b
---

## Proof of Work (orchestrator-verified)
- parseParts struct + 16 methods co-located in parse_parts_test.go (NEW); drained from base + control/refs/parsefile pairs; control+refs pair files held ONLY parseParts → deleted (empty)
- gates re-run: gofmt empty, build/vet 0, lint `0 issues.`, junit 315
- Residual: fileParts, envParts, serverParts, cookies/oauth2 renames (next per-state waves)

## Proof of Work (orchestrator-verified)
- fileParts struct + all 17 methods co-located in file_parts_test.go (NEW); drained from base + core/vars/parsefile pairs
- gates re-run: gofmt empty, build/vet 0, lint `0 issues.`, junit 315
- Residual: envParts, serverParts, cookies/oauth2 renames

## Proof of Work (orchestrator-verified, incl. orchestrator fix-ups)
- envParts struct + 7 methods → env_parts_test.go; serverParts struct + 8 methods + cannedRoute/serveCannedRoute/mockRoundTripper/RoundTrip/graphql types → server_parts_test.go; graphql pair file emptied+deleted
- orchestrator correction: pi parked cookiesState→cookiesParts methods (aJsonEchoServer/aRestExtensionServer) in env_parts — moved to client_cookies_redirects_parts_test.go (cookies pair) + import fixes
- gates re-run: gofmt empty, build/vet 0, lint `0 issues.`, junit 315
- Residual: cookies/oauth2 pair renames to state names

## Proof of Work (orchestrator-verified)
- R100 renames: client_cookies_redirects_parts_test.go → cookies_parts_test.go; client_oauth2_parts_test.go → oauth2_parts_test.go
- gates re-run: gofmt empty, lint `0 issues.`, junit 315
- state-cohesive layout COMPLETE: capture/parse/file/env/server/cookies/oauth2 parts files each hold struct+ctor+all methods
- Residual: V5W4b comment sweep + final audit

## Parity re-verification (orchestrator)
- pi's '313 vs 315' = miscount (cmd/restclient main_test has its own CLI newParts; root junit 'Action:pass' includes subtests+fuzz seeds)
- DEFINITIVE: root-package Test-name set diff vs pre-V5 commit 494ef81 → ROOT-PARITY-EXACT (270 = 270, zero names changed)
