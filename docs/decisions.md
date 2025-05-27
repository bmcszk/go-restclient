# Decisions Log

Last Updated: 2025-05-27

## YYYY-MM-DD: Initial Design Choices

- **Decision**: The project will be a Go library for sending HTTP requests defined in `.rest`/`.http` files and validating responses.
- **Rationale**: To provide a reusable component for E2E testing in Go projects, inspired by tools like VSCode REST Client but as a library.
- **Alternatives Considered**: n/a (Initial project scope definition). 

## 2025-05-27: Removal of End-to-End (E2E) Testing for the Library Itself

- **Decision**: The `go-restclient` library will not have its own suite of E2E tests. The focus will be on comprehensive unit and integration tests for its components.
- **Rationale**: User directive. The library is intended to be *used* in other projects' E2E tests, but does not require its own complex E2E setup (e.g., Dockerized dependencies) for self-testing.
- **Impact**: Removed E2E tasks from `docs/tasks.md`, deleted `e2e/` directory, removed `unimock` dependency, and updated project guidelines to reflect this change. Test coverage will be ensured by unit/integration tests using real `.http` files and mocked transports/servers.

## 2025-05-27: Client API Refactoring and Test Strategy Standardization

- **Decision 1 (API Change)**: The `Client.ExecuteRequest(*Request)` method was made unexported (`client.executeRequest(*Request)`).
- **Rationale 1**: To simplify the public API surface. The primary intended entry point for execution is `ExecuteFile`, which handles parsing and then calls the internal `executeRequest`.
- **Decision 2 (API Change)**: The `Client.ExecuteFile` method and the internal `executeRequest` method now require a `context.Context` as their first parameter.
- **Rationale 2**: To allow for request cancellation, timeouts, and deadline propagation, which are standard practices in modern Go HTTP clients.
- **Decision 3 (Testing Strategy)**: All unit tests for the `Client`'s execution logic (`client_test.go`) must be based on executing `.http` files via `ExecuteFile`. Direct testing of `executeRequest` (now unexported) is removed.
- **Rationale 3**: To ensure client tests accurately reflect the primary use case of the library and to test the full flow from file parsing to request execution. This aligns with the guideline that scenarios in `docs/test_scenarios.md` should be covered by unit tests using real `.http` files.
- **Impact**: `client.go` modified. `client_test.go` significantly refactored: old `TestExecuteRequest_*` tests removed, existing `TestExecuteFile_*` tests updated, and new `TestExecuteFile_*` tests added to cover scenarios previously tested via direct `ExecuteRequest` calls (e.g., `BaseURL` and `DefaultHeaders` functionality).

## 2025-05-27: Expected Response File Format and Parsing

- **Decision**: Expected responses will be defined in files using the same `.http`-like format as request files. This includes:
    - A status line (e.g., `HTTP/1.1 200 OK`).
    - Header key-value pairs.
    - An optional body, separated from headers by a blank line.
    - Multiple expected response definitions can be included in a single file, separated by `###`.
- **Rationale**: To maintain consistency with the request file format, making it easier for users to create and manage expected response definitions. This approach is also more flexible than a simple JSON or YAML structure for defining HTTP responses.
- **Implementation**: The `parser.go` file contains `ParseExpectedResponseFile` and `parseExpectedResponses` functions to handle this format.
- **Exclusion**: Support for defining expected responses in other formats (e.g., dedicated JSON or YAML files) is explicitly excluded to simplify the library and maintain a consistent user experience.
