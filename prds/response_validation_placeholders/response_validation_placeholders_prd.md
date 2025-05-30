# PRD: Expected Response Validation Placeholders

**Document Version:** 1.0
**Date:** 2025-05-30
**Status:** Draft

## 1. Introduction

This document outlines the product requirements for dynamic Validation Placeholders in Expected Responses within the `go-restclient` library. This feature allows users to use special placeholders in their `.hresp` (expected response) files to validate parts of an actual HTTP response against patterns or types rather than exact static values.

## 2. Goals

- To enable flexible validation of dynamic response content (e.g., generated IDs, timestamps, content matching patterns).
- To provide a clear syntax for using these placeholders in expected response bodies.
- To support common validation needs like GUIDs, timestamps, regular expressions, and general "any" value matching.

## 3. User Stories / Functional Requirements

| ID          | Requirement / User Story                                                                                                                  |
| :---------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| US-RVP-001  | As a developer, I want to use `{{$regexp pattern}}` in an expected response body to validate a part against a regex. (REQ-LIB-022)            |
| US-RVP-002  | As a developer, I want to use `{{$anyGuid}}` in an expected response body to validate if a part is any valid GUID. (REQ-LIB-023)               |
| US-RVP-003  | As a developer, I want to use `{{$anyTimestamp}}` in an expected response body to validate if a part is any valid Unix timestamp. (REQ-LIB-024) |
| US-RVP-004  | As a developer, I want to use `{{$anyDatetime format}}` in an expected response body to validate a datetime string by format. (REQ-LIB-025)     |
| US-RVP-005  | As a developer, I want to use `{{$any}}` in an expected response body to validate if any value is present (non-empty). (REQ-LIB-026)             |

## 4. Acceptance Criteria

- **AC-RVP-001.1:** `{{$regexp pattern}}` correctly validates if the corresponding part of the actual response body matches the provided regular expression `pattern`.
- **AC-RVP-002.1:** `{{$anyGuid}}` correctly validates if the corresponding part of the actual response body is a valid GUID string.
- **AC-RVP-003.1:** `{{$anyTimestamp}}` correctly validates if the corresponding part of the actual response body is an integer representing a plausible Unix timestamp.
- **AC-RVP-004.1:** `{{$anyDatetime format}}` correctly validates if the corresponding part of the actual response body is a datetime string that matches the specified `format` (e.g., RFC1123, ISO8601, or a Go layout string).
- **AC-RVP-005.1:** `{{$any}}` correctly validates if the corresponding part of the actual response body is present and not empty.
- **AC-RVP-006.1:** Placeholders are correctly parsed and distinguished from literal text in the expected response body.
- **AC-RVP-007.1:** If a placeholder validation fails, the discrepancy report clearly indicates the placeholder and the reason for failure.

## 5. Non-Functional Requirements

- **NFR-RVP-001:** Placeholder validation should be reasonably performant. (NFR-LIB-001 related)
- **NFR-RVP-002:** Documentation for all supported placeholders and their usage must be clear and provide examples. (NFR-LIB-002)
- **NFR-RVP-003:** The placeholder validation logic must have good unit test coverage. (NFR-LIB-003)

## 6. Out of Scope

- User-defined custom validation placeholders (beyond those specified).
- Placeholders in expected headers or status codes (currently focused on response body).

## 7. Open Questions

- None at this time. 
