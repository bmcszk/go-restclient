# Test Scenarios

Last Updated: 2025-05-27

## REQ-LIB-001: Parse request definitions from `.rest` or `.http` files

- SCENARIO-LIB-001-001: Parse a simple GET request from an `.http` file.
  - File content:
    ```http
    GET https://jsonplaceholder.typicode.com/todos/1
    ```
  - Expected outcome: Correctly parsed method (GET), URL, no headers, no body.
- SCENARIO-LIB-001-002: Parse a GET request with headers from an `.http` file.
  - File content:
    ```http
    GET https://jsonplaceholder.typicode.com/todos/1
    Accept: application/json
    User-Agent: go-restclient-test
    ```
  - Expected outcome: Correctly parsed method (GET), URL, headers (Accept, User-Agent), no body.
- SCENARIO-LIB-001-003: Parse a POST request with a JSON body from an `.http` file.
  - File content:
    ```http
    POST https://jsonplaceholder.typicode.com/posts
    Content-Type: application/json

    {
      "title": "foo",
      "body": "bar",
      "userId": 1
    }
    ```
  - Expected outcome: Correctly parsed method (POST), URL, headers (Content-Type), and JSON body.
- SCENARIO-LIB-001-004: Parse a POST request with a plain text body from an `.http` file.
  - File content:
    ```http
    POST https://example.com/submit
    Content-Type: text/plain

    This is a plain text body.
    ```
  - Expected outcome: Correctly parsed method (POST), URL, headers (Content-Type), and plain text body.
- SCENARIO-LIB-001-005: Parse an `.http` file with comments.
  - File content:
    ```http
    # This is a comment
    GET https://jsonplaceholder.typicode.com/todos/1
    // Another comment
    Accept: application/json
    ```
  - Expected outcome: Comments are ignored, and the request is parsed correctly.
- SCENARIO-LIB-001-006: Parse an `.http` file with variables (if REQ-LIB-007 is implemented).
  - File content:
    ```http
    @host = https://jsonplaceholder.typicode.com
    GET {{host}}/todos/1
    ```
  - Expected outcome: Variable `host` is correctly substituted.
- SCENARIO-LIB-001-007: Handle an empty `.http` file.
  - Expected outcome: Error or no request parsed.
- SCENARIO-LIB-001-008: Handle an `.http` file with only comments.
  - Expected outcome: Error or no request parsed.
- SCENARIO-LIB-001-009: Handle an `.http` file with invalid syntax (e.g., missing URL).
  - Expected outcome: Error indicating invalid syntax.
- SCENARIO-LIB-001-010: Parse multiple request definitions from a single `.http` file (if supported, often separated by `###`).
  - File content:
    ```http
    GET https://jsonplaceholder.typicode.com/todos/1

    ###

    POST https://jsonplaceholder.typicode.com/posts
    Content-Type: application/json

    {
      "title": "another",
      "body": "request",
      "userId": 2
    }
    ```
  - Expected outcome: Both requests are parsed correctly. If multiple requests per file are not supported by the chosen parser, this scenario becomes about ensuring only the first is parsed or an error is raised.

## REQ-LIB-002: Request file format support

*(Test scenarios for REQ-LIB-002 are largely covered by SCENARIO-LIB-001-xxx as they test parsing of method, URL, headers, and body.)*

## REQ-LIB-003: Send parsed HTTP request

- SCENARIO-LIB-003-001: Send a parsed GET request and verify it reaches a mock server.
- SCENARIO-LIB-003-002: Send a parsed POST request with a JSON body and verify the mock server receives the correct data.

## REQ-LIB-004: Capture HTTP response

- SCENARIO-LIB-004-001: Capture status code from a mock server response.
- SCENARIO-LIB-004-002: Capture headers from a mock server response.
- SCENARIO-LIB-004-003: Capture JSON body from a mock server response.
- SCENARIO-LIB-004-004: Capture plain text body from a mock server response.

## REQ-LIB-005: Specify expected response

*(This will depend on how expected responses are defined. Scenarios to be added once the mechanism is clearer.)*
- SCENARIO-LIB-005-001: Define an expected status code and match it.
- SCENARIO-LIB-005-002: Define expected headers and match them.
- SCENARIO-LIB-005-003: Define an expected JSON body and match it.
- SCENARIO-LIB-005-004: Define an expected plain text body and match it.

## REQ-LIB-006: Compare actual and expected response

- SCENARIO-LIB-006-001: Report success when actual response matches expected (status, headers, body).
- SCENARIO-LIB-006-002: Report failure when actual status code differs from expected.
- SCENARIO-LIB-006-003: Report failure when an expected header is missing or has a different value.
- SCENARIO-LIB-006-004: Report failure when actual JSON body differs from expected.
- SCENARIO-LIB-006-005: Report failure when actual plain text body differs from expected.

## REQ-LIB-007: Support for variables in request files

*(Covered by SCENARIO-LIB-001-006 if implemented. Additional scenarios might be needed for variable scope, types, etc.)*
- SCENARIO-LIB-007-001: Use a variable defined in the same file.
- SCENARIO-LIB-007-002: Use environment variables.
- SCENARIO-LIB-007-003: Handle undefined variables. 

## REQ-LIB-008: The library should handle errors from `executeRequest` using an `errgroup`.

- SCENARIO-LIB-008-001: Verify that if one request in a multi-request execution (within `ExecuteFile`) fails, `errgroup` correctly captures and returns the error.
- SCENARIO-LIB-008-002: Verify that if multiple requests fail, `errgroup` captures all errors (or the first one, depending on `errgroup`'s behavior configuration).
- SCENARIO-LIB-008-003: Verify that successful requests complete even if other requests in the group fail, and their results are available (if applicable).

## REQ-LIB-009: The library must provide a method to validate if the actual HTTP response matches an expected response defined in a file.

- SCENARIO-LIB-009-001: Validate a successful response (status, headers, body) against an expected response file.
  - Expected: Validation passes.
- SCENARIO-LIB-009-002: Validate a response where the status code mismatches the expected response file.
  - Expected: Validation fails with a clear error about status code mismatch.
- SCENARIO-LIB-009-003: Validate a response where a header mismatches the expected response file.
  - Expected: Validation fails with a clear error about header mismatch.
- SCENARIO-LIB-009-004: Validate a response where the body (JSON) mismatches the expected response file.
  - Expected: Validation fails with a clear error/diff about body mismatch.
- SCENARIO-LIB-009-005: Validate a response against an expected response file that specifies only a status code.
  - Expected: Validation passes if status code matches, ignoring headers/body.
- SCENARIO-LIB-009-006: Validate a response against an expected response file that specifies only certain headers.
  - Expected: Validation passes if specified headers match, ignoring others and body.
- SCENARIO-LIB-009-007: Handle a missing expected response file.
  - Expected: Error reported.
- SCENARIO-LIB-009-008: Handle an incorrectly formatted expected response file.
  - Expected: Error reported.

## REQ-LIB-010: The response file format should allow for multiple responses, separated by `###`, similar to request files.

- SCENARIO-LIB-010-001: Parse an expected response file containing multiple response definitions separated by `###`.
  - Expected: All response definitions are parsed correctly.
- SCENARIO-LIB-010-002: Match a sequence of actual responses to a sequence of expected responses from a multi-response file.
- SCENARIO-LIB-010-003: Handle mismatch in the number of actual responses vs. expected responses in a multi-response file.

## REQ-LIB-011: The library must be tested for its capability to handle multiple requests separated by `###` in `.http` files.

- SCENARIO-LIB-011-001: Execute an `.http` file containing two valid GET requests separated by `###`.
  - Expected: Both requests are sent, and their respective responses are captured.
- SCENARIO-LIB-011-002: Execute an `.http` file where the first request is valid, and the second is invalid (e.g., bad syntax), separated by `###`.
  - Expected: The first request is executed; an error is reported for the second. Behavior of `ExecuteFile` (continues or stops) should be defined and tested. (Relates to REQ-LIB-008)
- SCENARIO-LIB-011-003: Execute an `.http` file with multiple (>2) requests separated by `###`.
  - Expected: All requests are processed sequentially.
- SCENARIO-LIB-011-004: (Covered by SCENARIO-LIB-001-010 for parsing aspect) Ensure correct parsing and execution of multiple requests defined in `sample1.http`.
