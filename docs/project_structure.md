# Project Structure (Go REST Client Library)

Last Updated: 2025-05-27

This document outlines the project structure for the `go-restclient` library.

-   `client.go`: Defines the `Client` struct and its methods, including the primary public API `ExecuteFile`.
-   `parser.go`: Logic for parsing `.rest` / `.http` request files into `Request` structs.
-   `request.go`: Defines the `Request` and `ParsedFile` structs representing parsed HTTP requests.
-   `response.go`: Defines the `Response` struct representing a received HTTP response and `ExpectedResponse` for validation purposes.
-   `validator.go`: Logic for validating an actual `Response` against an `ExpectedResponse`.
-   `client_test.go`, `parser_test.go`, `validator_test.go` etc.: Unit tests for the library components.
-   `testdata/`
    -   `http_request_files/`: Contains example `.http` files used in unit tests.
-   `docs/`
    -   `requirements.md`: Project requirements.
    -   `tasks.md`: Development tasks.
    -   `decisions.md`: Architectural and design decisions.
    -   `learnings.md`: Log of mistakes and resolutions.
    -   `project_structure.md`: This file.
-   `.cursor/rules/`: AI assistant guidelines.
-   `Makefile`: Standard build, test, lint commands.
-   `go.mod`, `go.sum`: Go module files.
-   `README.md`: Project overview, setup, and usage instructions.
-   `.gitignore`: Specifies intentionally untracked files that Git should ignore. 
