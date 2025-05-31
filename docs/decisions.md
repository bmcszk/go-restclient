# Decisions Log

Last Updated: 2025-05-30

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

## 2025-05-30: HResp Variable Support

- **Decision 1 (Variable Syntax & Sources)**: Variable support in `.hresp` files will mirror `.http` file functionality. This includes:
    - In-file definition: `@name = value`
    - Substitution syntax: `{{variable}}` or `{{variable | fallback}}`
    - Programmatic provision: Via an optional `map[string]interface{}` argument to `ValidateResponses`.
    - Environment variables as a source.
    - System variables (e.g., `{{$uuid}}`) if a `*Client` is provided.
- **Rationale 1**: Consistency with existing `.http` file variable features (PRD-NFR-001) and user expectations for dynamic response validation.

- **Decision 2 (Processing Order)**: For `.hresp` files used in `ValidateResponses`:
    1. Read the entire `.hresp` file content.
    2. Extract all `@name = value` definitions from the content (using `extractHrespDefines` in `hresp_vars.go`). The lines containing these definitions are removed from the content string.
    3. Perform variable substitution on the remaining content (using `resolveAndSubstitute` in `hresp_vars.go`). This function handles the precedence: programmatic map > in-file vars > environment vars > fallback > system vars.
    4. Pass the fully substituted string content as an `io.Reader` to `parseExpectedResponses` (in `parser.go`) for parsing into `ExpectedResponse` structs.
- **Rationale 2**: This order ensures that variables (including those defined within the file) are resolved *before* the `.hresp` content is parsed into structured `ExpectedResponse` objects. This allows variables to be used in any part of the expected response (status, headers, body).

- **Decision 3 (New Functions/File)**: Variable extraction and substitution logic for `.hresp` files are encapsulated in a new file, `hresp_vars.go`, with functions `extractHrespDefines` and `resolveAndSubstitute`.
- **Rationale 3**: Separation of concerns. This keeps the `validator.go` and `parser.go` files focused on their primary roles. `hresp_vars.go` specifically handles the variable templating aspects for response files.

- **Decision 4 (Signature Change for `ValidateResponses`)**: The `ValidateResponses` function in `validator.go` was updated to accept `client *Client` and `vars ...map[string]interface{}`.
- **Rationale 4**: To allow passing a `Client` instance for system variable resolution and to provide the optional map of execution-time variables.

- **Decision 5 (Parser Update for `parseExpectedResponses`)**: The `parseExpectedResponses` function in `parser.go` was updated to accept an `io.Reader` instead of a file path (for its primary input).
- **Rationale 5**: To allow it to parse in-memory, already-substituted string content, fitting into the new processing pipeline where substitution happens before parsing. The `filePath` argument is retained for error reporting context.

## 2025-05-31: Programmatic Variables via Client Option and Refined Variable Resolution

- **Decision 1 (Programmatic Variables Storage)**: Programmatic variables are now exclusively provided to the `Client` instance via a `WithVars(map[string]interface{})` client option. The `vars ...map[string]interface{}` parameter was removed from `Client.ExecuteFile` and `ValidateResponses`.
- **Rationale 1**: This centralizes client configuration and makes the execution/validation signatures cleaner. It aligns programmatic variables with other client-level settings like `BaseURL` or `DefaultHeaders`.

- **Decision 2 (Variable Resolution Owner & Precedence)**: The `Client` struct is now the primary owner of variable resolution logic for `.http` request execution and `.hresp` validation (via `client.resolveVariablesInText` and `client.substituteDynamicSystemVariables`).
    - **For `.http` file parsing (`parser.go`)**: `@variable = value` definitions have their `value` resolved at parse time. This resolution uses: Client programmatic vars > request-scoped system vars (generated once for the file parse) > OS env vars > .env vars.
    - **For `.http` request execution (URL, Headers, Body in `client.go`)**: Placeholders are resolved with the following precedence: Client programmatic vars > file-scoped `@vars` (already resolved) > request-scoped system vars (generated once *per request*) > OS env vars > .env vars > fallback. This is followed by a pass for dynamic system variables (`{{$dotenv NAME}}`, etc.).
    - **For `.hresp` validation content (`hresp_vars.go#resolveAndSubstitute`)**: Similar to request execution, but with a slightly different order to accommodate how `hresp_vars.go` calls client methods: Request-scoped System Variables (direct system var placeholders) > Client Programmatic vars > .hresp file-scoped `@vars` > OS env vars > fallback. This is followed by a pass for dynamic system variables and system variables exposed via fallbacks.
- **Rationale 2**: Consolidating variable resolution logic within the `Client` and its methods (`resolveVariablesInText`, `substituteDynamicSystemVariables`) promotes consistency. The specific precedence orders aim to provide flexibility while ensuring that higher-priority sources correctly override lower-priority ones. Request-scoped system variables ensure consistency for variables like `{{$uuid}}` within a single request/response validation pass.

- **Decision 3 (Request-Scoped System Variables)**: Simple, no-argument system variables (e.g., `{{$uuid}}`, `{{$timestamp}}`, `{{$randomInt}}`) are now generated once per request execution context (by `client.generateRequestScopedSystemVariables`) and substituted with high precedence. This ensures that multiple instances of `{{$uuid}}` within the same request (e.g., in URL, header, and body) resolve to the *same* value for that request.
- **Rationale 3**: This addresses inconsistencies where previously each instance of `{{$uuid}}` would generate a new UUID. For many use cases (e.g., correlation IDs), a single, consistent value per request is desired.

- **Decision 4 (Parser Update for `@var` Resolution)**: `parser.go` (`parseRequestFile` and `parseRequests`) was updated to accept the `*Client` instance. When an `@variable = value` line is parsed, the `value` on the right-hand side is immediately resolved using the client's programmatic variables, request-scoped system variables (generated for the file parse), OS environment variables, and .env variables. This means file-defined variables can themselves be composed from other, higher-precedence variables.
- **Rationale 4**: Allows more dynamic and flexible definition of file-level variables, making them more powerful (e.g., `@base_api_url = {{PROD_URL}}` where `PROD_URL` is a programmatic or environment variable).

- **Impact**: Significant refactoring in `client.go` (variable resolution logic, `ExecuteFile`), `hresp_vars.go` (`resolveAndSubstitute`), and `parser.go` (`parseRequestFile`, `parseRequests`). Updates to `Client` struct, `WithVars` option. Test files (`client_execute_vars_test.go`, `hresp_vars_test.go`, `parser_test.go`, etc.) updated to reflect new method signatures and test new variable behaviors.
