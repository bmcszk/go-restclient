---
# go-restclient-v5w4aayua
title: 'V5W4a: state-cohesive sweep — parse/file/env/server/cookies/oauth2 struct+all methods into their own parts files'
status: in-progress
type: task
priority: normal
created_at: 2026-10-16T13:20:16Z
updated_at: 2026-10-16T14:16:47Z
parent: go-restclient-495b
---

## Proof of Work (orchestrator-verified)
- parseParts struct + 16 methods co-located in parse_parts_test.go (NEW); drained from base + control/refs/parsefile pairs; control+refs pair files held ONLY parseParts → deleted (empty)
- gates re-run: gofmt empty, build/vet 0, lint `0 issues.`, junit 315
- Residual: fileParts, envParts, serverParts, cookies/oauth2 renames (next per-state waves)

## Proof of Work (orchestrator-verified)
- fileParts struct + all 17 methods co-located in file_parts_test.go (NEW); drained from base + core/vars/parsefile pairs
- gates re-run: gofmt empty, build/vet 0, lint `0 issues.`, junit 315
- Residual: envParts, serverParts, cookies/oauth2 renames
