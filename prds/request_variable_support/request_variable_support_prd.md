# PRD: Request File Variable Support

**Document Version:** 2.0
**Date:** 2025-06-09
**Status:** Implemented

## 1. Introduction

This document outlines the product requirements for Variable Support in Request Files within the `go-restclient` library. This feature allows users to define and use variables within `.http` request files for dynamic content generation and configuration, including system-generated values, environment variables, and user-provided custom variables.

**IMPLEMENTATION STATUS: ✅ FULLY IMPLEMENTED**
All variable support features for `.http` request files are implemented and working, with comprehensive support for all documented variable types, precedence rules, and dynamic system variables.

## 2. Goals

- To enable dynamic data in request files through variable substitution.
- To support various sources for variables: system-generated, environment, `.env` files, and programmatic custom inputs.
- To provide a clear syntax for defining and using variables.
- To allow overriding of variables from different sources.

## 3. User Stories / Functional Requirements

| ID          | Requirement / User Story                                                                                                    | Status |
| :---------- | :-------------------------------------------------------------------------------------------------------------------------- | :------ |
| US-RVS-001  | As a developer, I want the library to generally support variables in request files (e.g., for hostnames, tokens). (REQ-LIB-007)      | ✅ IMPLEMENTED |
| US-RVS-002  | As a developer, I want to define and use custom variables within the request file. (REQ-LIB-013)                                | ✅ IMPLEMENTED |
| US-RVS-003  | As a developer, I want to use a `{{$guid}}` system variable that generates a new GUID for each substitution. (REQ-LIB-014)         | ✅ IMPLEMENTED |
| US-RVS-004  | As a developer, I want to use a `{{$randomInt min max}}` system variable for a random integer in a range. (REQ-LIB-015)           | ✅ IMPLEMENTED |
| US-RVS-005  | As a developer, I want to use a `{{$timestamp}}` system variable for the current UTC Unix epoch seconds. (REQ-LIB-016)             | ✅ IMPLEMENTED |
| US-RVS-006  | As a developer, I want to use a `{{$datetime format}}` system variable for a formatted current UTC datetime. (REQ-LIB-017)         | ✅ IMPLEMENTED |
| US-RVS-007  | As a developer, I want to use a `{{$localDatetime format}}` system variable for a formatted current local datetime. (REQ-LIB-018) | ✅ IMPLEMENTED |
| US-RVS-008  | As a developer, I want to use `{{$processEnv variableName}}` to substitute system environment variables. (REQ-LIB-019)             | ✅ IMPLEMENTED |
| US-RVS-009  | As a developer, I want to use `{{$dotenv variableName}}` to substitute variables from a `.env` file. (REQ-LIB-020)                   | ✅ IMPLEMENTED |
| US-RVS-010  | As a developer, I want to programmatically provide custom variables that can override variables from other sources. (REQ-LIB-021) | ✅ IMPLEMENTED |

## 4. Acceptance Criteria

- **AC-RVS-001.1:** Variables defined using `{{variableName}}` syntax are substituted correctly before sending the request. ✅ IMPLEMENTED
- **AC-RVS-002.1:** `{{$guid}}` / `{{$uuid}}` substitutes a valid unique GUID. ✅ IMPLEMENTED
- **AC-RVS-003.1:** `{{$randomInt min max}}` substitutes an integer within the specified inclusive range. ✅ IMPLEMENTED
- **AC-RVS-004.1:** `{{$timestamp}}` substitutes the current Unix epoch time in seconds (UTC). ✅ IMPLEMENTED
- **AC-RVS-005.1:** `{{$datetime format}}` substitutes the current UTC datetime formatted according to the Go layout string provided in `format`. ✅ IMPLEMENTED
- **AC-RVS-006.1:** `{{$localDatetime format}}` substitutes the current local datetime formatted according to the Go layout string provided in `format`. ✅ IMPLEMENTED
- **AC-RVS-007.1:** `{{$processEnv variableName}}` correctly substitutes the value of the specified environment variable; if not set, substitution results in an empty string. ✅ IMPLEMENTED
- **AC-RVS-008.1:** `{{$dotenv variableName}}` correctly substitutes the value from a `.env` file in the request directory; if not found, substitution results in an empty string. ✅ IMPLEMENTED
- **AC-RVS-009.1:** Custom variables provided programmatically via `WithVars` client option take precedence over variables defined within the file or loaded from `.env` / system environment if names conflict. ✅ IMPLEMENTED
- **AC-RVS-010.1:** Undefined variables use fallback values if provided, otherwise result in empty string replacement. ✅ IMPLEMENTED

### Additional Implemented Features
- **AC-RVS-011.1:** Support for `{{$isoTimestamp}}` ISO-8601 formatted timestamps. ✅ IMPLEMENTED
- **AC-RVS-012.1:** Support for JetBrains-style random generators (`{{$random.integer()}}`, `{{$random.float()}}`, `{{$random.alphabetic()}}`, etc.). ✅ IMPLEMENTED
- **AC-RVS-013.1:** Support for `{{$env.VAR_NAME}}` JetBrains-style environment variable access. ✅ IMPLEMENTED
- **AC-RVS-014.1:** Support for predefined datetime formats (`rfc1123`, `iso8601`) and custom Go layout strings. ✅ IMPLEMENTED
- **AC-RVS-015.1:** Support for variable fallback syntax `{{variable | fallback_value}}`. ✅ IMPLEMENTED
- **AC-RVS-016.1:** Support for `http-client.env.json` environment files with environment selection. ✅ IMPLEMENTED

## 5. Non-Functional Requirements

- **NFR-RVS-001:** Variable substitution should be performant. (NFR-LIB-001 related - ease of use implies reasonable performance)
- **NFR-RVS-002:** Documentation for all supported variables, their syntax, and precedence rules must be clear. (NFR-LIB-002)
- **NFR-RVS-003:** The variable substitution logic must have good unit test coverage. (NFR-LIB-003)

## 6. Out of Scope

- Complex scripting or expressions within variables. ❌ NOT IMPLEMENTED
- Pre-request and post-request variable processing scripts. ❌ NOT IMPLEMENTED

## 7. Implementation Details

### Core Implementation Files
- **`client.go`**: Variable resolution methods and client-level variable storage
- **`parser.go`**: Variable extraction from `.http` files and substitution during parsing
- **`variables.go`**: Core variable resolution logic and system variable generation

### Key Features Implemented
1. **Variable Definition**: `@name = value` syntax with immediate resolution of right-hand side
2. **Variable Substitution**: Complete substitution system with proper precedence
3. **System Variables**: Comprehensive set of dynamic system variables
4. **Environment Integration**: Support for OS environment, `.env` files, and `http-client.env.json`
5. **Client Configuration**: `WithVars` option for programmatic variable provision
6. **Request-Scoped Variables**: Consistent system variable values within single request execution

### Variable Precedence (Implemented)
1. Client programmatic variables (`WithVars`)
2. File-scoped variables (`@name = value`)
3. Request-scoped system variables
4. OS environment variables
5. `.env` file variables
6. Fallback values (`{{var | fallback}}`)

### System Variables Implemented
- **UUIDs**: `{{$uuid}}`, `{{$guid}}`
- **Timestamps**: `{{$timestamp}}`, `{{$isoTimestamp}}`
- **Date/Time**: `{{$datetime format}}`, `{{$localDatetime format}}`
- **Random Values**: `{{$randomInt}}`, JetBrains random generators
- **Environment Access**: `{{$processEnv VAR}}`, `{{$env.VAR}}`, `{{$dotenv VAR}}`

### Test Coverage
- Comprehensive unit tests in `client_execute_vars_test.go`, `client_execute_inplace_vars_test.go`
- Parser tests in `parser_environment_vars_test.go`, `parser_system_vars_test.go`
- Integration tests with real `.http` files demonstrating all variable types
- Edge case coverage for precedence, error handling, and malformed variables

## 8. Current Status

**STATUS: ✅ FULLY IMPLEMENTED AND TESTED**

All functional and non-functional requirements for request variable support have been successfully implemented. The feature is production-ready with:

- Complete variable substitution system
- All documented system variables working
- Proper precedence handling
- Environment file integration
- Comprehensive error handling
- Full test coverage

**Resolution to Open Questions:**
- Undefined variables without fallbacks result in empty string replacement (graceful degradation approach)
- Variable precedence clearly defined and implemented as documented

No open questions or issues remain for this PRD. 
