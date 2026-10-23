# Fluent DSL discipline

Standing rules for every contract touching the fluent test DSL.

## Standing DSL rules (binding in every contract)

- Minimal comments; YAGNI; `.and()` stays at end of line; all state lives in the `parts` struct.
- No `//nolint`; max line 120 chars; testify via `parts.require` / `parts.assert`.
- COMMENT HYGIENE (binding): test godocs that merely restate the test name are DELETED, not shortened; helper doc one line; no TDD-phase artifacts ("RED: ..."). Sweep metric for review rounds: multi-line comment runs in `git diff master -- '*.go'` must be 0 (working tree, not committed-range — pi edits are uncommitted when you sweep).
- Migration-history / rationale godoc blocks over test funcs: shrink to at most one line or delete. Sweep rule: grep `*_test.go` for 5+ line comments mentioning legacy/migrated/gate/baseline and shrink each.

## TDD commit discipline (binding for this repo)
- RED commit policy, two tiers. Compile-broken RED is NEVER committed — it fails build+lint for the whole working session. Behavioral RED (compiles, lints clean, fails assertions only) MAY be committed+pushed on a feature branch BEFORE the PR exists: the reviewer never sees intermediate pushes, and the gate that counts is CI at PR head; NEVER push a red commit to a branch with an OPEN PR. Shipped work commits GREEN (tests+impl together) or pure-test pinning after impl.
- RED evidence still must exist: pi pastes verbatim RED gate output in the deliverable; orchestrator independently reruns the RED check before dispatching GREEN; RED output + confirmation goes into the bean PoW line ("RED verified: N undefined-symbol failures") so the TDD trail survives without a red commit.
- Behavioral RED (assertion-failing but compiling) is the preferred RED form when feasible — it demonstrates the failing behavior without breaking lint/build gates for the working session; compile-level RED (new struct fields) is acceptable, it just never gets committed.
- Multi-line (3+ line) external-file content strings in tests become committed fixture files; short single-line payloads stay inline.
- File moves only, no method renames, no behavior changes; ambiguous-domain methods stay where they are — never force-fit.

## Test-file organization (reorg requests → audit first)

Before proposing any rename/merge of the test corpus, run the layout audit
recipe in `references/test-corpus-layout.md` (package clauses, Test-func
counts, 1:1 pairing table, public-API census) and bring data, not opinions.
The v3 PAIRED layout is shipped: 24 test files = 10 named pairs
(`X_test.go` ↔ `X_parts_test.go`, stems following the impl/test domain) +
base `parts_test.go` + export/fuzz; every state struct embeds `*parts`
anonymously as FIRST line (zero `root *parts` fields, zero `.root` refs);
each parts file holds ONLY decls its own pair uses (multi-file builders →
base). `references/test-corpus-layout.md` carries the recipe, the rule-3
audit and the no-loss parity script rules.
Standing conclusions to re-verify at each ask: behavior lives end-to-end
(~14 exported symbols vs ~270 Test funcs → no strict unit-1:1 layer); httptest
is in-process (no unit/integration build-tag split).

## DSL file layout (keep new methods in-domain)

- FILE SUFFIX CONVENTION (user 2026-10-13): domain DSL-surface files use the
  `<domain>_parts_test.go` SUFFIX (capture_parts_test.go, validator_parts_test.go…),
  NOT `parts_<domain>_` prefix.
- CURRENT LAYOUT (v10): ONE `parts` struct holds ALL state (core + every domain's
  fields flattened in). Each domain file ALSO declares a stateless extension
  `type <dom>Parts struct { parts }` and its ctor returns
  `(given, when, then *<dom>Parts)` ×3, building the base core inline. ALL
  fluent methods stay `func (p *parts) … *parts`, so mixed-domain chains work
  via promotion with zero test-body churn. REJECTED shapes — do not
  reintroduce: `testFixture`/wrapping composites, per-domain STATE structs,
  generic self-typed `parts[S]` (breaks cross-domain chains).
- CTOR RULE (user binding): each domain file owns a ctor (`newCaptureParts`, …)
  returning its OWN `*<dom>Parts` 3×, constructed by building the base parts
  core inline + initializing its own domain state; NEVER delegate
  (`return newParts(t)` = violation). `newParts(t)` lives once in
  parts_test.go, and base-domain tests live in parts_test.go itself.
- PAIR RULE (user binding): every domain primary is an exact-stem pair
  `X_test.go` ↔ `X_parts_test.go` (capture/env/file/parse/validator/oauth2/
  cookies/server). Feature files inside a domain (`client_execute_control`,
  `*_vars`) share the domain parts file — no pair of their own.
- FILE-STRUCTURE RULE (user binding): every `_parts_test.go` = struct → ctor
  (returns struct pointer, ×3) → methods ONLY. No orphan method files (receiver
  methods in a parts file lacking that file's own struct+ctor = violation; merge
  into the owning domain file).
- FORBIDDEN shape (user rejects on sight): `parts` embedding domain structs —
  embedding flows ONE way only: `<dom>Parts` embeds `parts`; the reverse is a
  bidirectional cycle.
- Builder placement = measured callers: used by 2+ domains' tests → `parts` core;
  single-domain → that domain's file (call sites never change; the composite
  promotes everything). Shared fields (baseDir, expectedFilePath) stay in core —
  grep writers before moving. No `var _` appeasement blocks.

## Fixtures

- Root is `test/data/` — http_request_files/, http_response_files/, external_files/, graphql/, etc. `test/fixtures/` is retired; its files moved into http_request_files/ and http_response_files/. New fixtures go under test/data/, never a parallel root.
- Fixture-loading DSL methods delegate to the inline variants (read file → call aHttpFileWithExternalFile) — one code path per behavior.
