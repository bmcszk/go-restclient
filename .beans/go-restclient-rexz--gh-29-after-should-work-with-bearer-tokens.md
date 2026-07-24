---
# go-restclient-rexz
title: 'gh-29: --after should work with  bearer tokens'
status: in-progress
type: feature
priority: high
created_at: 2026-07-24T11:32:05Z
updated_at: 2026-07-24T12:06:49Z
parent: go-restclient-vidx
---

## Issue #29

--after flag doesn't work well with dotenv-based auth files.

Tasks:
- [ ] Support dotenv variables in prerequisite requests
- [ ] Allow chaining auth token from .env file
- [ ] Add integration test for OAuth flow
