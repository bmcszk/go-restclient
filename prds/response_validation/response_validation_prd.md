# PRD: Response Validation

**Document Version:** 2.0
**Date:** 2025-06-09
**Status:** Implemented

## 1. Introduction

This document outlines the product requirements for the Response Validation functionality of the `go-restclient` library. This feature allows users to define expected HTTP responses in `.hresp` files and compare them against actual responses received from a server.

**IMPLEMENTATION STATUS: ✅ FULLY IMPLEMENTED**
All response validation features are implemented and working, with comprehensive support for status, headers, and body validation using `.hresp` files.

## 2. Goals

- To enable users to specify expected HTTP responses (status code, headers, body).
- To provide a mechanism for comparing an actual HTTP response with an expected one.
- To support validation of multiple actual responses against expected responses defined in a single file.
- To standardize on the `.http` file format for defining expected responses.

## 3. User Stories / Functional Requirements

| ID         | Requirement / User Story                                                                                                                               | Status |
| :--------- | :----------------------------------------------------------------------------------------------------------------------------------------------------- | :------ |
| US-RV-001  | As a developer, I want to specify an expected response (status code, headers, body) via a corresponding `.hresp` file. (REQ-LIB-005)                     | ✅ IMPLEMENTED |
| US-RV-002  | As a developer, I want the library to compare the actual response with the expected response and report discrepancies. (REQ-LIB-006)                  | ✅ IMPLEMENTED |
| US-RV-003  | As a developer, I want a method (`ValidateResponses`) to validate one or more actual HTTP responses against expected responses in a single `.hresp` file (using `###` separator). (REQ-LIB-009) | ✅ IMPLEMENTED |
| US-RV-004  | As a developer, I want the response file format to allow for multiple expected responses, separated by `###`. (REQ-LIB-010)                               | ✅ IMPLEMENTED |
| US-RV-005  | As a developer, I want the library to use the `.hresp` file format for defining expected responses with support for validation placeholders. (REQ-LIB-012) | ✅ IMPLEMENTED |

## 4. Acceptance Criteria

- **AC-RV-001.1:** The library correctly parses expected response definitions (status code, headers, body) from `.hresp` files. ✅ IMPLEMENTED
- **AC-RV-002.1:** The comparison logic accurately identifies matches and mismatches between actual and expected status codes, headers, and bodies. ✅ IMPLEMENTED
- **AC-RV-003.1:** Discrepancies found during validation are clearly reported as part of a consolidated Go error with detailed differences. ✅ IMPLEMENTED
- **AC-RV-004.1:** The `ValidateResponses` method correctly handles multiple actual responses and maps them to the corresponding expected responses in a multi-response `.hresp` file. ✅ IMPLEMENTED
- **AC-RV-005.1:** The library correctly parses `.hresp` files containing multiple expected responses separated by `###`. ✅ IMPLEMENTED
- **AC-RV-006.1:** The library uses the dedicated `.hresp` file format optimized for response validation with placeholder support. ✅ IMPLEMENTED

### Additional Implemented Features
- **AC-RV-007.1:** Support for variable substitution in `.hresp` files using `@name = value` syntax and client variables. ✅ IMPLEMENTED
- **AC-RV-008.1:** Support for validation placeholders (`{{$any}}`, `{{$regexp}}`, `{{$anyGuid}}`, etc.) in response bodies. ✅ IMPLEMENTED
- **AC-RV-009.1:** Comprehensive error reporting with diff-style output for body mismatches. ✅ IMPLEMENTED
- **AC-RV-010.1:** Header validation with support for exact matching and partial matching (contains). ✅ IMPLEMENTED

## 5. Non-Functional Requirements

- **NFR-RV-001:** The response validation functionality must be easy to integrate into Go E2E test suites. (NFR-LIB-001)
- **NFR-RV-002:** Documentation for defining expected responses and using validation methods must be clear and provide examples. (NFR-LIB-002)
- **NFR-RV-003:** The code implementing this feature must have good unit test coverage. (NFR-LIB-003)

## 6. Out of Scope

- Complex scripting or custom validation logic beyond placeholders. ❌ NOT IMPLEMENTED
- Automatic response generation or recording features. ❌ NOT IMPLEMENTED

Note: Placeholders and wildcard matching are now IMPLEMENTED (originally was out of scope but has been added).

## 7. Implementation Details

### Core Implementation Files
- **`validator.go`**: Main validation logic with `ValidateResponses` method
- **`response.go`**: Response and ExpectedResponse data structures
- **`parser.go`**: `.hresp` file parsing with `parseExpectedResponses` function
- **`hresp_vars.go`**: Variable substitution support for `.hresp` files

### Key Features Implemented
1. **Status Validation**: Exact matching of HTTP status codes and status text
2. **Header Validation**: Support for exact header matching and partial matching
3. **Body Validation**: Text-based body comparison with placeholder support
4. **Multiple Response Support**: Handle multiple expected responses in single `.hresp` file
5. **Variable Support**: Full variable substitution in `.hresp` files
6. **Error Reporting**: Comprehensive error messages with diff output for debugging

### Validation Logic
- **Status Code**: Exact match required between actual and expected
- **Headers**: Configurable exact or substring matching
- **Body**: Text comparison with support for validation placeholders
- **Error Consolidation**: All validation errors collected and reported together

### File Format
- **`.hresp` Extension**: Dedicated file type for expected responses
- **HTTP Response Format**: Standard HTTP response format with status line, headers, body
- **Separators**: Multiple responses separated by `###`
- **Variables**: Support for `@name = value` definitions and `{{variable}}` substitution

### Test Coverage
- Comprehensive response validation tests in `validator_test.go`
- Integration tests with real `.hresp` files
- Edge case coverage for various response scenarios
- Placeholder validation tests

## 8. Current Status

**STATUS: ✅ FULLY IMPLEMENTED AND TESTED**

All functional and non-functional requirements for response validation have been successfully implemented. The feature is production-ready with:

- Complete `.hresp` file parsing and validation
- Comprehensive status, header, and body validation
- Variable substitution support
- Validation placeholder integration
- Robust error handling and reporting
- Full test coverage

The implementation exceeds the original scope by including placeholder support (originally planned for separate PRD) and comprehensive variable support, making it a complete response validation solution.

No open questions or issues remain for this PRD. 
