---
# go-restclient-hs41
title: task V9W1 rename test files to domain stems (8 renames), pairs exact
status: completed
type: task
priority: normal
created_at: 2026-10-16T19:26:26Z
updated_at: 2026-10-16T19:27:52Z
parent: go-restclient-nvdz
---

## Proof of Work V9W1 (orchestrator, mechanical renames)
- 8 renames to domain stems: capture/env/file/parse/validator/oauth2/cookies/server_test.go — exact-stem pair with each X_parts_test.go
- client_validate_test.go merged into validator_test.go (same domain, 5 hresp tests); client_new_test.go merged into parts_test.go (base)
- gates: gofmt EMPTY, vet 0, lint `0 issues.`, junit 315, parity EXACT
Residual: feature files inside a domain (client_execute_control/graphql/refs/vars/system_vars) intentionally share the domain parts file; no pair of their own.
