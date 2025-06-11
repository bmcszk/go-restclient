# Project Learnings & Mistake Log

**Consolidated Learnings & Rule References (As of 2025-06-11)**

Many specific issues and lessons learned previously documented in detail below have now been consolidated and formalized into the project's operational guidelines. For comprehensive guidance, please refer to:

*   **AI Tool Usage (Code Editing):** [`/.windsurf/rules/ai_tool_usage_code_editing.md`](/.windsurf/rules/ai_tool_usage_code_editing.md)
    *   Covers best practices, verification steps, limitations, and fallback strategies for `edit_file` and `replace_file_content`.
    *   Key Takeaways:
        *   **Verification is CRITICAL:** Always check file content before and after edits.
        *   **`TargetContent` Precision:** For `replace_file_content`, `TargetContent` must be exact and often needs unique surrounding context.
        *   **Tool Limitations:** Be aware of `edit_file` issues with special characters, large diffs, and its potential for corruption.
        *   **Fallback Strategies:** Know when and how to use patch files or manual edits if tools fail.
*   **Git Workflow & Commits:** [`/.windsurf/rules/git_guidelines.md`](/.windsurf/rules/git_guidelines.md)
    *   Details branching, commit messages (including special character handling), staging, pushing, PR/MR processes, and file restoration.
    *   Key Takeaways:
        *   **Commit Message Escaping:** Ensure special characters in commit messages are handled correctly.
        *   **File Restoration:** Understand how to restore tracked vs. untracked files.
        *   **Adherence to Workflow:** Follow defined branching and PR processes.

**Ongoing Learnings:**

This document will continue to log *new* or particularly nuanced learnings that are not yet fully covered by the established rules, or specific incidents that highlight the importance of adhering to them.
