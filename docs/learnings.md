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
