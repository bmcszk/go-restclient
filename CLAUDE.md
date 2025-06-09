# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build & Testing
```bash
# Run all pre-commit checks (build, lint, unit tests)
make check

# Run unit tests only  
make test-unit

# Run linting only
golangci-lint run ./...

# Build the library
go build ./...

# Format code
go fmt ./...

# Run specific test
go test -run TestName ./...

# Tidy dependencies
go mod tidy
```

### Dependencies
Install required tools if missing:
```bash
make install-lint      # Install golangci-lint
make install-gotestsum # Install gotestsum
```

## Architecture Overview

This is a Go library for executing HTTP requests defined in `.http`/`.rest` files and validating responses against `.hresp` files, inspired by JetBrains HTTP Client and VS Code REST Client.

### Core Components

**Client (`client.go`)**: Main entry point providing:
- `NewClient()` with functional options (WithVars, WithBaseURL, WithDefaultHeaders, etc.)
- `ExecuteFile(ctx, filePath)` - parses and executes all requests in a .http file
- `ValidateResponses(responseFilePath, responses...)` - validates actual responses against expected .hresp file
- Variable resolution system with precedence: programmatic > file-defined > environment > system

**Parser (`parser.go`)**: Handles parsing of:
- `.http`/`.rest` files into Request structs
- `.hresp` files into ExpectedResponse structs  
- Variable definitions (`@name = value`) and substitution (`{{name}}`)
- Request separation via `###` delimiters
- Import statements for modular request files

**Validator (`validator.go`)**: Compares actual vs expected responses:
- Status code and message validation
- Header validation (exact match or contains)
- Body validation with placeholder support (`{{$any}}`, `{{$regexp}}`, `{{$anyGuid}}`, etc.)
- Detailed error reporting with diffs

**Variable System**: Supports multiple variable types:
- Custom variables: `@baseUrl = https://api.example.com`
- System variables: `{{$guid}}`, `{{$randomInt}}`, `{{$timestamp}}`, `{{$datetime}}`, etc.
- Environment variables: `{{$processEnv VAR}}`, `{{$dotenv VAR}}`
- Programmatic variables: passed via `WithVars()` client option

### Key Design Principles

- **File-based approach**: HTTP requests defined in text files using standard HTTP syntax
- **Variable substitution**: Rich templating system for dynamic values
- **Response validation**: Structured validation against expected responses
- **Context support**: All operations accept `context.Context` for cancellation/timeouts
- **Error consolidation**: Uses `hashicorp/go-multierror` for comprehensive error reporting

## Testing Strategy

### Unit Tests (`make test-unit`)
- Test files follow `*_test.go` naming convention
- Uses `testify` library for assertions and mocks
- Extensive test data in `testdata/` directory with real `.http` and `.hresp` files
- Client tests use `ExecuteFile` to test full integration path rather than isolated methods

### Test Data Structure
- `testdata/http_request_files/`: Sample .http files for various scenarios
- `testdata/http_response_files/`: Expected .hresp files for validation testing
- `testdata/authentication/`, `testdata/cookies_redirects/`, etc.: Organized by feature

## Important Conventions

- **Variable precedence**: Programmatic vars > file-defined vars > environment vars > system vars > fallback
- **Request-scoped system variables**: Variables like `{{$guid}}` generate once per request execution to ensure consistency
- **Context requirement**: All public methods require `context.Context` as first parameter
- **Error wrapping**: All errors are wrapped with descriptive context using `fmt.Errorf`
- **No E2E tests**: Library focuses on comprehensive unit/integration tests instead of end-to-end testing