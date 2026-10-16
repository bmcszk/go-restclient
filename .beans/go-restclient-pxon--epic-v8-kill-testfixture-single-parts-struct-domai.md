---
# go-restclient-pxon
title: 'epic V8: kill testFixture — single parts struct, domain ctors, zero comments'
status: todo
type: epic
priority: normal
created_at: 2026-10-16T18:56:33Z
updated_at: 2026-10-16T18:57:10Z
---

User rejected testFixture composite. Option B (ponytail-minimal): parts = one struct w/ all state; ALL builders on *parts returning *parts (chains untouched, 1253 .and() sites survive); domain files = ctor(3x) + methods; scratch-verified pattern (dsl-v8). W1 retype+fold; W2 comment purge per coding-rules. Parity 270, junit 315, gates green, pi-executed.
