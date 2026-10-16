---
# go-restclient-v4w3zxob
title: 'V4W3: server/cookies/oauth2 parts — serverState struct+all methods (withServerAddressVars home), cookies+oauth2 renames to state names, pull strays back'
status: todo
type: task
priority: normal
created_at: 2026-10-16T11:22:08Z
updated_at: 2026-10-16T11:22:08Z
parent: go-restclient-495b
---

FULL USER REQUIREMENTS: struct+methods same file; withServerAddressVars MUST live with serverState; cookiesState methods out of parsefile pair; receiver rule oracle: git show 39000b5:{server,cookies,oauth2}_parts_test.go. Moves-only, byte-identical, no new comments, no //nolint/lint-config, ≤1000 ln.
## Scope
server_parts_test.go: serverState struct+8 methods+cannedRoute+serveCannedRoute+mockRoundTripper/RoundTrip+graphqlResponse/graphqlError+aGraphQLServer+withServerAddressVars (from cookies pair). git mv client_cookies_redirects_parts_test.go→cookies_parts_test.go (pull its 2 methods back from parsefile pair); git mv client_oauth2_parts_test.go→oauth2_parts_test.go.
## Gates
gofmt/build/vet/lint 0 issues./test -count=1; 270+2; serverState methods in ONE file
## COMMENT GATE: comment diff per touched file; added narration = FAILED.
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules — read §Comments/§File-structure first; binding.
