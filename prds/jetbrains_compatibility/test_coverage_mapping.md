# HTTP Syntax Feature Coverage Analysis

This document maps requirements from `http_syntax.md` to existing tests in the codebase. It helps identify which features are already tested and implemented, and which ones need additional coverage.

## FR1: Request Structure Basics

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR1.1 | Support `.http` and `.rest` file extensions (Implicitly defined by usage of these files throughout `docs/http_syntax.md`) | `client_execute_external_file_test.go/TestExecuteFile_WithRestExtension` | `test/client_execute_external_file_test.go` | Complete | `TestExecuteFile_WithRestExtension` explicitly verifies execution of a `.rest` file. |
| FR1.2 | Support multiple requests delimited by `###` (Syntax: `docs/http_syntax.md#L200-L216`) | `client_execute_core_test.go/TestExecuteFile_MultipleRequests`, `client_execute_core_test.go/TestExecuteFile_MultipleRequests_GreaterThanTwo`, (Implicitly by tests in `parser_test.go`) | `test/client_execute_core_test.go`, `test/parser_test.go (to be refactored to test/client_execute_file_parser_integration_test.go)` | Complete | Client execution tests cover multiple requests (Task T22). Parser must handle separation for these to work. | 
| FR1.3 | Support request naming via `### Name` (Syntax: `docs/http_syntax.md#L218-L228`) and `# @name name` (Syntax: `docs/http_syntax.md#L232-L246`) | `parser_test.go/TestParseRequest_RequestNaming` | `test/parser_test.go (to be refactored to test/client_execute_file_parser_integration_test.go)` | Complete | `TestParseRequest_RequestNaming` covers both `### Name` (Task T23) and `# @name name` (Task T24) syntaxes and precedence. | 
| FR1.4 | Support comments starting with `#` or `//` (Syntax: `docs/http_syntax.md#L248-L255`) | `parser_test.go/TestParseRequest_SlashStyleComments` (for `//`). `#` comments implicitly handled/ignored in various client execution tests. | `test/parser_test.go (to be refactored to test/client_execute_file_parser_integration_test.go)` | Complete | `TestParseRequest_SlashStyleComments` covers `//`. `#` comments are generally ignored by parser. Task T25 is Done. | 
| FR1.5 | Support all standard HTTP methods (Syntax: `docs/http_syntax.md#L138-L147`) | Implicitly by various client execution tests (e.g., `client_execute_core_test.go/TestExecuteFile_SingleRequest` (GET), `client_execute_core_test.go/TestExecuteFile_MultipleRequests` (POST)) | `test/client_execute_core_test.go` and others. | Complete (Client Execution) | Client execution tests use various methods; parser must correctly identify them. Task T26 is Done. | 
| FR1.6 | Support request line format: Method URL [HTTP-Version] (Syntax: `docs/http_syntax.md#L130-L136`), short form for GET (Syntax: `docs/http_syntax.md#L149-L155`), optional HTTP version (Syntax: `docs/http_syntax.md#L192-L198`). VSCode multi-line query params (Syntax: `docs/http_syntax.md#L157-L166`) | `parser_test.go/TestParseRequest_ShortFormGET` (for short-form GET). Full request line & optional HTTP version implicitly covered by all client execution tests. | `test/parser_test.go (to be refactored to test/client_execute_file_parser_integration_test.go)` | Partial (Multi-line query params need dedicated tests) | `TestParseRequest_ShortFormGET` covers short-form GET (Task T31 Done). Standard request line (Task T27 Done) and optional HTTP version (Task T28 Done) are implicitly covered. Multi-line query params (Task T32 Todo) need specific parser tests. | 
| FR1.7 | Parse `Name: Value` headers (Syntax: `docs/http_syntax.md#L168-L176`) | Implicitly by various client execution tests (e.g., `client_execute_vars_test.go/TestExecuteFile_WithCustomVariables` uses headers with variables). | `test/client_execute_vars_test.go` and others. | Complete (Client Execution) | Headers are parsed in client execution tests. Task T29 is Done. | 
| FR1.8 | Support for request body presence and structure (separated by a blank line after headers) (Syntax: `docs/http_syntax.md#L178-L190`). Specific body *formats* are FR4. | Implicitly by client execution tests with bodies (e.g., `client_execute_core_test.go/TestExecuteFile_MultipleRequests` (POST)). | `test/client_execute_core_test.go` and others. | Complete (Client Execution) | Parser must correctly identify body start. Task T30 is Done. |

## FR2: Environment Variables and Placeholders

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR2.1 | Support `{{variable_name}}` placeholders in URL, headers, body (Syntax: `docs/http_syntax.md#L257-L271`) | `client_execute_vars_test.go/TestExecuteFile_WithCustomVariables` and others in this file. | `test/client_execute_vars_test.go` | Complete | Comprehensive coverage for variable substitution in various request parts. Task T33 is Done. |
| FR2.2 | Support for undefined variables (resolved to empty string) (Syntax: `docs/http_syntax.md#L273-L279`) | `client_execute_vars_test.go/TestExecuteFile_WithCustomVariables` (scenario SCENARIO-LIB-013-005) | `test/client_execute_vars_test.go` | Complete | Verified that undefined variables resolve to empty strings. Task T34 is Done. |
| FR2.3 | Support for file-level variables (`@name = value`) and request-level overrides (Syntax: `docs/http_syntax.md#L281-L309`) | `client_execute_vars_test.go/TestExecuteFile_WithCustomVariables` (demonstrates file-level and implies override if used within a request block) | `test/client_execute_vars_test.go` | Complete | File-level variables and their substitution are tested. Request-level overrides are implicitly part of this. |
| FR2.4 | Support for environment variables in `.env` files (`{{$dotenv VAR_NAME}}`) and OS environment (`{{$processEnv VAR_NAME}}`) (Syntax: `docs/http_syntax.md#L323-L336`, `docs/http_syntax.md#L338-L347`) | `client_execute_vars_test.go/TestExecuteFile_WithDotEnvSystemVariable`, `client_execute_vars_test.go/TestExecuteFile_WithProcessEnvSystemVariable` | `test/client_execute_vars_test.go` | Complete | Both `$dotenv` and `$processEnv` are tested. Task T37 is Done. |
| FR2.5 | Support for response reference variables (`{{requestName.response.*}}`) (Syntax: `docs/http_syntax.md#L365-L368`) | `parser_authentication_test.go/TestOAuthFlowWithRequestReferences` (client execution) | `test/parser_authentication_test.go` | Complete (Client Execution) | Client execution uses these. Parser must identify the syntax. Task T35 is Done. (Note: Task tracker T35 maps this to FR2.5, http_syntax.md has this under 'Dynamic Variables') |

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

## Summary of Coverage Gaps

Based on the detailed mapping of functional requirements (FR1-FR10) against `docs/http_syntax.md` and existing tests, the following areas have been identified:

1.  **Parser-Level Test Enhancement:**
    *   While many features have client execution tests and some existing parser tests, a comprehensive suite of focused parser unit tests was identified as an area for enhancement. Tasks T22-T44 have been created to address this. These tasks aim to ensure robust, direct testing of the parser's ability to handle various specific syntax elements outlined in `docs/http_syntax.md` and covered in FR1, FR2, FR3, FR4, and FR6. This includes detailed testing for request separation, naming, comments, HTTP methods, request line variations, header parsing, body detection, variable placeholders (environment, file-level, system), external file body directives, multi-line form data, and request setting directives with their placement.
    *   Addressing these tasks will improve the confidence in the parser's correctness and adherence to the specified syntax, complementing existing client-level and broader parser tests.

2.  **Previously Noted Partial Coverage:**
    *   Several items previously marked as 'Partial' (e.g., for VSCode multi-line query params, specific header parsing scenarios, `{{$env VAR_NAME}}` syntax, comprehensive variable scoping, Faker library coverage) are now explicitly targeted for improved parser-level testing through tasks T22-T44. While client-side execution tests for some of these might still be relevant for full end-to-end validation, the new parser tasks will ensure the syntax itself is correctly understood by the parser.

3.  **Ongoing and Future Work:**
    *   Focus can shift to implementing these new parser tests (T22-T44).
    *   The `LoadExpectedResponseFromHTTPFile` TODO in `validator.go` remains a potential future enhancement for response validation.

The primary focus of this mapping document, especially with the recent updates, is to ensure that the parser can correctly interpret all documented syntax. End-to-end client execution tests, while also important, are a separate layer of testing.
