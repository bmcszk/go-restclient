---
# go-restclient-v3w5x23s
title: 'V3W5: v3 audit — every X_parts_test.go pairs X_test.go + contains only its pair-used decls; parity; PR body'
status: completed
type: task
priority: normal
created_at: 2026-10-15T15:55:30Z
updated_at: 2026-10-15T16:40:52Z
parent: go-restclient-495b
---

FULL REQUIREMENTS: same as epic v3. Orchestrator-run script audit.
## Scope
Script: (1) every *_parts_test.go basename X_parts_test.go has existing X_test.go; (2) each parts file's decls used ONLY by its pair (+base); (3) zero `root *parts`; (4) `*parts` first-line embedding in every state struct; (5) 270+2 parity, ≤1000 ln; (6) PR #84 body v3 update.
## Gates
all PASS with verbatim script output; PR body updated

## Proof of Work (orchestrator-run final audit, verbatim)
1. ALL parts files paired: 10 pairs (client_oauth2/client_cookies_redirects/client_execute_{control,core,graphql,refs,system_vars,vars}/client_parsefile/validator) + base parts_test.go
2. `root *parts` refs: 0; every state struct embeds `*parts` as FIRST line (cookiesState in its pair; capture/parse/file/env/server states in base — multi-pair usage)
3. rule-3 independent audit: violations NONE; single-user methods in base: NONE (user-flagged aRestExtensionServer/aJsonEchoServer in client_parsefile_parts_test.go)
4. ≤1000 lines: all (max base 835); parity 270 Test + 2 Fuzz; 24 test files = 11 parts + 13 tests (export/fuzz sanctioned)
5. V3W4: validator parts all 12 methods validator_test-only — already compliant
- Residual: none
