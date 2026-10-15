---
# go-restclient-v3w2pzjp
title: 'V3W2: rename oauth2_parts/cookies_parts to client_oauth2_parts_test.go/client_cookies_redirects_parts_test.go (+2 PF methods to parsefile pair)'
status: todo
type: task
priority: normal
created_at: 2026-10-15T15:55:30Z
updated_at: 2026-10-15T15:55:58Z
parent: go-restclient-495b
---

FULL REQUIREMENTS: same as epic v3 (3-file pairs, parts file ONLY serves its pair, anonymous embedding already done in V3W1). Moves-only; no test-body edits; no //nolint/lint-config; ≤1000 ln.
## Scope
- git mv oauth2_parts_test.go → client_oauth2_parts_test.go (pairs client_oauth2_test.go; impl client_oauth2.go)
- git mv cookies_parts_test.go → client_cookies_redirects_parts_test.go (pairs client_cookies_redirects_test.go); move its 2 parsefile-used methods (aHttpFile-family? verify by grep which 2 methods client_parsefile_test.go uses) OUT into a temp home (they go to client_parsefile_parts_test.go in V3W3 — if V3W3 runs after, put them in parts_test.go temporarily? NO: do V3W3's parsefile-pair creation is separate wave; here just move the 2 methods into client_parsefile_test.go-adjacent... brainer plan says cookies pair −2 PF methods +withServerAddressVars; verify with grep what parsefile actually uses from cookies file and move those methods to parts_test.go base temporarily if their final home doesn't exist yet)
- withServerAddressVars (from server domain) → into cookies pair if client_cookies_redirects_test.go uses it, else base
## Gates
gofmt/build/vet/lint 0 issues./test -count=1; 270+2 parity; renames show R100
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
