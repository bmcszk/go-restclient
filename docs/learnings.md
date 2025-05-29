# Learnings Log

Last Updated: 2025-05-28

## 2025-05-27: Library Structure Refinement

*   **Decision:** Shifted from placing library Go source files under `pkg/restclient/` to placing them directly in the project root.
*   **Rationale:** For a relatively small library with a public interface, a flatter structure can be simpler and more direct. The `pkg/` convention is often more beneficial for larger projects or when internal packages are numerous.
*   **Impact:** `docs/project_structure.md` and `docs/tasks.md` updated. `pkg/restclient` directory removed.

## 2025-05-28: Persistent Failure of `{{$datetime}}` Substitution in URL Paths

- **Issue**: Despite multiple attempts and refactoring strategies for the `substituteSystemVariables` function in `client.go`, the `{{$datetime "format"}}` system variable consistently failed to be substituted correctly when it appeared within a URL path. While the substitution worked for headers and request bodies, in URLs, the placeholder would either remain or be partially mangled, leading to URL-encoded braces (`%7B%7B`) in the path received by the mock server. This caused test failures for `TestExecuteFile_WithDatetimeSystemVariable`.
- **Strategies Attempted**: 
    1. Initial implementation using `regexp.ReplaceAllStringFunc`.
    2. Refactoring to use `regexp.FindAllStringSubmatch` followed by a loop of `strings.ReplaceAll`.
    3. Ensuring precise string generation in tests using `fmt.Sprintf` for the expected placeholders.
    4. Adding and analyzing debug logs (which confirmed the input string to `substituteSystemVariables` was correct but the output for URLs was still wrong).
- **Impact**: The inability to reliably substitute `{{$datetime}}` in URL paths means this feature (TASK-048) and its tests (TASK-049) cannot be completed correctly at this time. Combined with the earlier blocking issues for `{{$randomInt}}` and `{{$timestamp` (TASK-044 to TASK-047) due to regex tool problems, a significant portion of system variable functionality is currently stalled.
- **Resolution**: The commit implementing `{{$datetime}}` (bff4fea) was reverted. Tasks TASK-048 and TASK-049 were marked as "Blocked".
- **Learning**: There appears to be a very subtle and persistent issue with how system variable placeholders are processed or substituted when they are part of a larger URL string path, which was not resolved by standard regex techniques or refactoring of the substitution logic. The problem might lie in an unexpected interaction between URL parsing, string manipulation, and the regex engine for these specific substring cases, or an as-yet unidentified pre-processing step altering the URL string before `substituteSystemVariables` is effective. Further investigation would require deeper insights into the `net/url` package\'s parsing intricacies or more advanced debugging of the string states at each step within `ExecuteFile`.

## 2025-05-29: Persistent `edit_file` Failures for Backslash Corrections

- **Issue**: The `edit_file` tool repeatedly failed to apply corrections to `.hresp` files when the goal was to change the number of backslashes in a regular expression pattern (e.g., changing `\\d` to `\d`). Despite multiple attempts using various `code_edit` formats (full file content, specific line with context markers), the tool would report "no changes made".
- **Example**: Attempts to correct `Value: {{$regexp `^\\d{3}\\.test$`}}` to `Value: {{$regexp `^\d{3}\.test$`}}` in `testdata/http_response_files/validator_body_regexp_special_chars.hresp` were unsuccessful.
- **Impact**: This prevented automated correction of test files crucial for `{{$regexp pattern}}` validation, specifically `SCENARIO-LIB-022-004`. It also affected `{{$anyDatetime format}}` validation where escaped quotes (e.g., `\"2006-01-02\"`) needed to be changed to regular quotes (`"2006-01-02"`), although one instance of this was eventually corrected.
- **Learning**: The `edit_file` tool appears to have limitations or bugs when handling string literals containing backslashes, backticks, and quotes, especially when the desired change involves altering the number of backslashes. Python string escaping rules for the `code_edit` argument, combined with the tool's own parsing and diffing logic, make these types of changes highly unreliable. For such corrections, manual intervention or a different file manipulation strategy might be necessary.
