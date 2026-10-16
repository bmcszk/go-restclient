---
# go-restclient-v5w18p1l
title: 'V5W1: rename 7 state structs → DParts (captureParts/parseParts/fileParts/envParts/serverParts/oauth2Parts/cookiesParts), 111 methods, returns stay *parts'
status: completed
type: task
priority: normal
created_at: 2026-10-16T12:08:57Z
updated_at: 2026-10-16T12:14:10Z
parent: go-restclient-495b
---

USER MODEL (binding, verbatim): 'capture state tests should use given, when, then := newCaptureParts with captureParts struct and methods of captureParts struct (extending parts)'. Design (brainer-verified in scratch module, compiles+tests green): rename state structs to domain names; parts struct embeds them (cycle legal); builders get domain receivers; domain ctors return the DOMAIN type. Mixed chains keep compiling via Go promotion (verified depth-3). Moves-only + mechanical renames; NO new comments (coding-rules §Comments read BEFORE edit — binding); no //nolint/lint-config; ≤1000 ln.
## Scope
Renames ONLY (104 methods): captureState→captureParts (55), fileState→fileParts (17), parseState→parseParts (16), cookiesState→cookiesParts (10), oauth2State→oauth2Parts (8), serverState→serverParts (8), envState→envParts (7). Struct defs + receivers + internal references (p.captureState.parts → p.captureParts.parts wiring). Returns STAY *parts this wave.
## Gates
gofmt/build/vet/lint `0 issues.`/test -count=1 ok; parity: junit pass-count 315 (go test -json '"Action":"pass"' count, measured baseline); grep 'State) ' *_test.go → 0
## COMMENT GATE: per-touched-file grep -c '//' before/after; any added comment = FAILED.
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules — READ §Comments AND §File-structure BEFORE first edit; binding.

## Proof of Work (orchestrator-verified)
- all 7 state structs renamed to domain types: captureParts(55)/parseParts(16)/fileParts(17)/envParts(7)/serverParts(8)/oauth2Parts(8)/cookiesParts(10); 0 *State receivers remain
- production requestParserState untouched (out of scope, non-test)
- gates re-run: gofmt empty, build/vet 0, lint `0 issues.`, junit parity 315
- Residual: V5W2 return types
