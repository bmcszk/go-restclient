---
# go-restclient-wlqn
title: 'S6: oauth2+cookies+validator rename+restructure'
status: completed
type: task
priority: normal
created_at: 2026-10-15T11:20:03Z
updated_at: 2026-10-15T13:31:29Z
parent: go-restclient-495b
blocked_by:
    - go-restclient-1sgh
---

## Triage (brainer endgame re-plan 2026-10-15, disk-verified)
3 leaves: (a) fold fluent_parts_oauth2_test.go(88) into client_oauth2_test.go → 376ln/12 tests; (b) fold fluent_parts_cookies_redirects_test.go(107) into client_cookies_redirects_test.go → 186ln/4 tests; (c) rename fluent_parts_validator_test.go → validator_parts_test.go + validatorState restructure (cross-cutting builders used by 4-6 files).
## Acceptance criteria
- [ ] oauth2 file folded, cookies file folded; fluent_parts_oauth2/cookies_redirects GONE
- [ ] validator_parts_test.go + validatorState; chains byte-identical
- [ ] gates green, 270 tests, zero scenario diffs
## Gates
gofmt/build/vet 0; lint `0 issues.`; go test -count=1; inventory 270; moves-only; CI terminal on #84.
## Skills for coder
- ~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
## Delegation
pi contract per pi-delegation skill.

## Proof of Work (orchestrator-verified)
- oauth2State (tokenServerURL) embedded; 8 oauth2 builders folded into client_oauth2_test.go; fluent_parts_oauth2_test.go DELETED (wave 1 of split S6)
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory
- Residual: S6b (cookies fold + validator rename) next.

## Proof of Work (orchestrator-verified)
- S6b: cookiesState (root + cookieCheck) embedded; cookies builders + aCookieRedirectFixture folded into client_cookies_redirects_test.go; fluent_parts_cookies_redirects_test.go DELETED; validator file renamed validator_parts_test.go (rename-only, builders cross-cutting on base fields)
- S6 complete (a+b): gates re-run by orchestrator — gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, 270 inventory
- Residual: none. fluent_* remaining: ext + core (S7 scope).
