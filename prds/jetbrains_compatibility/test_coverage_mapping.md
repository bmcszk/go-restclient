# HTTP Syntax Feature Coverage Analysis

This document maps requirements from `http_syntax.md` to existing tests in the codebase. It helps identify which features are already tested and implemented, and which ones need additional coverage.

## FR1: Request Structure Basics

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR1.1 | Support `.http` and `.rest` file extensions | `TestParseRequests_*` | `parser_test.go` | Complete | Current tests use `.http` files |
| FR1.2 | Support multiple requests delimited by `###` | `TestParseRequests_Separators` | `parser_test.go` | Complete | Tests separation of requests with `###` |
| FR1.3 | Support request naming | `TestParseRequests_Name` | `parser_test.go` | Partial | Tests for `### Name` but may need more tests for `# @name` directive |
| FR1.4 | Support comments using `#` and `//` | `TestParseRequests_IgnoreComments` | `parser_test.go` | Partial | Needs tests for `//` style comments |
| FR1.5 | Support all major HTTP methods | `TestParseRequests_Methods` | `parser_test.go` | Partial | Tests basic methods but not all required (PATCH, HEAD, OPTIONS) |
| FR1.6 | Support request line format | `TestParseRequests_RequestLine` | `parser_test.go` | Partial | Needs tests for optional HTTP version |
| FR1.7 | Parse headers | Various tests | `parser_test.go` | Complete | Various tests cover header parsing |
| FR1.8 | Handle standard body formats | Various tests | `parser_test.go` | Partial | Basic body handling tested, but not all content types |

## FR2: Environment Variables and Placeholders

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR2.1 | Support variable substitution in URL, headers, body | `TestSubstituteVariable*` | `client_execute_vars_test.go` | Complete | Tests for variables in various contexts |
| FR2.2 | Load environment variables from JSON files | `TestExecuteFile_WithHttpClientEnvJson` | `client_execute_vars_test.go` | Complete | Tests loading from `http-client.env.json` |
| FR2.3 | Support in-place variables | `TestInPlaceVariableDefinitions` | `parser_test.go` | Complete | Tests `@name = value` syntax |
| FR2.4 | Support `$shared` environment | Not found | - | Missing | Needs tests for shared environment |

## FR3: Dynamic System Variables

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR3.1 | Support common placeholders | Various tests | `client_execute_system_vars_test.go` | Partial | Tests for `{{$uuid}}`, `{{$timestamp}}`, `{{$datetime}}`, `{{$randomInt}}` |
| FR3.2 | Support JetBrains-specific placeholders | `TestExecuteFile_WithExtendedRandomSystemVariables` | `client_execute_vars_test.go` | Partial | Tests for some random functions but not all |
| FR3.3 | Support system environment access | `TestSubstituteDynamicSystemVariables_EnvVars` | `client_execute_vars_test.go` | Partial | Tests for `{{$env.VAR_NAME}}` but not for VS Code syntax |

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
| FR5.1 | Support Basic Authentication | Not found | - | Missing | Needs tests for both header and URL-based auth |
| FR5.2 | Support Bearer token authentication | Not found | - | Missing | Needs tests for Bearer token auth |
| FR5.3 | Support OAuth authentication | Not found | - | Missing | Needs tests for OAuth flow |

## FR6: Request Settings

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR6.1 | Support request-specific options | Not found | - | Missing | Needs tests for `@no-redirect`, `@no-cookie-jar` |
| FR6.2 | Support request timeout setting | Not found | - | Missing | Needs tests for `@timeout` directive |

## FR7: Response Handling and Validation

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR7.1 | Support defining expected responses | `TestValidateResponses_*` | `validator_body_test.go` | Partial | Basic validation tested, not all response formats |
| FR7.2 | Support response references | Not found | - | Missing | Needs tests for `{{requestName.response.body.field}}` |
| FR7.3 | Support response validation placeholders | Not found | - | Missing | Needs tests for validation placeholders |

## FR8: Request Imports

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR8.1 | Support importing requests | `TestParseRequestFile_Imports` | `parser_test.go` | Complete | Tests for basic import functionality |
| FR8.2 | Support correct variable scoping in imports | `TestParseRequestFile_Imports` | `parser_test.go` | Partial | Tests variable scoping but not extensively |
| FR8.3 | Detect and prevent circular imports | `TestParseRequestFile_Imports` | `parser_test.go` | Complete | Tests for circular imports |

## FR9: Cookies and Redirect Handling

| Requirement | Description | Tests | Files | Coverage Status | Notes |
|-------------|-------------|-------|-------|----------------|-------|
| FR9.1 | Support cookie management | Not found | - | Missing | Needs tests for cookie jar between requests |
| FR9.2 | Support redirect following | Not found | - | Missing | Needs tests for redirect handling and disabling |

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
