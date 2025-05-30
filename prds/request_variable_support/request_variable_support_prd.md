# PRD: Request File Variable Support

**Document Version:** 1.0
**Date:** 2025-05-30
**Status:** Draft

## 1. Introduction

This document outlines the product requirements for Variable Support in Request Files within the `go-restclient` library. This feature allows users to define and use variables within `.http` request files for dynamic content generation and configuration, including system-generated values, environment variables, and user-provided custom variables.

## 2. Goals

- To enable dynamic data in request files through variable substitution.
- To support various sources for variables: system-generated, environment, `.env` files, and programmatic custom inputs.
- To provide a clear syntax for defining and using variables.
- To allow overriding of variables from different sources.

## 3. User Stories / Functional Requirements

| ID          | Requirement / User Story                                                                                                    |
| :---------- | :-------------------------------------------------------------------------------------------------------------------------- |
| US-RVS-001  | As a developer, I want the library to generally support variables in request files (e.g., for hostnames, tokens). (REQ-LIB-007)      |
| US-RVS-002  | As a developer, I want to define and use custom variables within the request file. (REQ-LIB-013)                                |
| US-RVS-003  | As a developer, I want to use a `{{$guid}}` system variable that generates a new GUID for each substitution. (REQ-LIB-014)         |
| US-RVS-004  | As a developer, I want to use a `{{$randomInt min max}}` system variable for a random integer in a range. (REQ-LIB-015)           |
| US-RVS-005  | As a developer, I want to use a `{{$timestamp}}` system variable for the current UTC Unix epoch seconds. (REQ-LIB-016)             |
| US-RVS-006  | As a developer, I want to use a `{{$datetime format}}` system variable for a formatted current UTC datetime. (REQ-LIB-017)         |
| US-RVS-007  | As a developer, I want to use a `{{$localDatetime format}}` system variable for a formatted current local datetime. (REQ-LIB-018) |
| US-RVS-008  | As a developer, I want to use `{{$processEnv variableName}}` to substitute system environment variables. (REQ-LIB-019)             |
| US-RVS-009  | As a developer, I want to use `{{$dotenv variableName}}` to substitute variables from a `.env` file. (REQ-LIB-020)                   |
| US-RVS-010  | As a developer, I want to programmatically provide custom variables that can override variables from other sources. (REQ-LIB-021) |

## 4. Acceptance Criteria

- **AC-RVS-001.1:** Variables defined using `{{variableName}}` syntax are substituted correctly before sending the request.
- **AC-RVS-002.1:** `{{$guid}}` substitutes a valid unique GUID.
- **AC-RVS-003.1:** `{{$randomInt min max}}` substitutes an integer within the specified inclusive range.
- **AC-RVS-004.1:** `{{$timestamp}}` substitutes the current Unix epoch time in seconds (UTC).
- **AC-RVS-005.1:** `{{$datetime format}}` substitutes the current UTC datetime formatted according to the Go layout string provided in `format`.
- **AC-RVS-006.1:** `{{$localDatetime format}}` substitutes the current local datetime formatted according to the Go layout string provided in `format`.
- **AC-RVS-007.1:** `{{$processEnv variableName}}` correctly substitutes the value of the specified environment variable; if not set, substitution results in an empty string or a specific error/behavior.
- **AC-RVS-008.1:** `{{$dotenv variableName}}` correctly substitutes the value from a `.env` file in the CWD or specified path; if not found, substitution results in an empty string or a specific error/behavior.
- **AC-RVS-009.1:** Custom variables provided programmatically during `ExecuteFile` calls take precedence over variables defined within the file or loaded from `.env` / system environment if names conflict.
- **AC-RVS-010.1:** Undefined variables (not system, not env, not custom) result in an error or are replaced by an empty string (TBD: clarify behavior).

## 5. Non-Functional Requirements

- **NFR-RVS-001:** Variable substitution should be performant. (NFR-LIB-001 related - ease of use implies reasonable performance)
- **NFR-RVS-002:** Documentation for all supported variables, their syntax, and precedence rules must be clear. (NFR-LIB-002)
- **NFR-RVS-003:** The variable substitution logic must have good unit test coverage. (NFR-LIB-003)

## 6. Out of Scope

- Complex scripting or expressions within variables.

## 7. Open Questions

- What should be the behavior for undefined variables if not covered by any source (error, empty string, leave as is)? Currently AC-RVS-010.1 suggests clarifying this. 
