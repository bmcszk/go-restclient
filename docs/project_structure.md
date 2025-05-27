# Project Structure (Go REST Client Library)

Last Updated: 2025-05-27

This document outlines the project structure for the `go-restclient` library.

-   `parser.go`: Logic for parsing `.rest` / `.http` request files.
-   `request.go`: Structures and functions for representing and sending HTTP requests.
-   `response.go`: Structures and functions for handling and validating HTTP responses.
-   `runner.go`: Orchestrates parsing, request execution, and response validation.
-   `file_utils.go`: (Optional) Utilities for file reading/writing if needed beyond standard library.
-   `client.go`: Public interface for the library.
-   `parser_test.go`, `request_test.go`, `client_test.go` etc.: Unit tests for the library components.
-   `docs/`
    -   `requirements.md`: Project requirements.
    -   `tasks.md`: Development tasks.
    -   `decisions.md`: Architectural and design decisions.
    -   `learnings.md`: Log of mistakes and resolutions.
    -   `project_structure.md`: This file.
    -   `examples/`: (Optional) Directory containing example `.rest` and expected response files.
-   `.cursor/rules/`: AI assistant guidelines.
-   `e2e/`
    -   `e2e_test.go`: End-to-end tests for the library, using the library itself to make requests.
    -   `testdata/`: Directory containing `.rest` files and expected response files for E2E tests.
-   `Makefile`: Standard build, test, lint commands.
-   `go.mod`, `go.sum`: Go module files.
-   `README.md`: Project overview, setup, and usage instructions.
-   `.gitignore`: Specifies intentionally untracked files that Git should ignore. 
