# Tasks

Last Updated: 2025-05-27

| ID       | Description                                                                 | Status   | Assignee | Due Date   |
| :------- | :-------------------------------------------------------------------------- | :------- | :------- | :--------- |
| TASK-021 | Refactor `ExecuteFile` to use `errgroup` for handling errors from `executeRequest`. | Done     | AI       | 2025-05-28 |
| TASK-022 | Implement `ValidateResponse` method in `Client` or a new validator component. | Done     | AI       | 2025-05-29 |
| TASK-023 | Define response file format allowing `###` separator and update parser.     | Blocked  | AI       | 2025-05-29 |
| TASK-024 | Add comprehensive unit tests for `ValidateResponse` using `sample1.http` as basis for expected data. | Done     | AI       | 2025-05-30 |
| TASK-030 | Implement `JSONPathChecks` validation in `ValidateResponse` and `ExpectedResponse`. | ToDo     | AI       | 2025-06-04 |
| TASK-031 | Add unit tests for `JSONPathChecks` in `validator_test.go`.                 | ToDo     | AI       | 2025-06-04 |
| TASK-032 | Implement `HeadersContain` validation in `ValidateResponse` and `ExpectedResponse`. | ToDo     | AI       | 2025-06-05 |
| TASK-033 | Add unit tests for `HeadersContain` in `validator_test.go`.               | ToDo     | AI       | 2025-06-05 |
