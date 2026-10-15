---
# go-restclient-1f36
title: 'S8: docs — fluent-testing-guideline.md + skill table'
status: todo
type: task
priority: normal
created_at: 2026-10-15T11:20:04Z
updated_at: 2026-10-15T12:35:22Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-euu1
    - go-restclient-9ysx
    - go-restclient-awnn
    - go-restclient-4r1g
    - go-restclient-7a9s
    - go-restclient-4m54
    - go-restclient-zicv
    - go-restclient-2ih6
---

## Triage (brainer endgame re-plan 2026-10-15, disk-verified)
docs/fluent-testing-guideline.md: DSL-split paragraph rewrite (domainState + root back-pointer), fix stale e2e_parts refs (~ln1025/1272), file-map table to FINAL 21-file layout; Hermes go-restclient-fluent-dsl skill layout table updated same session. Endgame audit: ls fluent_* empty; *_test.go count 21; max file 832 < 1000; inventory 270; CI green; epic PoW = audit transcript.
## Acceptance criteria
- [ ] docs + skill table match final layout
- [ ] endgame audit all green → epic ready to close on PR merge
## Gates
gofmt/build/vet 0; lint `0 issues.`; go test -count=1; inventory 270; moves-only; CI terminal on #84.
## Skills for coder
- ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
## Delegation
pi contract per pi-delegation skill.
Blocked-by: S6, S7, C1-C7
