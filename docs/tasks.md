# Tasks

Last Updated: 2025-05-27

| ID       | Description                                                                 | Status   | Assignee | Due Date   |
| :------- | :-------------------------------------------------------------------------- | :------- | :------- | :--------- |
| TASK-021 | Refactor `ExecuteFile` to use `errgroup` for handling errors from `executeRequest`. | Done     | AI       | 2025-05-28 |
| TASK-022 | Implement `ValidateResponse` method in `Client` or a new validator component. | Done     | AI       | 2025-05-29 |
| TASK-023 | Define response file format allowing `###` separator and update parser.     | Done | AI       | 2025-05-29 |
| TASK-024 | Create E2E tests for `ValidateResponse` using `sample1.http` and its expected response. | ToDo     | AI       | 2025-05-30 |
| TASK-025 | Add/verify E2E test for multiple requests in one `.http` file (e.g. `sample1.http`). | ToDo     | AI       | 2025-05-30 |
| TASK-026 | Implement E2E tests for SCENARIO-LIB-008-001, SCENARIO-LIB-008-002, SCENARIO-LIB-008-003 | ToDo     | AI       | 2025-05-31 |
| TASK-027 | Implement E2E tests for SCENARIO-LIB-009-001 to SCENARIO-LIB-009-008          | ToDo     | AI       | 2025-06-01 |
| TASK-028 | Implement E2E tests for SCENARIO-LIB-010-001 to SCENARIO-LIB-010-003          | ToDo     | AI       | 2025-06-02 |
| TASK-029 | Implement E2E tests for SCENARIO-LIB-011-001 to SCENARIO-LIB-011-004          | ToDo     | AI       | 2025-06-03 |
| TASK-030 | Implement `JSONPathChecks` validation in `ValidateResponse` and `ExpectedResponse`. | ToDo     | AI       | 2025-06-04 |
| TASK-031 | Add unit tests for `JSONPathChecks` in `validator_test.go`.                 | ToDo     | AI       | 2025-06-04 |
| TASK-032 | Implement `HeadersContain` validation in `ValidateResponse` and `ExpectedResponse`. | ToDo     | AI       | 2025-06-05 |
| TASK-033 | Add unit tests for `HeadersContain` in `validator_test.go`.               | ToDo     | AI       | 2025-06-05 |
