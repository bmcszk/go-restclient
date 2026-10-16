---
# go-restclient-v5w3i0lu
title: 'V5W3: 4 new constructors (newCaptureParts/newParseParts/newOauth2Parts/newCookiesParts) + swap 174 newParts(t) call lines to domain ctors'
status: todo
type: task
priority: normal
created_at: 2026-10-16T12:08:57Z
updated_at: 2026-10-16T12:09:21Z
parent: go-restclient-495b
---

USER MODEL: 'capture state tests should use given, when, then := newCaptureParts'. Add 4 constructors (only uncalled ones fail unused lint): newCaptureParts/newParseParts/newOauth2Parts/newCookiesParts(t) (given, when, then *DParts) { p,_,_ := newParts(t); d := &p.dparts; return d,d,d }. Then swap call sites: client_execute_control(42)+core(29)+sysvars(27)+vars(27)+refs(18)+graphql(7) → newCaptureParts(t); client_parsefile(27) → newParseParts(t); client_oauth2(12) → newOauth2Parts(t); client_cookies_redirects(4) → newCookiesParts(t). validator/validator_placeholders KEEP newParts (pure core, no domain calls — measured). These ARE the user-mandated test-body edits (construction line only; chains untouched).
## Gates
gofmt/build/vet/lint `0 issues.`/test ok; junit 315; grep 'newParts(t)' outside validator*/parts_test.go → 0
## COMMENT GATE: per-file comment diff; added comment = FAILED.
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules — read §Comments/§File-structure first; binding.
