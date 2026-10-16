---
# go-restclient-u98z
title: 'epic V7 one-domain-per-test-file split (audit: 9 files mix parts domains)'
status: completed
type: task
priority: normal
created_at: 2026-10-16T18:24:06Z
updated_at: 2026-10-16T18:33:38Z
---

Plan /tmp/plan-v7.md: new client_parse_test.go (newParseParts), client_env_test.go (newEnvParts), client_new_test.go (TestNewClient*), client_server_test.go (newServerParts+TLS), client_validate_test.go (15 validator tests); ctor swaps to domain ctors everywhere; parity 270, junit 315.


Residual: cross-domain builder calls remain by design (setup file/server + promoted assertions); one-domain rule holds at ctor level

## Proof of Work (epic closure)
- Full suite: go test ./... -count=1 → junit 315 pass; build/vet 0; golangci-lint `0 issues.`; gofmt EMPTY
- Test parity: root Test-name set vs 494ef81 → EXACT (270)
- One-domain-per-file audit: builder-call census per test file; splits shipped (client_parse/env/validate/new/server_test.go); all 8 domain ctors used; ctor order struct→ctor→methods in every *_parts_test.go
- Ctor rule: newFixture(t) constructs base; domain ctors construct, zero newParts-delegation
- CI: PR #84 checks all pass (test 1.21–1.24, CodeQL, gosec, govulncheck, Scorecard), MERGEABLE
- PR: https://github.com/bmcszk/go-restclient/pull/84
Residual: cross-domain builder calls inside test bodies remain by design via fixture promotion (setup in file/server domains, assertions in primary domain).
