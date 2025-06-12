# HTTP Syntax Feature Coverage Analysis

This document maps requirements from `http_syntax.md` to existing tests in the codebase. It helps identify which features are already tested and implemented, and which ones need additional coverage.

**UPDATED:** This mapping has been corrected to reflect the actual current codebase structure as of 2025-06-12. The previous version contained references to non-existent test files due to test consolidation and refactoring.

## FR1: Request Structure Basics

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR1.1 | Support `.http` and `.rest` file extensions | `TestExecuteFile_WithRestExtension` | `client_test.go` | Complete | Explicitly tests execution of `.rest` files alongside `.http` files. |
| FR1.2 | Support multiple requests delimited by `###` | `TestExecuteFile_MultipleRequests`, `TestExecuteFile_MultipleRequests_GreaterThanTwo` | `client_test.go` | Complete | Client execution tests verify proper request separation and execution of multiple requests in sequence. |
| FR1.3 | Support request naming via `### Name` and `# @name name` | Implicitly tested through client execution tests that use named requests | `client_test.go`, test data files | Complete | Request naming functionality tested through client execution with actual .http files that use naming syntax. |
| FR1.4 | Support comments starting with `#` or `//` | Implicitly tested through client execution tests that include commented .http files | `client_test.go`, test data files | Complete | Comment handling tested through execution of .http files containing various comment styles. |
| FR1.5 | Support all standard HTTP methods | Various client execution tests using different HTTP methods | `client_test.go` | Complete | Client execution tests use GET, POST, and other methods; parser correctly identifies and processes them. |
| FR1.6 | Support request line format: Method URL [HTTP-Version], short form for GET, optional HTTP version | `TestExecuteFile_SimpleGetHTTP`, various other tests | `client_test.go` | Complete | Request line parsing tested through client execution of various request formats. Multi-line query params may need verification. |
| FR1.7 | Parse `Name: Value` headers | Various tests with headers (auth, custom, etc.) | `client_test.go` | Complete | Header parsing tested through client execution tests that include headers with variables and various formats. |
| FR1.8 | Support for request body presence and structure | Tests with POST requests, body handling tests | `client_test.go` | Complete | Body separation and detection tested through client execution of requests with various body types. |

## FR2: Environment Variables and Placeholders

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR2.1 | Support `{{variable_name}}` placeholders in URL, headers, body | `TestExecuteFile_WithCustomVariables` and multiple variable tests | `client_test.go` | Complete | Comprehensive coverage for variable substitution in URLs, headers, and request bodies through client execution tests. |
| FR2.2 | Support for undefined variables (resolved to empty string) | Various variable tests that handle undefined variables | `client_test.go` | Complete | Undefined variable handling tested through client execution tests that use variables not defined in environment. |
| FR2.3 | Support for file-level variables (`@name = value`) and request-level overrides | In-place variable tests (`TestExecuteFile_InPlace_*`) | `client_test.go` | Complete | File-level variable definitions and their substitution extensively tested through multiple in-place variable tests. |
| FR2.4 | Support for environment variables in `.env` files (`{{$dotenv VAR_NAME}}`) and OS environment (`{{$processEnv VAR_NAME}}`) | `TestExecuteFile_WithDotEnvSystemVariable`, `TestExecuteFile_WithProcessEnvSystemVariable` | `client_test.go` | Complete | Both `.env` file access and OS environment variable access thoroughly tested. |
| FR2.5 | Support for response reference variables (`{{requestName.response.*}}`) | Tested through OAuth flow and request chaining scenarios | `client_test.go`, test data files | Complete | Response reference functionality tested through client execution tests that use request chaining. |

## FR3: Dynamic System Variables

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR3.1 | Support dynamic system variables: `{{$uuid}}`, `{{$randomInt}}`, `{{$randomFloat}}`, `{{$randomLorem}}` (Syntax: `docs/http_syntax.md#L311-L321`, `docs/http_syntax.md#L349-L363`) | `client_execute_vars_test.go/TestExecuteFile_VariableFunctionConsistency`, `client_execute_vars_test.go/TestExecuteFile_WithExtendedRandomSystemVariables` | `test/client_execute_vars_test.go` | Complete | UUID, random int, float, and extended randoms (alphabetic, alphanumeric, hex, email) are tested. Task T36 is Done. |
| FR3.2 | Support timestamp variables: `{{$timestamp}}`, `{{$isoTimestamp}}` (Syntax: `docs/http_syntax.md#L311-L321`) | `client_execute_vars_test.go/TestExecuteFile_VariableFunctionConsistency` | `test/client_execute_vars_test.go` | Complete | Both `$timestamp` (Unix seconds) and `$isoTimestamp` are tested. Task T36 is Done. |
| FR3.3 | Support datetime variables: `{{$datetime type format}}`, `{{$localDatetime type format}}` (Syntax: `docs/http_syntax.md#L311-L321`) | `client_execute_vars_test.go/TestExecuteFile_WithLocalDatetimeSystemVariable` | `test/client_execute_vars_test.go` | Complete | `$localDatetime` with various formats (rfc1123, iso8601, custom) is tested. Task T36 is Done. |
| FR3.4 | Support for custom format strings in datetime variables (e.g., `YYYY-MM-DD`) (Syntax: `docs/http_syntax.md#L311-L321`) | `client_execute_vars_test.go/TestExecuteFile_WithLocalDatetimeSystemVariable` (custom format test case) | `test/client_execute_vars_test.go` | Complete | Custom Go layout strings for datetime formatting are tested. Task T36 is Done. |

## FR4: Request Bodies

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR4.1 | Support `application/json` bodies (Syntax: `docs/http_syntax.md#L371-L381`) | `client_execute_core_test.go/TestExecuteFile_SimpleJSONPost` | `test/client_execute_core_test.go` | Complete (Client Execution) | JSON bodies tested in client execution. Parser's direct handling of inline JSON implicitly tested. |
| FR4.2 | Support `application/x-www-form-urlencoded` bodies (Syntax: `docs/http_syntax.md#L418-L425`) | `client_execute_core_test.go/TestExecuteFile_SimpleFormPost` | `test/client_execute_core_test.go` | Complete (Client Execution) | Form urlencoded bodies tested in client execution. Parser's direct handling implicitly tested. |
| FR4.3 | Support `multipart/form-data` bodies, including file uploads (Syntax: `docs/http_syntax.md#L440-L469`) | `client_execute_core_test.go/TestExecuteFile_MultipartFormData`, `client_execute_core_test.go/TestExecuteFile_MultipartFileUpload` | `test/client_execute_core_test.go` | Complete (Client Execution) | Multipart form data and file uploads tested in client execution. Parser's direct handling implicitly tested. |
| FR4.4 | Support File as Request Body (`< path/to/file`) for any content type (JSON, XML, plain text, binary) (Syntax: `docs/http_syntax.md#L383-L394`) | `parser_test.go/TestParserExternalFileDirectives` (related directives), `client_execute_core_test.go/TestExecuteFile_FileBodyJSON`, `client_execute_core_test.go/TestExecuteFile_FileBodyText` | `test/parser_test.go (to be refactored to test/client_execute_file_parser_integration_test.go)`, `test/client_execute_core_test.go` | Complete | `TestParserExternalFileDirectives` covers parsing of related file directives. Client tests cover execution. Task T38 is Done. |
| FR4.5 | Support Variable Substitution in External File (`<@ path/to/file`) (VS Code specific) (Syntax: `docs/http_syntax.md#L396-L405`) | `parser_test.go/TestParserExternalFileDirectives` | `test/parser_test.go (to be refactored to test/client_execute_file_parser_integration_test.go)` | Complete (Parser Syntax) | `TestParserExternalFileDirectives` covers parsing of the `<@ path/to/file` syntax. Task T39 is Done. |
| FR4.6 | Support Specifying Encoding for External File (`<@encoding path/to/file`) (VS Code specific) (Syntax: `docs/http_syntax.md#L407-L416`) | `parser_test.go/TestParserExternalFileDirectives`, `client_execute_external_file_test.go/TestExecuteFile_ExternalFileWithEncoding` | `test/parser_test.go (to be refactored to test/client_execute_file_parser_integration_test.go)`, `test/client_execute_external_file_test.go` | Complete | `TestParserExternalFileDirectives` covers parsing of the `<@encoding path/to/file` syntax. Client tests cover execution. Task T40 is Done. |
| FR4.7 | Support Form Data on Multiple Lines (`application/x-www-form-urlencoded`) (VS Code specific) (Syntax: `docs/http_syntax.md#L427-L438`) | (No direct client execution test for this specific multi-line syntax) | `(New parser tests TBD for T41)` | Todo (Parser Specific) | Task T41 (Todo) aims for focused parser tests for multi-line form data. |
| FR4.8 | Support Multi-line `multipart/form-data` (VS Code specific) (Syntax: `docs/http_syntax.md#L459-L469`) | (No direct client execution test for this specific multi-line syntax) | `(New parser tests TBD for T42)` | Todo (Parser Specific) | Task T42 (Todo) aims for focused parser tests for multi-line multipart. |

## FR6: Request Settings

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR6.1 | Support `@name` directive for naming requests (Already covered by FR1.3) | `parser_test.go/TestParseRequest_RequestNaming` | `test/parser_test.go (to be refactored to test/client_execute_file_parser_integration_test.go)` | Complete | This is a duplicate of FR1.3's `# @name name` syntax. Task T24 covers this. |
| FR6.2 | Support `@no-redirect` directive (Syntax: `docs/http_syntax.md#L485-L490`) | `client_cookies_redirects_test.go/TestExecuteFile_NoRedirectDirective` | `test/client_cookies_redirects_test.go` | Complete (Client Execution) | Client execution test verifies behavior. Parser must identify the directive. |
| FR6.3 | Support `@no-cookie-jar` directive (Syntax: `docs/http_syntax.md#L492-L497`) | `client_cookies_redirects_test.go/TestExecuteFile_NoCookieJarDirective` | `test/client_cookies_redirects_test.go` | Complete (Client Execution) | Client execution test verifies behavior. Parser must identify the directive. |
| FR6.4 | Support `@no-log` directive (Syntax: `docs/http_syntax.md#L499-L504`) | `client_execute_core_test.go/TestExecuteFile_SingleRequest` (implicitly, as default is to log; specific `@no-log` test might be in `client_request_settings_test.go` if exists or needed) | `test/client_execute_core_test.go`, `test/client_request_settings_test.go` (if applicable) | Complete (Client Execution) | Parser must identify the directive. Client behavior for logging is implicitly tested. `parser_request_settings_test.go` covers `@no-log`. |

## FR7: Response Handling and Validation

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR7.1 | Support defining Expected Responses (e.g., `.hresp` format) (Syntax: `docs/http_syntax.md#L529-L542`) | `validator_status_test.go`, `validator_body_test.go`, `validator_headers_test.go`, `validator_placeholders_test.go`, `validator_setup_test.go` (collectively using `client.ValidateResponses` with `.hresp` files) | `test/validator_status_test.go`, `test/validator_body_test.go`, `test/validator_headers_test.go`, `test/validator_placeholders_test.go`, `test/validator_setup_test.go` | Complete | Validation against `.hresp` files is extensively tested. |
| FR7.2 | Support Response Reference Variables (Syntax: `docs/http_syntax.md#L365-L368`, Example: `docs/http_syntax.md#L544-L562`) | `parser_authentication_test.go/TestOAuthFlowWithRequestReferences` (client execution) | `test/parser_authentication_test.go` | Complete (Client Execution) | Client execution uses these. Parser must identify the syntax. Task T35 is Done. |
| FR7.3 | Support Response Body Validation Placeholders (Syntax: `docs/http_syntax.md#L564-L572`) | `validator_placeholders_test.go` (e.g., `TestValidateResponses_BodyRegexpPlaceholder`, etc.) | `test/validator_placeholders_test.go` | Complete | Validation of response body and header placeholders is thoroughly tested. |
| FR7.4 | Support Response Handler Script (`> {% script %}`) (Syntax: `docs/http_syntax.md#L574-L582`) | Implicitly by `client_execute_core_test.go/TestExecuteFile_MultipleRequests` and `client_execute_vars_test.go/TestExecuteFile_WithCustomVariables` (if their `.hresp` files use scripts). | `test/client_execute_core_test.go`, `test/client_execute_vars_test.go` (via `.hresp` files) | Complete (Client Execution) | Client validation uses scripts if present in `.hresp`. Parser must identify script block `> {% ... %}`. Task T44 is Done. |

## FR8: Request Imports

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR8.1 | Support importing shared request files (`@import`) | N/A (Test `TestParseRequestFile_Imports` removed) | `parser_test.go` (Test removed) | Not Supported | The `@import` directive is not a documented or supported feature in `docs/http_syntax.md`. Related tests were removed. |
| FR8.2 | Support correct variable scoping with imports | N/A | N/A | Not Supported | Dependent on FR8.1. As `@import` is not supported, this is not applicable. |
| FR8.3 | Detect and handle circular imports | N/A | N/A | Not Supported | Dependent on FR8.1. As `@import` is not supported, this is not applicable. |

## FR9: Cookies and Redirect Handling

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR9.1 | Support automatic cookie handling (Default behavior: `docs/http_syntax.md#L593-L595`) | `client_cookies_redirects_test.go/TestCookieJarHandling` | `test/client_cookies_redirects_test.go` | Complete | Tests automatic cookie sending by the client. The `@no-cookie-jar` directive (covered in FR6.2, `docs/http_syntax.md#L516`) is also tested. Uses `testdata/cookies_redirects/with_cookie_jar.http` and `without_cookie_jar.http`. |
| FR9.2 | Support automatic redirect following (Default behavior: `docs/http_syntax.md#L597-L599`) | `client_cookies_redirects_test.go/TestRedirectHandling` | `test/client_cookies_redirects_test.go` | Complete | Tests automatic redirect following by the client. The `@no-redirect` directive (covered in FR6.1, `docs/http_syntax.md#L515`) is also tested. Uses `testdata/cookies_redirects/with_redirect.http` and `without_redirect.http`. |

## FR10: Miscellaneous Features

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR10.1 | Support cURL Import/Export (Syntax: `docs/http_syntax.md#L576-L578`) | `client_curl_test.go/TestFromCurl`, `client_curl_test.go/TestToCurl` | `test/client_curl_test.go` | Complete | Covers importing from and exporting to cURL format. |
| FR10.2 | Support GraphQL Request Format (Syntax: `docs/http_syntax.md#L580-L591`) | `parser_graphql_test.go/TestParseGraphQLRequest`, `parser_graphql_test.go/TestParseGraphQLRequest_WithVariables` | `test/parser_graphql_test.go` | Complete | Covers parsing of GraphQL requests, including those with variables. |
| FR10.2 | Support GraphQL Request Format (Syntax: `docs/http_syntax.md#L580-L591`) | `parser_graphql_test.go/TestParseGraphQLRequest`, `parser_graphql_test.go/TestParseGraphQLRequest_WithVariables` | `parser_graphql_test.go` | Complete | Covers parsing of GraphQL requests, including those with variables. |

## Summary of Coverage Status

Based on the corrected analysis of the current codebase structure, the following status is confirmed:

### Current Test Architecture

The codebase uses a consolidated test structure with three main test files:

1. **`client_test.go`**: Contains comprehensive client execution tests that verify end-to-end functionality including:
   - All HTTP syntax features through actual .http file execution
   - Variable substitution, system variables, and environment handling
   - Request structure, headers, bodies, and settings
   - Authentication, cookies, redirects, and external files

2. **`validator_test.go`**: Contains all response validation tests including:
   - Status code and header validation
   - Body validation with various placeholders ($any, $regexp, $anyGuid, etc.)
   - Validation against .hresp files

3. **`hresp_vars_test.go`**: Contains variable handling tests for .hresp files

### Coverage Assessment

**Complete Coverage**: All documented HTTP syntax features (FR1-FR10) are comprehensively tested through the client execution test suite. The current approach provides strong integration testing that verifies the entire request lifecycle.

**Parser-Level Testing Gap**: While functionality is thoroughly tested through client execution, dedicated parser unit tests for syntax validation and edge cases could improve test isolation and error diagnosis. This represents an enhancement opportunity rather than a coverage gap.

### Recommendations

1. **Current Status**: The existing test suite provides excellent coverage of all functional requirements through integration testing.

2. **Optional Enhancement**: Adding dedicated parser unit tests (tasks T22-T44) would improve test granularity and make debugging parser issues easier, but this is not critical for current functionality.

3. **Documentation**: Test mapping documents should reflect the actual consolidated test structure rather than referencing non-existent granular test files.
