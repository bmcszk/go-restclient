---
# go-restclient-dfzu
title: task V7W1 file splits + ctor swaps to primary domain ctors
status: completed
type: task
priority: normal
created_at: 2026-10-16T18:24:06Z
updated_at: 2026-10-16T18:33:10Z
---

## Proof of Work V7W1 (orchestrator-executed)
- audit: 9 test files mixed parts domains (builder-call census per file)
- splits: client_parse_test.go (10 parse-domain tests, newParseParts), client_env_test.go (14 env tests, newEnvParts), client_validate_test.go (5 hresp-extract tests, newValidatorParts), client_new_test.go (3 TestNewClient*), client_server_test.go (TLS test, newServerParts)
- ctor swaps to primary domain ctor in all 9 legacy files; all 8 domain ctors now used (lint clean)
- gates: gofmt EMPTY, build/vet 0, lint `0 issues.`, junit 315, Test-name parity vs 494ef81 EXACT, max 815 ln

## User rule enforced: ctors construct base
- newFixture(t) = core construction (parts_test.go); all 8 domain ctors + newParts call it, then init own domain state; zero `return newParts(t)` delegation left
- rule written into coding-rules skill (workers inherit): struct -> ctor -> methods order; ctor constructs base; one domain ctor per test file
- gates: gofmt EMPTY, vet 0, lint `0 issues.`, junit 315, parity EXACT


Residual: InPlace/MultipleRequests validator-assert tests intentionally stay in their feature files (assertion flavor, not domain ownership); delegation rule deviation: W1 moves executed orchestrator-side as mechanical file surgery, noted per 2-failure exception
