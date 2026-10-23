# Agent Guidelines for go-restclient

## Commands
- `make check` - Run all pre-commit checks (lint + unit tests)
- `make lint` - Run golangci-lint
- `make test-unit` - Run unit tests with coverage
- `go test -run TestName ./...` - Run single test
- `make fmt` - Format code

## Code Style
- Go 1.21+ formatting with `go fmt`
- Import grouping: stdlib, third-party, local
- Error handling: always check errors, use multierror for aggregation
- Naming: PascalCase for exported, camelCase for unexported
- No //nolint comments - fix issues instead
- Max line length: 120 characters
- Function length: < 100 lines
- File length: < 1000 lines

## Testing
- Use testify for assertions
- Test files: *_test.go pattern
- Coverage required for all changes

## Fluent test DSL
- Test DSL + guideline: docs/fluent-testing-guideline.md (binding for any test change: given/when/then, state in parts, .and() end of line, comments <= 1 line whole diff, fixtures under test/data/, no linter-appeasement var _ blocks).
- Existing DSL surface lives in parts_test.go (base) plus one <domain>_parts_test.go per domain (capture/client/cookies/env/file/oauth2/parse/server/validator), each paired with its <domain>_test.go primary; feature test files of a domain share that domain's parts file.

## Docs map

- `docs/fluent-testing-guideline.md` — binding fluent-DSL guideline.
- `docs/recipes/pr-workflow.md` — PR workflow, review rounds, DoD, close-out.
- `docs/recipes/github-mechanics.md` — gh CLI mechanics and pitfalls.
- `docs/recipes/dsl-discipline.md` — standing DSL rules, TDD discipline, file layout.
- `docs/recipes/cli-and-releases.md` — CLI tests, versioning, releases.
- `docs/recipes/pitfalls.md` — known pitfalls.
