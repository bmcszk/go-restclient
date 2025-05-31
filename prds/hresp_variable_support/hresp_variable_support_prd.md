**Version:** 1.0
**Date:** 2025-05-30
**Status:** Draft

## 1. Introduction

This document outlines the requirements for adding variable substitution capabilities to `.hresp` files, mirroring the functionality currently available in `.http` files. It also includes the ability to provide these variables as a map during relevant function calls.

## 2. Goals

*   Allow users to define and use variables within `.hresp` files for dynamic content.
*   Support defining variables directly within `.hresp` files using `@name = value` syntax.
*   Enable passing variables to response parsing/validation logic via a `map[string]interface{}`.
*   Maintain consistency with the existing variable substitution mechanism in `.http` files.

## 3. User Stories

*   **US-001:** As a developer, I want to use variables (e.g., `{{my_variable}}`) in my `.hresp` files for expected status, headers, and body content, so that I can create more flexible and reusable response validation templates.
*   **US-002:** As a developer, I want to provide a map of variables when **initializing my HTTP client**, so that I can dynamically set expected values for response validation across multiple `.hresp` files processed by that client.
*   **US-003:** As a developer, I want to define common variables directly inside an `.hresp` file using `@name = value` syntax, so I don't have to pass them programmatically for every validation using that file.
*   **US-004:** As a developer, I want the variable substitution in `.hresp` files to behave identically to the substitution in `.http` files (e.g., regarding syntax, fallback values, environment variable sourcing, in-file definitions), so that I don't have to learn a new system.

## 4. Functional Requirements

*   **FR-001 (In-File Variable Definition):** Users MUST be able to define variables directly within `.hresp` files using the `@name = value` syntax, similar to `.http` files. These definitions should typically be at the top of the file.
*   **FR-002 (Variable Usage):** Users MUST be able to use these variables (and others defined via client options or environment) in the expected status line, header values, and the expected body of an `.hresp` file, using the `{{variable_name}}` or `{{variable_name | fallback_value}}` syntax.
*   **FR-003 (Programmatic Variable Provision):** The `Client` MUST provide an option (e.g., `WithVars(map[string]interface{})`) to allow users to supply a map of variables programmatically during client initialization. These variables can then be used for substitution in both `.http` and `.hresp` files processed by that client instance.
*   **FR-004 (Variable Resolution Precedence):** The system MUST resolve variables with the following order of precedence (highest to lowest):
    1.  Variables provided programmatically to the `Client` instance (via the option in FR-003).
    2.  Variables defined within the `.hresp` file itself (using `@name = value`).
    3.  Environment variables.
    4.  Fallback values specified in the substitution syntax (e.g., `{{variable_name | fallback_value}}`).
*   **FR-005 (Optional Programmatic Variables):** Providing programmatic variables to the `Client` (FR-003) MUST be optional. If not provided, substitution proceeds with other sources (in-file, environment, fallbacks).
*   **FR-006 (Error on Undefined Variable without Fallback):** If a variable placeholder `{{variable_name}}` is used in an `.hresp` file, and that variable is not defined through any source (programmatic, in-file, environment) and no fallback is provided, the response validation SHOULD error appropriately, indicating the missing variable and failed substitution.

## 5. Non-Functional Requirements

*   **NFR-001: Performance:** The variable substitution process should not introduce significant performance overhead.
*   **NFR-002: Error Handling:** Clear error messages MUST be provided if variables are malformed, or if required variables are not found and have no fallbacks.

## 6. Acceptance Criteria

*   **AC-001:** Unit tests verify successful variable substitution in status line, headers, and body of an `.hresp` file using a provided variable map.
*   **AC-002:** Unit tests verify successful variable substitution using variables defined within the `.hresp` file itself (`@name = value`).
*   **AC-003:** Unit tests verify successful variable substitution using environment variables when not present in the map or in-file definitions.
*   **AC-004:** Unit tests verify successful variable substitution using fallback values when not present in the map, in-file definitions, or environment.
*   **AC-005:** Unit tests verify the correct order of precedence for variable resolution (map overrides in-file, in-file overrides environment, environment overrides fallback).
*   **AC-006:** Unit tests verify appropriate error handling for missing variables without fallbacks after checking all sources.
*   **AC-007:** Unit tests verify that the existing `.http` file variable substitution and definition mechanisms remain unaffected.
*   **AC-008:** An `.hresp` file using variables (defined in-file and/or provided via map) for status, headers, and body can be successfully parsed and used for validation.
*   **AC-009:** The `ValidateResponse` function signature is updated to accept the variable map.
*   **AC-010:** The `.hresp` parser correctly extracts variables defined with `@name = value`.

## 7. Out of Scope

*   Complex logic or expressions within the variable syntax (beyond simple fallbacks).
*   Introducing new variable sources beyond the map and environment variables.

## 8. Open Questions/Ambiguities

*   (None at this time, assuming behavior should closely mirror `.http` variable processing.) 
