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
| TASK-028 | Implement unit tests covering SCENARIO-LIB-010-001 to SCENARIO-LIB-010-003.          | Blocked     | AI       | 2025-06-02 |
| TASK-029 | Implement unit tests covering SCENARIO-LIB-011-001 to SCENARIO-LIB-011-004.          | Done        | AI       | 2025-06-03 |
| TASK-030 | Implement `JSONPathChecks` validation in `ValidateResponse` and `ExpectedResponse`. | Skipped     | AI       | 2025-06-04 |
| TASK-031 | Add unit tests for `JSONPathChecks` in `validator_test.go`.                 | Skipped     | AI       | 2025-06-04 |
| TASK-032 | Implement `HeadersContain` validation in `ValidateResponse` and `ExpectedResponse`. | Done        | AI       | 2025-06-05 |
| TASK-033 | Add unit tests for `HeadersContain` in `validator_test.go`.               | Done        | AI       | 2025-06-05 |
| TASK-034 | Implement unit tests covering SCENARIO-LIB-012-001 (reject JSON response file). | Skipped     | AI       | 2025-06-06 |
| TASK-035 | Implement unit tests covering SCENARIO-LIB-012-002 (reject YAML response file). | Skipped     | AI       | 2025-06-06 |
| TASK-036 | Fix trailing space in `testdata/http_response_files/multiple_responses_gt2_expected.http` (line with `response3`). May require manual edit. | Done        | AI       | 2025-05-29 |
