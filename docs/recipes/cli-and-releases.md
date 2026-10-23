# CLI tests, versioning, releases & standing efforts

## CLI test migration (binary-driving tests)
- The library DSL (fluent_parts_test.go) drives the in-process Client and can NOT test the compiled binary; CLI tests get their OWN cliParts DSL in the same package (newParts(t) → given/when/then, state in parts, .and() end-of-line) with build/run-binary logic INSIDE the DSL methods — never duplicate buildBinary/runBinary as free helpers alongside the DSL.
- Migration contract must pin: same test names 1:1, same assertions, ZERO behavior change to production code, and a RUN-count check (`grep -c '^=== RUN.*TestCLI'` on -v output equals the before count) — without the count gate a dropped test looks like a green migration.

## Repo refactor program (user-approved multi-phase slimming)
- All planned phases SHIPPED; env-duality RESOLVED as not-a-dup (selectedEnvironmentName = http-client.env.json env via WithEnvironment; envName = .env.<name> overlay via WithEnvName — different features, comment-clarity fix only, no merge). Optional `internal/parser` extraction LAST (only real dependency-rule seam: parser must not import executor) — ask user before doing it.
- Refactor contracts (pi-dispatched) must pin: impl files only with ZERO `*_test.go` diff, public API byte-identical (programmatic diff of exported func/type names vs HEAD), exact lookup-order/precedence preservation when porting engines, test-referenced symbols KEPT alive (grep `*_test.go` per symbol before deleting), dead code reported-not-deleted.
- Pure-move discipline (Terraform-style): moves verbatim in commit 1, the single sanctioned simplification in commit 2 — never mix; reviewability depends on the separation.
- Verify exported-set parity yourself with a scripted diff (`^func ([A-Z]\w*)\(` + `^type ([A-Z]\w*) ` vs `git show HEAD:<file>` over the full file list) — check cwd before trusting glob results (an empty glob silently reads as "everything missing").
- Deletion accounting that matters: report per-file LOC before/after (`git show HEAD:file | wc -l` vs `wc -l`); PR body carries the net delta and the engine/constructor counts (engines 3→1 etc.), not just "refactored".

## Scorecard improvement levers (repo standing effort)
- Live per-check scores: `https://api.scorecard.dev/projects/github.com/<owner>/<repo>` JSON (`.checks[]` name/score) — query before planning; the README badge number lags.
- Fuzzing check detects in-repo Go-native fuzz targets (`func FuzzXxx(f *testing.F)` in `*_test.go`): 2 targets on pure URL funcs with seed corpus + invariants (no panic, output re-parseable, absolute preserved) suffice; real-fuzz locally before claiming it works (15s/target, count execs, assert 0 crashes).
- Signed-Releases: keyless cosign in the release workflow (`sign-blob` per binary + `.sig` + bundle; job needs `id-token: write`) — the check greps the last 5 releases' ASSETS for `*.sig`/`*.sigstore`/`*.intoto.jsonl`; a signed TAG alone scores nothing.
- CII-Best-Practices: score = none 0 / in-progress 2 / passing 5 / silver 7 / gold 10, queried from bestpractices.dev API by repo URL; registration + questionnaire is a maintainer (user-side) action — agent adds the README badge after the project entry exists. Check for an entry: `bestpractices.dev/projects.json?q=<name>`.
- SBOM check: 5 pts for source/pipeline SBOM, full 10 only as a release artifact — CycloneDX from go.mod via cyclonedx-gomod in the release job (not yet shipped here).
