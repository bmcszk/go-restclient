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