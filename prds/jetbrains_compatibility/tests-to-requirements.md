# Test Cases to HTTP Syntax Requirements Mapping

**🚨 DOCUMENT OBSOLETE - DO NOT USE 🚨**

This document has been **COMPLETELY REPLACED** by `test_coverage_mapping.md` which reflects the current test structure.

## Current Test Structure (December 2025)

✅ **Use this instead**: `test_coverage_mapping.md` - Accurate mapping with verified test references

The current test suite consists of:
- `client_test.go` (84 test functions) - All HTTP syntax testing through `Client.ExecuteFile()`
- `validator_test.go` (15 test functions) - Response validation testing 
- `hresp_vars_test.go` (1 test function) - Variable extraction testing

## Why This Document is Obsolete

This document previously contained **hundreds of references to test files that no longer exist**. The test structure was consolidated to use public APIs rather than private parser functions, providing better integration testing.

## Migration Guide

| Old Reference | Current Location |
|---------------|------------------|
| `parser_*.go` test files | Consolidated into `client_test.go` using `Client.ExecuteFile()` |
| `client_execute_*.go` test files | Consolidated into `client_test.go` |
| `validator_*.go` test files | Consolidated into `validator_test.go` |
| Individual parser unit tests | Replaced with integration tests using real .http files |

## Accurate Documentation

For current test coverage and requirements mapping, see:
- **`test_coverage_mapping.md`** - Complete coverage analysis with verified test references
- **`jetbrains_compatibility_prd.md`** - Feature requirements and implementation status
- **`jetbrains_compatibility_prd_task_tracking.md`** - Task completion tracking

---

*This document is preserved for historical reference but should not be used for current development planning.*