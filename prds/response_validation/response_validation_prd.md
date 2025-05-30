# PRD: Response Validation

**Document Version:** 1.0
**Date:** 2025-05-30
**Status:** Draft

## 1. Introduction

This document outlines the product requirements for the Response Validation functionality of the `go-restclient` library. This feature allows users to define expected HTTP responses in `.http` files and compare them against actual responses received from a server.

## 2. Goals

- To enable users to specify expected HTTP responses (status code, headers, body).
- To provide a mechanism for comparing an actual HTTP response with an expected one.
- To support validation of multiple actual responses against expected responses defined in a single file.
- To standardize on the `.http` file format for defining expected responses.

## 3. User Stories / Functional Requirements

| ID         | Requirement / User Story                                                                                                                               |
| :--------- | :----------------------------------------------------------------------------------------------------------------------------------------------------- |
| US-RV-001  | As a developer, I want to specify an expected response (status code, headers, body) via a corresponding `.http` file. (REQ-LIB-005)                     |
| US-RV-002  | As a developer, I want the library to compare the actual response with the expected response and report discrepancies. (REQ-LIB-006)                  |
| US-RV-003  | As a developer, I want a method (`ValidateResponses`) to validate one or more actual HTTP responses against expected responses in a single `.http` file (using `###` separator). (REQ-LIB-009) |
| US-RV-004  | As a developer, I want the response file format to allow for multiple expected responses, separated by `###`. (REQ-LIB-010)                               |
| US-RV-005  | As a developer, I want the library to exclusively use the `.http` file format for defining expected responses, not other formats like JSON/YAML. (REQ-LIB-012) |

## 4. Acceptance Criteria

- **AC-RV-001.1:** The library correctly parses expected response definitions (status code, headers, body) from `.http` files.
- **AC-RV-002.1:** The comparison logic accurately identifies matches and mismatches between actual and expected status codes, headers, and bodies.
- **AC-RV-003.1:** Discrepancies found during validation are clearly reported (e.g., as part of a Go error or a validation result struct).
- **AC-RV-004.1:** The `ValidateResponses` method correctly handles multiple actual responses and maps them to the corresponding expected responses in a multi-response `.http` file.
- **AC-RV-005.1:** The library correctly parses `.http` files containing multiple expected responses separated by `###`.
- **AC-RV-006.1:** The library rejects attempts to load expected responses from formats other than `.http` (e.g., if a file has a `.json` extension or content that is clearly not in the `.http` response format).

## 5. Non-Functional Requirements

- **NFR-RV-001:** The response validation functionality must be easy to integrate into Go E2E test suites. (NFR-LIB-001)
- **NFR-RV-002:** Documentation for defining expected responses and using validation methods must be clear and provide examples. (NFR-LIB-002)
- **NFR-RV-003:** The code implementing this feature must have good unit test coverage. (NFR-LIB-003)

## 6. Out of Scope

- Placeholders or wildcard matching in expected responses (covered in "Expected Response Validation Placeholders" PRD).

## 7. Open Questions

- None at this time. 
