# Learnings Log

Last Updated: 2025-05-27

## 2025-05-27: Project Scope Misunderstanding

*   **Mistake:** Initiated project setup for a full Go microservice (`services/restclient/...`) based on `go_project_setup_guideline.mdc` before fully understanding the user's intent for a simpler Go library.
*   **Correction:** The user clarified that the goal is to build a Go library for E2E testing that parses `.rest`/`.http` files, sends HTTP requests, and validates responses against expected output files. The project structure will be much simpler, focusing on a library in the `pkg/` directory rather than a full service.
*   **Learning:** Always confirm the high-level project goal and type (e.g., service vs. library) before diving into detailed setup, especially when multiple guidelines might seem applicable. Clarify if a general guideline (like project setup) should be applied in its entirety or adapted for a more specific need.

## 2025-05-27: Library Structure Refinement

*   **Decision:** Shifted from placing library Go source files under `pkg/restclient/` to placing them directly in the project root.
*   **Rationale:** For a relatively small library with a public interface, a flatter structure can be simpler and more direct. The `pkg/` convention is often more beneficial for larger projects or when internal packages are numerous.
*   **Impact:** `docs/project_structure.md` and `docs/tasks.md` updated. `pkg/restclient` directory removed.

## 2025-05-27: Parser Implementation Oversight

*   **Mistake:** Planned to create a new `http_file_parser.go` and associated tests from scratch (TASK-003, TASK-004 initial versions) without first thoroughly verifying if the existing `parser.go` and `parser_test.go` already covered the necessary functionality for `.http` files.
*   **Correction:** Upon inspection, it was found that `parser.go` was already capable of handling `.http` file syntax (as it's similar to `.rest`). Tasks were revised to add specific test cases for `.http` files to the existing test suite, which then passed, confirming its suitability.
*   **Learning:** Before creating new components, always perform a thorough check of existing codebase elements to see if they can be leveraged or extended. This prevents redundant work and keeps the codebase leaner.

## 2025-05-27: Premature File Creation Planning

*   **Mistake:** When initially planning for E2E tests (before they were descoped), I listed the creation of the specific test file `e2e/scenario_lib_001_001_test.go` before ensuring its parent directory `e2e/` would be created.
*   **Correction:** I identified the missing step and added the directory creation task first. This particular sequence became moot when E2E tests were removed from scope, but the procedural learning is valid.
*   **Learning:** Always ensure directory structures are planned for creation before the files within them, especially when automating file/directory operations.

## 2025-05-27: Missed Git Workflow Steps

*   **Mistake:** Completed several development tasks (TASK-001 through TASK-015) and marked them as 'Done' in `docs/tasks.md` without following the prescribed Git workflow of proposing a `git commit` and `git push` after each task (or a logical group of tasks that pass `make check`).
*   **Correction:** Acknowledged the oversight. Will proceed to commit the accumulated changes for the completed tasks and will adhere to the commit/push workflow for all subsequent tasks.
*   **Learning:** Adhere strictly to all documented workflow steps, including version control procedures, as they are crucial for maintaining a clean and traceable project history. Regularly re-check guidelines if unsure.

## 2025-05-27: Git Staging/Committing Issues & File System State Uncertainty

*   **Mistake:** After being reminded about the Git workflow, attempts to stage and commit files (`git add`, `git commit`) failed with "nothing to commit, working tree clean," despite numerous `edit_file` operations reporting success. This indicates a potential misunderstanding of how `edit_file` interacts with the file system and Git staging, or an issue with the environment/tooling that prevents `git` from seeing the changes.
*   **Resolution (Pending):** Paused to seek clarification on the actual state of the files in the workspace. If files are modified, the `git add/commit` process needs to be debugged. If files are not modified, the `edit_file` tool's behavior needs to be investigated.
*   **Learning:** The successful reported execution of a file modification tool does not automatically guarantee that the changes are 1) physically written to disk in a way that `git` can see, or 2) automatically staged. Explicitly verify file changes and Git status when troubleshooting commit issues. Do not assume `edit_file` stages changes. 

## 2025-05-27: Incorrect Error Handling Strategy for `ExecuteFile`

*   **Mistake:** Implemented error handling in `ExecuteFile` using `golang.org/x/sync/errgroup` as per the initial interpretation of the requirement "ExecuteFile has wrong error handling, it should use error group (errgroup) for errors from executeRequest".
*   **Correction:** The user clarified that the intention was to use `github.com/hashicorp/go-multierror` for collecting multiple errors from the requests executed by `ExecuteFile`.
*   **Resolution:** The implementation will be refactored to use `go-multierror` instead of `errgroup`.
*   **Learning:** Clarify ambiguous terms like "error group" if multiple interpretations or common libraries exist. Explicitly confirm the intended library or pattern when a general term is used in requirements.
