# Tasks

Last Updated: 2025-05-28

| ID       | Description                                                                 | Status      | Assignee | Due Date   |
| :------- | :-------------------------------------------------------------------------- | :---------- | :------- | :--------- |
| TASK-021 | Refactor `ExecuteFile` to use `errgroup` for handling errors from `executeRequest`. | Done        | AI       | 2025-05-28 |
| TASK-022 | Implement `ValidateResponses` method to compare actual responses against an expected response file (supports multiple via `###`). | Done        | AI       | 2025-05-29 |
| TASK-023 | Define response file format allowing `###` separator and update parser.     | Done        | AI       | 2025-05-29 |
| TASK-024 | Add comprehensive unit tests for `ValidateResponse` using `sample1.http` as a basis for test data, covering various scenarios. | Done        | AI       | 2025-05-30 |
| TASK-025 | Add unit tests for `Client.ExecuteFile` to verify handling of multiple requests from a single `.http` request file and validating each response. | Done        | AI       | 2025-05-30 |
| TASK-026 | Implement unit tests covering SCENARIO-LIB-008-001, SCENARIO-LIB-008-002, SCENARIO-LIB-008-003. | Done        | AI       | 2025-05-31 |
| TASK-027 | Implement unit tests covering SCENARIO-LIB-009-001 to SCENARIO-LIB-009-008.          | Done        | AI       | 2025-06-01 |
| TASK-028 | Implement unit tests covering SCENARIO-LIB-010-001 to SCENARIO-LIB-010-003.          | Done        | AI       | 2025-06-02 |
| TASK-029 | Implement unit tests covering SCENARIO-LIB-011-001 to SCENARIO-LIB-011-004.          | Done        | AI       | 2025-06-03 |
| TASK-030 | Implement `JSONPathChecks` validation in `ValidateResponse` and `ExpectedResponse`. | Skipped     | AI       | 2025-06-04 |
| TASK-031 | Add unit tests for `JSONPathChecks` in `validator_test.go`.                 | Skipped     | AI       | 2025-06-04 |
| TASK-032 | Implement `HeadersContain` validation in `ValidateResponse` and `ExpectedResponse`. | Done        | AI       | 2025-06-05 |
| TASK-033 | Add unit tests for `HeadersContain` in `validator_test.go`.               | Done        | AI       | 2025-06-05 |
| TASK-034 | Implement unit tests covering SCENARIO-LIB-012-001 (reject JSON response file). | Skipped     | AI       | 2025-06-06 |
| TASK-035 | Implement unit tests covering SCENARIO-LIB-012-002 (reject YAML response file). | Skipped     | AI       | 2025-06-06 |
| TASK-036 | Fix trailing space in `testdata/http_response_files/multiple_responses_gt2_expected.http` (line with `response3`). May require manual edit. | Done        | AI       | 2025-05-29 |
| TASK-037 | Manually review and fix all test data files in `testdata/http_request_files/` and `testdata/http_response_files/` for whitespace/newline inconsistencies. | Done        | User     | 2025-05-29 |
| TASK-038 | Refactor validator_test.go to remove `writeExpectedResponseFile` and use real .http files from `testdata/http_response_files/`. | Done        | AI       | 2025-05-30 |
| TASK-039 | Manually restore/fix `validator_test.go` due to tool-induced corruption. File may need revert from VCS or careful manual edit to fix duplication/deletion. | Done        | User     | 2025-05-28 |
| TASK-040 | Implement REQ-LIB-013: Support for user-defined custom variables.             | Done        | AI       | 2025-06-07 |
| TASK-041 | Add unit tests for REQ-LIB-013 (SCENARIO-LIB-013-001 to SCENARIO-LIB-013-005). | Done        | AI       | 2025-06-07 |
| TASK-042 | Implement REQ-LIB-014: Support for `{{$guid}}` system variable.                | Done        | AI       | 2025-06-08 |
| TASK-043 | Add unit tests for REQ-LIB-014 (SCENARIO-LIB-014-001 to SCENARIO-LIB-014-004). | Done        | AI       | 2025-06-08 |
| TASK-044 | Implement REQ-LIB-015: Support for `{{$randomInt min max}}` system variable.   | Done        | AI       | 2025-06-09 |
| TASK-045 | Add unit tests for REQ-LIB-015 (SCENARIO-LIB-015-001 to SCENARIO-LIB-015-004). | Done        | AI       | 2025-06-09 |
| TASK-046 | Implement REQ-LIB-016: Support for `{{$timestamp}}` system variable.           | Done        | AI       | 2025-06-10 |
| TASK-047 | Add unit tests for REQ-LIB-016 (SCENARIO-LIB-016-001 to SCENARIO-LIB-016-002). | Done        | AI       | 2025-06-10 |
| TASK-048 | Implement REQ-LIB-017: Support for `{{$datetime format}}` system variable.     | Blocked     | AI       | 2025-06-11 |
| TASK-049 | Add unit tests for REQ-LIB-017 (SCENARIO-LIB-017-001 to SCENARIO-LIB-017-004). | Blocked     | AI       | 2025-06-11 |
| TASK-050 | Implement REQ-LIB-018: Support for `{{$localDatetime format}}` system variable. | Blocked     | AI       | 2025-06-12 |
| TASK-051 | Add unit tests for REQ-LIB-018 (SCENARIO-LIB-018-001 to SCENARIO-LIB-018-002). | Blocked     | AI       | 2025-06-12 |
| TASK-052 | Implement REQ-LIB-019: Support for `{{$processEnv variableName}}` system variable. | Done        | AI       | 2025-06-13 |
| TASK-053 | Add unit tests for REQ-LIB-019 (SCENARIO-LIB-019-001 to SCENARIO-LIB-019-002). | Done        | AI       | 2025-06-13 |
| TASK-054 | Implement REQ-LIB-020: Support for `{{$dotenv variableName}}` system variable.   | Done        | AI       | 2025-06-14 |
| TASK-055 | Add unit tests for REQ-LIB-020 (SCENARIO-LIB-020-001 to SCENARIO-LIB-020-003). | Done        | AI       | 2025-06-14 |
