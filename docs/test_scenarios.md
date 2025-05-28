# Test Scenarios

Last Updated: 2025-05-28

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

## REQ-LIB-009: The library must provide a method (`ValidateResponses`) to validate one or more actual HTTP responses against a corresponding set of expected responses defined in a single `.http` file (using `###` as a separator for multiple expected responses).

- SCENARIO-LIB-009-001: Validate a single successful actual response (status, headers, body) against an expected response file containing one definition.
  - Expected: Validation passes (no error returned from `ValidateResponses`).
- SCENARIO-LIB-009-002: Validate a single actual response where the status code mismatches the definition in an expected response file.
  - Expected: `ValidateResponses` returns an error detailing the status code mismatch.
- SCENARIO-LIB-009-003: Validate a single actual response where a header mismatches the definition in an expected response file.
  - Expected: `ValidateResponses` returns an error detailing the header mismatch.
- SCENARIO-LIB-009-004: Validate a single actual response where the body (JSON) mismatches the definition in an expected response file.
  - Expected: `ValidateResponses` returns an error detailing the body mismatch (e.g., a diff).
- SCENARIO-LIB-009-005: Validate a single actual response against an expected response file that specifies only a status code.
  - Expected: Validation passes if status code matches; headers/body in actual response are ignored for this check if not specified in expected.
- SCENARIO-LIB-009-006: Validate a single actual response against an expected response file that specifies only certain headers.
  - Expected: Validation passes if specified headers match; status/body/other headers are ignored if not specified.
- SCENARIO-LIB-009-007: Call `ValidateResponses` with a path to a missing expected response file.
  - Expected: `ValidateResponses` returns an error indicating the file could not be parsed/found.
- SCENARIO-LIB-009-008: Call `ValidateResponses` with a path to an incorrectly formatted expected response file.
  - Expected: `ValidateResponses` returns an error indicating the file parsing failed.
- SCENARIO-LIB-009-009: Validate multiple actual responses against an expected response file containing multiple definitions (separated by `###`).
  - Expected: Validation passes for all corresponding pairs.
- SCENARIO-LIB-009-010: Validate multiple actual responses where one response mismatches its corresponding definition in a multi-response expected file.
  - Expected: `ValidateResponses` returns a multierror containing the specific mismatch.
- SCENARIO-LIB-009-011: Call `ValidateResponses` with a different number of actual responses than expected responses defined in the file.
  - Expected: `ValidateResponses` returns an error indicating the count mismatch.

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

## REQ-LIB-012: Exclusive use of .http format for expected responses

- SCENARIO-LIB-012-001: Attempt to load an expected response defined in a JSON file.
  - Expected: The library either errors out or fails to parse the file, as only `.http` format is supported for expected responses.
- SCENARIO-LIB-012-002: Attempt to load an expected response defined in a YAML file.
  - Expected: The library either errors out or fails to parse the file, as only `.http` format is supported for expected responses.

## REQ-LIB-013: Support for user-defined custom variables

- SCENARIO-LIB-013-001: Define and use a simple custom variable in the request URL.
  - File content:
    ```http
    @host = https://example.com
    GET {{host}}/users
    ```
  - Expected: Request sent to `https://example.com/users`.
- SCENARIO-LIB-013-002: Define and use a custom variable in a request header.
  - File content:
    ```http
    @token = mysecrettoken
    GET https://api.example.com/data
    Authorization: Bearer {{token}}
    ```
  - Expected: `Authorization` header sent as `Bearer mysecrettoken`.
- SCENARIO-LIB-013-003: Define and use a custom variable in the request body.
  - File content:
    ```http
    @username = testuser
    POST https://api.example.com/login
    Content-Type: application/json

    {
      "user": "{{username}}"
    }
    ```
  - Expected: JSON body sent with `user` field as `testuser`.
- SCENARIO-LIB-013-004: Override a custom variable defined earlier in the file.
  - File content:
    ```http
    @baseUrl = https://api.sandbox.com
    GET {{baseUrl}}/status

    ###

    @baseUrl = https://api.production.com
    GET {{baseUrl}}/status
    ```
  - Expected: First request to `https://api.sandbox.com/status`, second to `https://api.production.com/status`.
- SCENARIO-LIB-013-005: Use an undefined custom variable.
  - Expected: Error reported, or variable is substituted as an empty string (behavior should be defined).

## REQ-LIB-014: Support for {{$guid}} system variable

- SCENARIO-LIB-014-001: Use `{{$guid}}` in request URL.
  - Expected: A valid GUID is generated and inserted into the URL.
- SCENARIO-LIB-014-002: Use `{{$guid}}` in request header.
  - Expected: A valid GUID is generated and inserted into the header value.
- SCENARIO-LIB-014-003: Use `{{$guid}}` in request body.
  - Expected: A valid GUID is generated and inserted into the body.
- SCENARIO-LIB-014-004: Multiple `{{$guid}}` instances in one request generate different GUIDs.
  - File content:
    ```http
    GET https://example.com/{{$guid}}/{{$guid}}
    ```
  - Expected: The two GUIDs in the URL are different.

## REQ-LIB-015: Support for {{$randomInt min max}} system variable

- SCENARIO-LIB-015-001: Use `{{$randomInt 1 10}}` in request URL.
  - Expected: A random integer between 1 and 10 (inclusive) is generated and inserted.
- SCENARIO-LIB-015-002: Use `{{$randomInt 100 105}}` in request body.
  - Expected: A random integer between 100 and 105 (inclusive) is generated and inserted.
- SCENARIO-LIB-015-003: Use `{{$randomInt}}` (missing arguments).
  - Expected: Error or default behavior (e.g., 0-100, to be defined).
- SCENARIO-LIB-015-004: Use `{{$randomInt max min}}` (min > max).
  - Expected: Error or specific behavior (e.g., swap, to be defined).

## REQ-LIB-016: Support for {{$timestamp}} system variable

- SCENARIO-LIB-016-001: Use `{{$timestamp}}` in request header.
  - Expected: Current UTC timestamp (Unix epoch seconds) is inserted.

## REQ-LIB-017: Support for {{$datetime format}} system variable

- SCENARIO-LIB-017-001: Use `{{$datetime 'YYYY-MM-DD'}}`.
  - Expected: Current UTC date formatted as `YYYY-MM-DD`.
- SCENARIO-LIB-017-002: Use `{{$datetime 'HH:mm:ss'}}`.
  - Expected: Current UTC time formatted as `HH:mm:ss`.
- SCENARIO-LIB-017-003: Use `{{$datetime 'rfc1123'}}` (or similar common format name).
  - Expected: Current UTC datetime in RFC1123 format.
- SCENARIO-LIB-017-004: Use `{{$datetime}}` (missing format).
  - Expected: Error or default format (e.g., ISO8601, to be defined).

## REQ-LIB-018: Support for {{$localDatetime format}} system variable

- SCENARIO-LIB-018-001: Use `{{$localDatetime 'YYYY-MM-DD HH:mm'}}`.
  - Expected: Current local date and time formatted as `YYYY-MM-DD HH:mm`.
- SCENARIO-LIB-018-002: Use `{{$localDatetime}}` (missing format).
  - Expected: Error or default format (e.g., ISO8601 local, to be defined).

## REQ-LIB-019: Support for {{$processEnv variableName}} system variable

- SCENARIO-LIB-019-001: Use `{{$processEnv HOME}}` (assuming HOME is set).
  - Expected: Value of `HOME` environment variable is substituted.
- SCENARIO-LIB-019-002: Use `{{$processEnv NON_EXISTENT_VAR}}`.
  - Expected: Error or empty string substitution (behavior to be defined).

## REQ-LIB-020: Support for {{$dotenv variableName}} system variable

- SCENARIO-LIB-020-001: Use `{{$dotenv DOTENV_VAR}}` where `DOTENV_VAR` exists in a `.env` file in the request file's directory.
  - Expected: The value of `DOTENV_VAR` from the `.env` file is substituted.
- SCENARIO-LIB-020-002: Use `{{$dotenv UNDEFINED_DOTENV_VAR}}` where `UNDEFINED_DOTENV_VAR` does not exist in the `.env` file.
  - Expected: Error or empty string substitution (behavior to be defined).
- SCENARIO-LIB-020-003: Use `{{$dotenv SOME_VAR}}` with no `.env` file present in the request file's directory.
  - Expected: Error or empty string substitution (behavior to be defined).

## REQ-LIB-021: Programmatic custom variables for ExecuteFile

- SCENARIO-LIB-021-001: Pass a map of custom variables to `ExecuteFile` and verify they are substituted in the URL.
  - Example: `vars := map[string]string{"userId": "prog_user_123"}`
  - File content: `GET https://api.example.com/users/{{userId}}`
  - Expected: Request sent to `https://api.example.com/users/prog_user_123`.
- SCENARIO-LIB-021-002: Pass a map of custom variables to `ExecuteFile` and verify they are substituted in headers.
  - Example: `vars := map[string]string{"authToken": "prog_token_abc"}`
  - File content: `GET /data\nAuthorization: Bearer {{authToken}}`
  - Expected: `Authorization` header is `Bearer prog_token_abc`.
- SCENARIO-LIB-021-003: Pass a map of custom variables to `ExecuteFile` and verify they are substituted in the body.
  - Example: `vars := map[string]string{"orderId": "prog_order_456"}`
  - File content: `POST /orders\nContent-Type: application/json\n\n{"id": "{{orderId}}"}`
  - Expected: JSON body sent with `id` field as `prog_order_456`.
- SCENARIO-LIB-021-004: Programmatic variables override variables defined in the request file.
  - Example: `vars := map[string]string{"baseUrl": "https://prog.example.com"}`
  - File content: `@baseUrl = https://file.example.com\nGET {{baseUrl}}/path`
  - Expected: Request sent to `https://prog.example.com/path`.
- SCENARIO-LIB-021-005: Variables defined in the file are still used if not overridden programmatically.
  - Example: `vars := map[string]string{"otherVar": "prog_value"}`
  - File content: `@fileVar = file_value\nGET /path?p1={{fileVar}}&p2={{otherVar}}`
  - Expected: Request sent to `/path?p1=file_value&p2=prog_value`.
- SCENARIO-LIB-021-006: Pass an empty map of variables to `ExecuteFile`.
  - Expected: No error, and only file-defined variables (if any) are used.
- SCENARIO-LIB-021-007: Pass `nil` as the variables map to `ExecuteFile`.
  - Expected: No error, and only file-defined variables (if any) are used.
