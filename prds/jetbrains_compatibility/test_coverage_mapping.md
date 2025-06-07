# HTTP Syntax Feature Coverage Analysis

This document maps requirements from `http_syntax.md` to existing tests in the codebase. It helps identify which features are already tested and implemented, and which ones need additional coverage.

## FR1: Request Structure Basics

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR1.1 | Support `.http` and `.rest` file extensions | `TestParseRequests_*` (implicit) | `parser_test.go` | Partial | Code comment in `parser.go` mentions `.rest` but no explicit tests exist for `.rest` files. Current tests use `.http` files. |
| FR1.2 | Support multiple requests delimited by `###` | `TestParseRequests_Separators` | `parser_test.go` | Complete | Tests separation of requests with `###` |
| FR1.3 | Support request naming | `TestParseRequests_Name` | `parser_test.go` | Partial | Tests for `### Name` but may need more tests for `# @name` directive |
| FR1.4 | Support comments using `#` and `//` | `TestParseRequests_SeparatorComments` (primarily for `#` style and comments near separators) | `parser_test.go` | Partial | Current tests cover `#` style comments and `//` when used for directives or on the same line as a request. Needs explicit tests for general `//` style comments on their own lines. |
| FR1.5 | Support all major HTTP methods | Various tests (e.g., `TestParseRequests_SeparatorComments`, `TestParseRequestFile_VariableScoping`) | `parser_test.go` | Partial | GET, POST, PUT are covered. PATCH, HEAD, OPTIONS are not explicitly tested. |
| FR1.6 | Support request line format | Various tests (e.g., `TestParseRequests_SeparatorComments`) | `parser_test.go` | Partial | Implicitly tested by all request parsing. Needs explicit tests for parsing optional HTTP version in the request line. |
| FR1.7 | Parse headers | Various tests | `parser_test.go` | Complete | Various tests cover header parsing |
| FR1.8 | Handle standard body formats | Various tests (e.g., `TestParseRequest_BodyContent`, `TestParseRequestFile_FileBody`) | `parser_test.go` | Partial | Test data exists for `application/json`, `application/xml`, `text/plain`, `application/x-www-form-urlencoded`, and `multipart/form-data`. Coverage depth for each type may vary. |

## FR2: Environment Variables and Placeholders

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR2.1 | Support variable substitution in URL, headers, body | `TestParseRequestFile_VariableScoping` | `parser_test.go` | Complete | Tests substitution of file-level and request-specific variables in URLs and bodies (e.g., in `testdata/variables/variable_references.http`) |
| FR2.2 | Load environment variables from JSON files | `TestExecuteFile_WithHttpClientEnvJson` | `client_execute_vars_test.go` | Complete | Tests loading from `http-client.env.json` |
| FR2.3 | Support in-place variables (file-level @name = value) | `TestParseRequestFile_VariableScoping` | `parser_test.go` | Complete | Tests defining and using file-level variables with `@name = value` syntax (e.g., in `testdata/variables/variable_references.http`) |
| FR2.4 | Support `$shared` environment (via `http-client.private.env.json`) | `TestExecuteFile_WithHttpClientEnvJson` | `client_execute_vars_test.go` | Complete | Tested via the `privateEnvFilePath` in test case "SCENARIO-LIB-018-005", which uses `http-client.private.env.json` for shared/override variables. |

## FR3: Dynamic System Variables

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR3.1 | Support JetBrains-specific system variables (`{{$guid}}`, `{{$randomInt}}`, `{{$timestamp}}`, `{{$processEnv VAR_NAME}}`, extended random, datetime formats) | Complete | `client_execute_vars_test.go`:
  - `TestExecuteFile_WithProcessEnvSystemVariable` (for `{{$processEnv}}`)
  - `TestExecuteFile_WithLocalDatetimeSystemVariable` (covers `{{$timestamp}}`)
  - `TestExecuteFile_WithExtendedRandomSystemVariables` (for `{{$random.*}}` variants: integer, float, alphabetic, alphanumeric, hexadecimal, email)
`client_execute_system_vars_test.go`:
  - `TestExecuteFile_WithDatetimeSystemVariables` (for `{{$datetime}}` and `{{$localDatetime}}` with various formats)
Coverage for `{{$guid}}`, `{{$random.uuid}}`, and `{{$randomInt}}` is provided by tests utilizing `testdata/http_request_files/system_var_guid.http`, `testdata/system_variables/random_values.http`, and `testdata/system_variables/basic_system_vars.http`. |
| FR3.2 | Support VS Code-specific system variables (`{{$aadToken}}`, `{{$azureADToken}}`, `{{$dotenv VAR_NAME}}`, `{{$env VAR_NAME}}`, `{{$machineEnvs VAR_NAME}}`, `{{$processEnv VAR_NAME}}`, `{{$shared VAR_NAME}}`) | Partial | `client_execute_vars_test.go`:
  - `TestExecuteFile_WithDotEnvSystemVariable` (for `{{$dotenv VAR_NAME}}`)
  - `TestSubstituteDynamicSystemVariables_EnvVars` (for `{{$env VAR_NAME}}`)
  - `TestExecuteFile_WithProcessEnvSystemVariable` (for `{{$processEnv VAR_NAME}}` - shared with JetBrains FR3.1)
Missing tests for `{{$aadToken}}`, `{{$azureADToken}}`, `{{$machineEnvs VAR_NAME}}`, `{{$shared VAR_NAME}}` (VSCode variant). |
| FR3.3 | Support custom dynamic variables defined in scripts (e.g., `{% client.global.set("name", "value"); %}`, `client.test(...)`, access `response.body`) | Missing | - | Missing | No test data or tests found for `{% ... %}` script blocks or related script APIs like `client.global.set`, `request.variables.set`, `client.test`. |

## FR4: Request Bodies

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR4.1 | Support `application/json` bodies | Various | `parser_test.go` | Complete | JSON bodies tested in various contexts |
| FR4.2 | Support `application/x-www-form-urlencoded` | Not found | - | Missing | Needs specific tests for form urlencoded |
| FR4.3 | Support `multipart/form-data` | Not found | - | Missing | Needs tests for multipart and file uploads |
| FR4.4 | Support `text/plain` and raw text bodies | Various | `parser_test.go` | Partial | Basic text is tested but no specific tests for content type |
| FR4.5 | Support GraphQL request format | Not found | - | Missing | Needs tests for GraphQL syntax |

## FR5: Authentication

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR5.1 | Support Basic Authentication | `TestBasicAuthHeader`, `TestBasicAuthURL` | `parser_authentication_test.go` | Complete | Covers both header (`Authorization: Basic ...`) and URL-based (`user:pass@host`) basic authentication. |
| FR5.2 | Support Bearer token authentication | `TestBearerTokenAuth` | `parser_authentication_test.go` | Complete | Covers `Authorization: Bearer <token>` syntax. |
| FR5.3 | Support OAuth authentication | `TestOAuthFlowWithRequestReferences` | `parser_authentication_test.go` | Complete | Covers parsing of requests involved in an OAuth flow, including using response references for tokens (e.g., `Authorization: Bearer {{getToken.response.body.access_token}}`). End-to-end execution is not covered by parser tests. |

## FR6: Request Settings

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR6.1 | Support request-specific options | `TestNoRedirectDirective`, `TestNoCookieJarDirective` | `parser_request_settings_test.go` | Complete | Covers parsing of `@no-redirect` and `@no-cookie-jar` directives. |
| FR6.2 | Support request timeout setting | `TestTimeoutDirective` | `parser_request_settings_test.go` | Complete | Covers parsing of `@timeout <milliseconds>` directive. |

## FR7: Response Handling and Validation

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR7.1 | Support defining expected responses | `TestValidateResponses_*` | `validator_body_test.go` | Partial | Basic validation tested, not all response formats |
| FR7.2 | Support response reference variables | `TestParseChainedRequests`, `TestOAuthFlowWithRequestReferences` | `parser_response_validation_test.go`, `parser_authentication_test.go` | Complete | Covers parsing of response references like `{{reqName.response.body.field}}`. |
| FR7.3 | Support response validation placeholders | Not found | - | Missing | Needs tests for validation placeholders |

## FR8: Request Imports

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR8.1 | Support importing shared request files (e.g., variables, common headers from another file) | `TestParseRequestFile_Imports` | `parser_test.go` | Missing | Feature is not documented in `docs/http_syntax.md` and currently not supported. Parser test `TestParseRequestFile_Imports` confirms `@import` directives are silently ignored, not that functionality is implemented. |
| FR8.2 | Support correct variable scoping with imports (if imports were functional) | `TestParseRequestFile_Imports` | `parser_test.go` | Missing | Dependent on FR8.1. Since imports are ignored, variable scoping with functional imports is not tested/supported. Test data `main_variable_override.http` exists but its import is ignored. |
| FR8.3 | Detect and handle circular imports (by ignoring them) | `TestParseRequestFile_Imports` | `parser_test.go` | Complete | Parser test `TestParseRequestFile_Imports` confirms that files with circular `@import` directives (e.g., `main_circular_import_a.http`) are processed without error as imports are ignored. |

## FR9: Cookies and Redirect Handling

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR9.1 | Support automatic cookie handling | `TestCookieJarHandling` | `client_cookies_redirects_test.go` | Complete | Tests automatic cookie sending by the client and respects the `@no-cookie-jar` directive. Uses `testdata/cookies_redirects/with_cookie_jar.http` and `without_cookie_jar.http`. |
| FR9.2 | Support automatic redirect following | `TestRedirectHandling` | `client_cookies_redirects_test.go` | Complete | Tests automatic redirect following by the client and respects the `@no-redirect` directive. Uses `testdata/cookies_redirects/with_redirect.http` and `without_redirect.http`. |

## Summary of Coverage Gaps

1. **Missing Test Coverage:**
   - Support for `$shared` environment
   - Form-urlencoded body type
   - Multipart/form-data and file uploads
   - GraphQL request format
   - Authentication methods (Basic, Bearer, OAuth)
   - Request directives (`@no-redirect`, `@no-cookie-jar`, `@timeout`)
   - Response references and validation placeholders
   - Cookie management and redirect handling

2. **Partial Test Coverage (needs enhancement):**
   - Request naming with `# @name` directive
   - Comment styles (particularly `//` comments)
   - All HTTP methods (missing PATCH, HEAD, OPTIONS)
   - Request line format with HTTP version
   - JetBrains-specific placeholders
   - VS Code environment variable syntax
   - Response validation for all formats

3. **Next Steps:**
   - Create test files for each missing feature
   - Enhance existing tests for partial coverage
   - Ensure all tests use external `.http` files
   - Follow the pattern of at most one positive and one negative test per requirement
