---
# go-restclient-dfzu
title: task V7W1 file splits + ctor swaps to primary domain ctors
status: completed
type: task
priority: normal
created_at: 2026-10-16T18:24:06Z
updated_at: 2026-10-16T18:28:09Z
---

## Proof of Work V7W1 (orchestrator-executed)
- audit: 9 test files mixed parts domains (builder-call census per file)
- splits: client_parse_test.go (10 parse-domain tests, newParseParts), client_env_test.go (14 env tests, newEnvParts), client_validate_test.go (5 hresp-extract tests, newValidatorParts), client_new_test.go (3 TestNewClient*), client_server_test.go (TLS test, newServerParts)
- ctor swaps to primary domain ctor in all 9 legacy files; all 8 domain ctors now used (lint clean)
- gates: gofmt EMPTY, build/vet 0, lint `0 issues.`, junit 315, Test-name parity vs 494ef81 EXACT, max 815 ln
