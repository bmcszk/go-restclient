# Test Consolidation Plan for go-restclient

**Version:** 1.0
**Date:** 2025-06-08
**Status:** Draft

## 1. Introduction and Goal

This document outlines a plan for consolidating and refactoring the existing Go test suite for the `go-restclient` project. The primary goal is to improve the test suite's organization, readability, maintainability, and efficiency by reducing redundancy and grouping related tests more logically. This plan is based on the initial test-to-requirement mappings found in `prds/jetbrains_compatibility/tests-to-requirements.md`.

## 2. Prerequisites

The test consolidation will proceed even if the test suite is not fully stable (i.e., `make check` has known failures). Addressing these failures will be a separate, parallel, or subsequent effort.

## 3. Methodology

The consolidation process will be approached in the following phases:

### Phase 1: Comprehensive Analysis and Candidate Identification
1.  **Parse `tests-to-requirements.md`:**
    *   Systematically review the `tests-to-requirements.md` mapping file.
    *   Build a reverse map: `Documentation Section/Feature -> List of Test Functions/Files`.
    *   This will help identify which features are tested by multiple test functions or across multiple test files.
2.  **Identify Consolidation Candidates:**
    *   Based on the reverse map, pinpoint areas where tests can be logically grouped or merged.
    *   **Criteria for Candidates:**
        *   Multiple test functions mapping to the exact same documentation line range or feature.
        *   Test functions in different files testing very similar aspects of a single feature.
        *   Tests exhibiting significant overlap in setup, execution steps, or assertion logic.
        *   Small, highly-focused test files whose contents could be logically merged into a more comprehensive file for a given component or feature set (e.g., merging multiple small `parser_featureX_test.go` files).

### Phase 2: Detailed Review of Candidate Tests
1.  **Understand Test Specifics:**
    *   For each group of consolidation candidates, perform a detailed code review of the relevant test functions and files.
    *   Analyze their structure (Given-When-Then), setup, action, and assertion logic.
    *   Determine if they represent genuinely distinct scenarios of the same feature or if there's true redundancy.
    *   Assess the complexity and risk of merging/refactoring.

### Phase 3: Strategy Formulation and Iterative Execution
1.  **Define Specific Consolidation Strategies:**
    *   For each identified group, formulate a concrete refactoring strategy. Examples include:
        *   **Merging Test Functions:** Combine multiple test functions into a single function using sub-tests (`t.Run("scenario_name", func(t *testing.T){...})`). This is suitable when tests share significant setup but test different facets or inputs of the same core functionality.
        *   **Relocating Test Functions:** Move test functions between files to group them more logically by the component (e.g., parser, validator, client) or feature they target.
        *   **Refactoring Helper Functions:** Identify and consolidate common test setup, action, or assertion logic into shared helper functions. These could reside in co-located `_test_helpers.go` files or a more general `testutils` package if broadly applicable.
        *   **Merging Test Files:** If multiple small test files cover closely related aspects of a single component, consider merging them into one more comprehensive test file, ensuring the resulting file remains within project size guidelines (e.g., max 1000 lines).
2.  **Iterative Implementation and Verification:**
    *   Apply consolidation changes incrementally, one logical group or strategy at a time.
    *   After each significant change:
        *   Run `make check` to ensure all tests still pass and no new issues are introduced.
        *   Update `prds/jetbrains_compatibility/tests-to-requirements.md` if test function names, file locations, or coverage scope changes.
        *   Commit changes with clear, descriptive messages (e.g., `refactor(tests): Consolidate parser tests for XYZ feature`).
        *   Push changes regularly.

## 4. Potential Areas for Consolidation (Initial High-Level View)

Based on the current structure and the `tests-to-requirements.md` mapping, the following areas are likely candidates for consolidation:

*   **Parser Tests (`parser_*_test.go`):**
    *   Tests for different aspects of request parsing (structure, headers, body, settings).
    *   Tests for variable definition and substitution parsing.
    *   Tests for authentication mechanism parsing.
*   **Validator Tests (`validator_*_test.go`):**
    *   Tests for various response validation placeholders (`$any`, `$regexp`, `$any(guid)`, etc.).
    *   Tests for status, header, and body validation logic.
*   **Client Execution Tests (`client_execute_*_test.go`):**
    *   Tests for variable handling during request execution (environment, file-level, in-place).
    *   Tests for core request execution flows and edge cases.
    *   Tests related to client configuration options (cookies, redirects, base URL).
*   **Variable Handling Tests (`hresp_vars_test.go`, parts of `client_execute_vars_test.go`, etc.):**
    *   Consolidate tests related to variable definition, parsing, substitution, and precedence across different scopes (.http files, .hresp files, environment).


## 4.A. Initial Review and Decisions (Pre-Stabilization)

This section documents specific decisions made during an initial review pass of the test suite, conducted prior to full test suite stabilization (`make check` passing). These decisions guide immediate cleanup actions.

1.  **`client_execute_config_test.go`**:
    *   **Decision:** All content removed (file emptied).
    *   **Rationale:** The file contained only commented-out tests (`TestExecuteFile_WithBaseURL`, `TestExecuteFile_WithDefaultHeaders`) and a minimal placeholder test (`TestMinimalInConfig`), none of which tested active, documented HTTP syntax features.

2.  **`parser_test.go/TestParseRequestFile_Imports`**:
    *   **Decision:** Test function, its helper struct (`parseRequestFileImportsTestCase`), and helper function (`runParseRequestFileImportsSubtest`) removed.
    *   **Rationale:** The `@import` directive tested is not a documented feature in `docs/http_syntax.md` that `go-restclient` aims to support or explicitly reject. It was considered a "hallucination" in the context of documented features.

3.  **`client_execute_core_test.go/TestExecuteFile_ParseError`**:
    *   **Decision:** Kept.
    *   **Rationale:** Classified as essential for testing overall system robustness and error handling (specifically, how the client handles parser failures). While not directly tied to a specific syntax element in `docs/http_syntax.md`, it ensures graceful failure modes.

4.  **`validator_setup_test.go/TestValidateResponses_NilAndEmptyActuals`**:
    *   **Decision:** Kept.
    *   **Rationale:** Classified as essential for testing the operational robustness of the response validation mechanism, particularly regarding problematic input arguments (e.g., nil or empty actual responses).

5.  **`validator_setup_test.go/TestValidateResponses_FileErrors`**:
    *   **Decision:** Kept.
    *   **Rationale:** Classified as essential for testing the error handling of the response validation mechanism, specifically concerning the integrity and accessibility of the `.hresp` file (e.g., missing, empty, or malformed files).


## 5. Guiding Principles for Consolidation

*   **Clarity and Readability:** The primary goal is to make tests easier to understand and maintain.
*   **Maintainability:** Consolidated tests should be easier to update when underlying code changes.
*   **Reduced Redundancy:** Eliminate unnecessary duplication of test logic or setup.
*   **Logical Grouping:** Tests for related functionalities should be grouped together.
*   **Adherence to Project Standards:** All refactored tests must continue to adhere to the project's Go development and testing guidelines (TDD, GWT, use of `testify`, file/function size limits, etc.).

## 6. Expected Benefits

*   A more organized and intuitive test suite.
*   Reduced effort for test maintenance and updates.
*   Faster identification of test coverage for specific features.
*   Potentially faster overall test execution if redundant setups are eliminated.

## 7. Future Execution Steps

1.  **Perform Detailed Analysis:** Execute Phase 1 and Phase 2 of the methodology described above.
2.  **Prioritize Consolidation Efforts:** Start with areas offering the most significant improvement for effort.
3.  **Implement Incrementally:** Execute Phase 3, applying and verifying changes in small, manageable steps.
4.  **Review and Iterate:** Periodically review the progress and adjust the plan as needed.
5.  **(Post-Consolidation/Parallel) Stabilize `make check`:** Resolve any outstanding test failures.
