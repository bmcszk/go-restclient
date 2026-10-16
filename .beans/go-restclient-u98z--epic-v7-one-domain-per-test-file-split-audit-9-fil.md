---
# go-restclient-u98z
title: 'epic V7 one-domain-per-test-file split (audit: 9 files mix parts domains)'
status: completed
type: task
priority: normal
created_at: 2026-10-16T18:24:06Z
updated_at: 2026-10-16T18:28:09Z
---

Plan /tmp/plan-v7.md: new client_parse_test.go (newParseParts), client_env_test.go (newEnvParts), client_new_test.go (TestNewClient*), client_server_test.go (newServerParts+TLS), client_validate_test.go (15 validator tests); ctor swaps to domain ctors everywhere; parity 270, junit 315.
