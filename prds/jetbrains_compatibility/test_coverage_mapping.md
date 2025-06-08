# HTTP Syntax Feature Coverage Analysis

This document maps requirements from `http_syntax.md` to existing tests in the codebase. It helps identify which features are already tested and implemented, and which ones need additional coverage.

## FR1: Request Structure Basics

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR1.1 | Support `.http` and `.rest` file extensions (Implicitly defined by usage of these files throughout `docs/http_syntax.md`) | `TestParseRequests_*` (implicit by usage in test data) | `parser_test.go` | Partial | Code comment in `parser.go` mentions `.rest` but no explicit tests exist for `.rest` files. Current tests use `.http` files. |
| FR1.2 | Support multiple requests delimited by `###` (Syntax: `docs/http_syntax.md#L200-L216`) | `parser_test.go/TestParseRequests_Separators`, `parser_test.go/TestParseRequests_IgnoreEmptyBlocks` (subtests: SCENARIO-LIB-028-002, SCENARIO-LIB-028-003, SCENARIO-LIB-028-006), `parser_test.go/TestParseExpectedResponses_Simple` (subtest: Multiple responses) | `parser_test.go` | Complete | Tests separation of requests with `###`, including with comments and empty blocks. |
| FR1.3 | Support request naming via `### Name` (Syntax: `docs/http_syntax.md#L218-L228`) and `# @name name` (Syntax: `docs/http_syntax.md#L232-L246`) | `parser_test.go/TestParseRequests_Name`, `parser_request_settings_test.go/TestParseRequest_Directives` (subtest: request name directive) | `parser_test.go`, `parser_request_settings_test.go` | Complete | Covers both `### Name` and `# @name name` styles for naming requests. |
| FR1.4 | Support comments starting with `#` or `//` (Syntax: `docs/http_syntax.md#L248-L255`) | `parser_test.go/TestParseRequests_SeparatorComments`, `parser_test.go/TestParseRequests_IgnoreEmptyBlocks` (various subtests), `parser_test.go/TestParseExpectedResponses_SeparatorComments` | `parser_test.go` | Partial | Good coverage for `#` style comments, especially around separators. `//` primarily tested as part of directives or on same line as request elements. General `//` comments on their own lines need explicit tests. |
| FR1.5 | Support all standard HTTP methods (Syntax: `docs/http_syntax.md#L138-L147`) | `client_execute_methods_test.go/TestExecuteFile_ValidMethods`, various `parser_test.go/TestParseRequests_*` (implicitly for GET/POST/PUT) | `client_execute_methods_test.go`, `parser_test.go` | Complete | `TestExecuteFile_ValidMethods` covers GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS. |
| FR1.6 | Support request line format: Method URL [HTTP-Version] (Syntax: `docs/http_syntax.md#L130-L136`), short form for GET (Syntax: `docs/http_syntax.md#L149-L155`), optional HTTP version (Syntax: `docs/http_syntax.md#L192-L198`). VSCode multi-line query params (Syntax: `docs/http_syntax.md#L157-L166`) | `parser_test.go/TestParseRequests_Simple` (subtests: basic GET, POST; GET request short form), `parser_test.go/TestParseRequest_HTTPVersion` | `parser_test.go` | Partial | Basic request line, short GET, and HTTP version parsing are covered. VS Code-specific multi-line query parameters are not explicitly tested. |
| FR1.7 | Parse `Name: Value` headers (Syntax: `docs/http_syntax.md#L168-L176`) | `parser_test.go/TestParseRequest_Headers`, `parser_test.go/TestParseRequest_EmptyHeaderValue`, `parser_test.go/TestParseRequest_HeaderCaseInsensitive`, `parser_test.go/TestParseRequest_DuplicateHeader` | `parser_test.go` | Complete | Covers various header parsing scenarios including empty values, case-insensitivity, and duplicate headers. |
| FR1.8 | Support for request body presence and structure (separated by a blank line after headers) (Syntax: `docs/http_syntax.md#L178-L190`). Specific body *formats* are FR4. | `parser_test.go/TestParseRequest_BodyContent`, `parser_test.go/TestParseRequest_BodyWithLeadingEmptyLines`, `parser_test.go/TestParseRequest_NoBodyButBlankLine` | `parser_test.go` | Complete | Tests presence of body, handling of blank lines. Specific content types (JSON, XML, etc.) are detailed under FR4. |

## FR2: Environment Variables and Placeholders

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR2.1 | Support variable substitution of file and environment variables in URL, headers, body (Syntax: `docs/http_syntax.md#L268-L271`, `docs/http_syntax.md#L299-L304`) | `parser_test.go/TestParseRequestFile_VariableScoping`, `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_SimpleVariableInURL`, `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInHeader`, `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInBody` | `parser_test.go`, `client_execute_inplace_vars_test.go` | Complete | Covers substitution from file-level (`@`) and environment variables (`{{ }}`) in various parts of the request. |
| FR2.2 | Load environment variables from `http-client.env.json` (Syntax: `docs/http_syntax.md#L273-L297`) | `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson` (multiple subtests covering different scenarios like env selection, overrides) | `client_execute_vars_test.go` | Complete | Tests loading from `http-client.env.json`, including environment selection and variable precedence. |
| FR2.3 | Support file-level variables defined with `@name = value` (Syntax: `docs/http_syntax.md#L259-L271`) | `parser_test.go/TestParseRequestFile_VariableScoping`, `client_execute_inplace_vars_test.go` (various tests using file-level vars) | `parser_test.go`, `client_execute_inplace_vars_test.go` | Complete | Tests defining and using file-level variables. |
| FR2.4 | Support `$shared` environment section in `http-client.env.json` and `http-client.private.env.json` for common/override variables (Syntax: `docs/http_syntax.md#L306-L332`) | `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson` (subtests: SCENARIO-LIB-010-002, SCENARIO-LIB-010-005, SCENARIO-LIB-010-010) | `client_execute_vars_test.go` | Complete | Tested via `http-client.env.json` and `http-client.private.env.json` usage, including `$shared` section and precedence. |

## FR3: Dynamic System Variables

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR3.1 | Support UUID/GUID generation: `{{$guid}}`, `{{$uuid}}`, `{{$random.uuid}}` (Syntax: `docs/http_syntax.md#L337-L338`) | `client_execute_system_vars_test.go/TestExecuteFile_WithGuidSystemVariable`, `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByGuid` | `client_execute_system_vars_test.go`, `client_execute_inplace_vars_test.go` | Complete | Covers various UUID/GUID placeholders. |
| FR3.2 | Support Date/Time variables: `{{$timestamp}}`, `{{$isoTimestamp}}`, `{{$datetime format}}`, `{{$localDatetime format}}` (Syntax: `docs/http_syntax.md#L340-L350`) | `client_execute_system_vars_test.go/TestExecuteFile_WithTimestampSystemVariable`, `client_execute_system_vars_test.go/TestExecuteFile_WithIsoTimestampSystemVariable`, `client_execute_system_vars_test.go/TestExecuteFile_WithDatetimeSystemVariables`, `client_execute_vars_test.go/TestExecuteFile_WithLocalDatetimeSystemVariable`, `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByTimestamp`, `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByDateTime` | `client_execute_system_vars_test.go`, `client_execute_vars_test.go`, `client_execute_inplace_vars_test.go` | Complete | Covers various date/time placeholders and formats. |
| FR3.3 | Support Random value generation: `{{$randomInt}}`, `{{$randomInt min max}}`, `{{$random.integer}}`, `{{$random.float}}`, `{{$random.alphabetic}}`, `{{$random.alphanumeric}}`, `{{$random.hexadecimal}}` (Syntax: `docs/http_syntax.md#L351-L359`). JetBrains Faker library variables (`{{$random.email}}`, etc.) are partially covered. | `client_execute_system_vars_test.go/TestExecuteFile_WithRandomIntSystemVariable`, `client_execute_vars_test.go/TestExecuteFile_WithExtendedRandomSystemVariables`, `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByRandomInt` | `client_execute_system_vars_test.go`, `client_execute_vars_test.go`, `client_execute_inplace_vars_test.go` | Partial | Standard random value generators well covered. JetBrains Faker library (`{{$random.email}}`, etc. from `docs/http_syntax.md#L61-L82`) has limited/implicit coverage via `TestExecuteFile_WithExtendedRandomSystemVariables` for email but not other Faker categories. |
| FR3.4 | Support Environment Access variables: `{{$processEnv NAME}}`, `{{$env.NAME}}`, `{{$dotenv NAME}}` (Syntax: `docs/http_syntax.md#L360-L363`) | `client_execute_vars_test.go/TestExecuteFile_WithProcessEnvSystemVariable`, `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByOsEnvVariable`, `hresp_vars_test.go/TestResolveAndSubstitute` (subtests for `$processEnvVariable`, `$dotenv system variable`) | `client_execute_vars_test.go`, `client_execute_inplace_vars_test.go`, `hresp_vars_test.go` | Complete | Covers accessing OS and .env variables. `{{$env.NAME}}` (JetBrains specific) implicitly covered by `{{$processEnv}}` for go-restclient. |

## FR4: Request Bodies

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR4.1 | Support `application/json` bodies (Syntax: `docs/http_syntax.md#L371-L381`) | `parser_test.go/TestParseRequest_BodyContent` (subtest: JSON body), `client_execute_test.go/TestExecuteFile_SimpleJSONPost` | `parser_test.go`, `client_execute_test.go` | Complete | JSON bodies tested in various contexts. |
| FR4.2 | Support `application/x-www-form-urlencoded` bodies (Syntax: `docs/http_syntax.md#L418-L425`) | `parser_test.go/TestParseRequest_BodyContent` (subtest: Form urlencoded body), `client_execute_test.go/TestExecuteFile_SimpleFormPost` | `parser_test.go`, `client_execute_test.go` | Complete | Form urlencoded bodies are tested. |
| FR4.3 | Support `multipart/form-data` bodies, including file uploads (Syntax: `docs/http_syntax.md#L440-L469`) | `parser_test.go/TestParseRequest_MultipartFormData`, `parser_test.go/TestParseRequest_MultipartFormDataWithFile`, `client_execute_test.go/TestExecuteFile_MultipartFormData`, `client_execute_test.go/TestExecuteFile_MultipartFileUpload` | `parser_test.go`, `client_execute_test.go` | Complete | Multipart form data and file uploads are tested. |
| FR4.4 | Support File as Request Body (`< path/to/file`) for any content type (JSON, XML, plain text, binary) (Syntax: `docs/http_syntax.md#L383-L394`) | `parser_test.go/TestParseRequestFile_FileBody`, `client_execute_test.go/TestExecuteFile_FileBodyJSON`, `client_execute_test.go/TestExecuteFile_FileBodyText` | `parser_test.go`, `client_execute_test.go` | Complete | Covers reading body from external files for various content types. `docs/http_syntax.md` notes this works for XML and binary too. |
| FR4.5 | Support Variable Substitution in External File (`<@ path/to/file`) (VS Code specific) (Syntax: `docs/http_syntax.md#L396-L405`) | `parser_test.go/TestParseRequestFile_FileBodyWithVariables` | `parser_test.go` | Complete | Tests parsing of `<@ file` syntax for variable substitution in external files. Execution with actual substitution may need client-side tests. |
| FR4.6 | Support Specifying Encoding for External File (`<@encoding path/to/file`) (VS Code specific) (Syntax: `docs/http_syntax.md#L407-L416`) | Not found | - | Missing | No specific tests found for parsing or handling `<@encoding file` syntax. |
| FR4.7 | Support Form Data on Multiple Lines (`application/x-www-form-urlencoded`) (VS Code specific) (Syntax: `docs/http_syntax.md#L427-L438`) | `parser_test.go/TestParseRequest_BodyContent` (subtest: Form urlencoded body multiline) | `parser_test.go` | Complete | Multi-line form data parsing is tested. |

## FR5: Authentication

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR5.1 | Support Basic Authentication (Header: `docs/http_syntax.md#L476-L478`, URL: `docs/http_syntax.md#L482-L487`) | `parser_authentication_test.go/TestBasicAuthHeader`, `parser_authentication_test.go/TestBasicAuthURL` | `parser_authentication_test.go` | Complete | Covers both header (`Authorization: Basic ...`) and URL-based (`user:pass@host`) basic authentication. |
| FR5.2 | Support Bearer token authentication (Syntax: `docs/http_syntax.md#L491-L494`) | `parser_authentication_test.go/TestBearerTokenAuth` | `parser_authentication_test.go` | Complete | Covers `Authorization: Bearer <token>` syntax. |
| FR5.3 | Support using Response References for Authentication Tokens (e.g., for OAuth flows) (General syntax for response refs: `docs/http_syntax.md#L365-L368`) | `parser_authentication_test.go/TestOAuthFlowWithRequestReferences` | `parser_authentication_test.go` | Complete | Covers parsing of requests that use response references for tokens (e.g., `Authorization: Bearer {{getToken.response.body.access_token}}`), a common pattern in OAuth. Full OAuth protocol support is not explicitly detailed as a standalone feature in `docs/http_syntax.md` authentication section. |

## FR6: Request Settings

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR6.1 | Support `@no-redirect` directive (Syntax: `docs/http_syntax.md#L515`) | `parser_request_settings_test.go/TestParseRequest_Directives` (subtest: no-redirect directive) | `parser_request_settings_test.go` | Complete | Covers parsing of `@no-redirect` directive. |
| FR6.2 | Support `@no-cookie-jar` directive (Syntax: `docs/http_syntax.md#L516`) | `parser_request_settings_test.go/TestParseRequest_Directives` (subtest: no-cookie-jar directive) | `parser_request_settings_test.go` | Complete | Covers parsing of `@no-cookie-jar` directive. |
| FR6.3 | Support `@no-log` directive (Syntax: `docs/http_syntax.md#L517`) | `parser_request_settings_test.go/TestParseRequest_Directives` (subtest: no-log directive) | `parser_request_settings_test.go` | Complete | Covers parsing of `@no-log` directive. |
| FR6.4 | Support `@timeout <milliseconds>` directive (Syntax: `docs/http_syntax.md#L518`, `docs/http_syntax.md#L520-L525`) | `parser_request_settings_test.go/TestParseRequest_Directives` (subtest: timeout directive) | `parser_request_settings_test.go` | Complete | Covers parsing of `@timeout <milliseconds>` directive. |

## FR7: Response Handling and Validation

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR7.1 | Support defining Expected Responses (e.g., `.hresp` format) (Syntax: `docs/http_syntax.md#L529-L542`) | `parser_response_validation_test.go/TestParseExpectedResponses_Valid`, `validator_body_test.go/*` | `parser_response_validation_test.go`, `validator_body_test.go` | Partial | Covers parsing of expected response status, headers. `validator_body_test.go` specifically tests body content matching. Full validation logic might be split. |
| FR7.2 | Support Response Reference Variables (Syntax: `docs/http_syntax.md#L365-L368`, Example: `docs/http_syntax.md#L544-L562`) | `parser_test.go/TestParseRequestFile_MultipleRequestsChained`, `parser_authentication_test.go/TestOAuthFlowWithRequestReferences` | `parser_test.go`, `parser_authentication_test.go` | Complete | Covers parsing of response references like `{{reqName.response.body.field}}`. |
| FR7.3 | Support Response Body Validation Placeholders (Syntax: `docs/http_syntax.md#L564-L572`) | `parser_response_validation_test.go/TestParseExpectedResponses_WithPlaceholders` | `parser_response_validation_test.go` | Complete | Covers parsing and recognition of placeholders like `{{$any}}`, `{{$regexp}}`, etc., in expected response bodies. |

## FR8: Request Imports

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR8.1 | Support importing shared request files (`@import`) | N/A (Test `TestParseRequestFile_Imports` removed) | `parser_test.go` (Test removed) | Not Supported | The `@import` directive is not a documented or supported feature in `docs/http_syntax.md`. Related tests were removed. |
| FR8.2 | Support correct variable scoping with imports | N/A | N/A | Not Supported | Dependent on FR8.1. As `@import` is not supported, this is not applicable. |
| FR8.3 | Detect and handle circular imports | N/A | N/A | Not Supported | Dependent on FR8.1. As `@import` is not supported, this is not applicable. |

## FR9: Cookies and Redirect Handling

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR9.1 | Support automatic cookie handling (Default behavior: `docs/http_syntax.md#L593-L595`) | `client_cookies_redirects_test.go/TestCookieJarHandling` | `client_cookies_redirects_test.go` | Complete | Tests automatic cookie sending by the client. The `@no-cookie-jar` directive (covered in FR6.2, `docs/http_syntax.md#L516`) is also tested. Uses `testdata/cookies_redirects/with_cookie_jar.http` and `without_cookie_jar.http`. |
| FR9.2 | Support automatic redirect following (Default behavior: `docs/http_syntax.md#L597-L599`) | `client_cookies_redirects_test.go/TestRedirectHandling` | `client_cookies_redirects_test.go` | Complete | Tests automatic redirect following by the client. The `@no-redirect` directive (covered in FR6.1, `docs/http_syntax.md#L515`) is also tested. Uses `testdata/cookies_redirects/with_redirect.http` and `without_redirect.http`. |

## FR10: Miscellaneous Features

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR10.1 | Support cURL Import/Export (Syntax: `docs/http_syntax.md#L576-L578`) | `client_curl_test.go/TestFromCurl`, `client_curl_test.go/TestToCurl` | `client_curl_test.go` | Complete | Covers importing from and exporting to cURL format. |
| FR10.2 | Support GraphQL Request Format (Syntax: `docs/http_syntax.md#L580-L591`) | `parser_graphql_test.go/TestParseGraphQLRequest`, `parser_graphql_test.go/TestParseGraphQLRequest_WithVariables` | `parser_graphql_test.go` | Complete | Covers parsing of GraphQL requests, including those with variables. |

## Summary of Coverage Gaps

Based on the detailed mapping of functional requirements (FR1-FR10) against `docs/http_syntax.md` and existing tests, the following areas have been identified:

1.  **Missing Test Coverage:**
    *   **FR4.5.3:** VS Code specific: Specify encoding for external file (`<@encoding path/to/file`) (Syntax: `docs/http_syntax.md#L457-L460`). Currently, no tests cover this specific directive.

2.  **Areas with Partial Coverage or Requiring Further Review:**
    *   **FR7.1:** Support defining Expected Responses (e.g., `.hresp` format) (Syntax: `docs/http_syntax.md#L529-L542`). While parsing of expected response components is tested, the note "Full validation logic might be split" suggests that end-to-end validation across all scenarios or deeper validation logic might warrant further review or enhanced testing beyond basic parsing.

All other features detailed in `docs/http_syntax.md` (and covered in FR1-FR10) appear to have corresponding parser tests. The primary focus of this mapping has been on ensuring that the parser can correctly interpret the documented syntax. End-to-end client execution tests are a separate concern.
