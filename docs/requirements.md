# Requirements

Last Updated: 2025-05-27

## Functional Requirements

- REQ-LIB-001: The library must parse request definitions from `.rest` or `.http` files.
- REQ-LIB-002: The request file format should support defining HTTP method, URL, headers, and body.
- REQ-LIB-003: The library must be able to send the parsed HTTP request to the target server.
- REQ-LIB-004: The library must capture the HTTP response (status code, headers, body).
- REQ-LIB-005: The library must allow specifying an expected response (status code, headers, body) via a corresponding file or structure.
- REQ-LIB-006: The library must compare the actual response with the expected response and report discrepancies.
- REQ-LIB-007: Support for variables in request files (e.g., for hostnames, tokens) is desirable.

## Non-Functional Requirements

- NFR-LIB-001: The library should be easy to integrate into Go E2E test suites.
- NFR-LIB-002: The library should have clear documentation and examples.
- NFR-LIB-003: The library should have good unit test coverage.
