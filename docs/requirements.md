# Requirements

Last Updated: 2025-05-27

## Functional Requirements

- REQ-LIB-001: The library must parse request definitions from `.rest` or `.http` files.
- REQ-LIB-002: The request file format should support defining HTTP method, URL, headers, and body.
- REQ-LIB-003: The library must be able to send the parsed HTTP request to the target server.
- REQ-LIB-004: The library must capture the HTTP response (status code, headers, body).
- REQ-LIB-005: The library must allow specifying an expected response (status code, headers, body) via a corresponding `.http` file.
- REQ-LIB-006: The library must compare the actual response with the expected response and report discrepancies.
- REQ-LIB-007: Support for variables in request files (e.g., for hostnames, tokens) is desirable.
- REQ-LIB-008: The library should handle errors from `executeRequest`.
- REQ-LIB-009: The library must provide a method (`ValidateResponses`) to validate one or more actual HTTP responses against a corresponding set of expected responses defined in a single `.http` file (using `###` as a separator for multiple expected responses).
- REQ-LIB-010: The response file format should allow for multiple responses, separated by `###`, similar to request files.
- REQ-LIB-011: The library must be tested for its capability to handle multiple requests separated by `###` in `.http` files.
- REQ-LIB-012: The library must exclusively use the `.http` file format (as exemplified by `testdata/http_response_files/sample1.http` and allowing `###` for multiple responses) for defining expected responses. Support for other formats like JSON or YAML for expected responses is explicitly excluded.
- REQ-LIB-013: The library must support user-defined custom variables in request files.
- REQ-LIB-014: The library must support a `{{$guid}}` system variable that generates a new GUID.
- REQ-LIB-015: The library must support a `{{$randomInt min max}}` system variable that generates a random integer within the specified range (inclusive).
- REQ-LIB-016: The library must support a `{{$timestamp}}` system variable that provides the current UTC timestamp (Unix epoch seconds).
- REQ-LIB-017: The library must support a `{{$datetime format}}` system variable that provides the current UTC datetime formatted according to the given `format` string.
- REQ-LIB-018: The library must support a `{{$localDatetime format}}` system variable that provides the current local datetime formatted according to the given `format` string.
- REQ-LIB-019: The library must support a `{{$processEnv variableName}}` system variable that substitutes the value of the specified system environment variable.
- REQ-LIB-020: The library must support a `{{$dotenv variableName}}` system variable that substitutes the value of the specified variable from a `.env` file in the current working directory or a user-specified path.
- REQ-LIB-021: The library must allow users to programmatically provide a map of custom variables during a call to `ExecuteFile`, which can be used for substitution in the request file, potentially overriding variables defined within the file itself.

## Non-Functional Requirements

- NFR-LIB-001: The library should be easy to integrate into Go E2E test suites.
- NFR-LIB-002: The library should have clear documentation and examples.
- NFR-LIB-003: The library should have good unit test coverage.
