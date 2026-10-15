---
# go-restclient-v3w460jk
title: 'V3W4: validator_parts_test.go trim to validator_test.go-only methods'
status: todo
type: task
priority: normal
created_at: 2026-10-15T15:55:30Z
updated_at: 2026-10-15T15:55:58Z
parent: go-restclient-495b
---

FULL REQUIREMENTS: same as epic v3. Moves-only; no test-body edits; no //nolint/lint-config.
## Scope
validator_parts_test.go: keep methods used by validator_test.go + validator_placeholders_test.go (both pair to validator_*? NO — placeholders pairs nothing; methods used by validator_placeholders_test.go must move THERE is impossible (test file can't hold methods) → they go to base parts_test.go; validator_parts_test.go keeps only validator_test.go-used methods). Verify with grep matrix.
## Gates
gofmt/build/vet/lint 0 issues./test -count=1; 270+2 parity
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
