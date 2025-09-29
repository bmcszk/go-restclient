# JSON Comparison Whitespace Agnostic PRD - Task Tracking

## Overview
This document tracks the implementation progress for the JSON whitespace-agnostic comparison feature.

## Current Status
**Branch**: `feature/json-whitespace-agnostic-comparison`  
**Last Updated**: 2025-09-29  
**Current Phase**: Phase 1 (Research and Planning)

## Task Progress

### Phase 1: Research and Planning ✅
- [x] **TASK-001**: Analyze existing test cases for JSON comparison scenarios
  - **Status**: Completed
  - **Notes**: Found JSON validation tests in `validator_test.go`, existing .hresp files with JSON content
  - **Date**: 2025-09-29

- [x] **TASK-002**: Review existing `.hresp` files to understand JSON validation patterns
  - **Status**: Completed
  - **Notes**: Reviewed multiple .hresp files, identified patterns:
    - JSON responses with placeholders: `{{$anyGuid}}`, `{{$anyTimestamp}}`, `{{$anyDatetime format}}`, `{{$any}}`
    - Mixed formatting: some single-line, some multi-line with indentation
    - GraphQL responses with nested JSON structures
    - Content-Type headers consistently set to `application/json`
  - **Date**: 2025-09-29

- [x] **TASK-003**: Design JSON detection heuristic (Content-Type vs content parsing)
  - **Status**: Completed
  - **Notes**: Designed JSON detection heuristic using parsing approach (more reliable than Content-Type)
  - **Date**: 2025-09-29
  - **Design**:
    - **Approach**: Attempt JSON parsing on both expected and actual bodies
    - **Function**: `isJSONContent(body string) bool`
    - **Logic**: Try `json.Unmarshal()` into `interface{}`, return true if success
    - **Fallback**: If either body fails JSON parsing, use existing string comparison
    - **Performance**: Only attempt JSON parsing if both bodies contain `{` or `[` (quick pre-check)
    - **Reliability**: More accurate than Content-Type headers which can be missing/incorrect 

- [ ] **TASK-004**: Create task tracking document
  - **Status**: In Progress
  - **Notes**: This document being created
  - **Date**: 2025-09-29

### Phase 2: Unit Tests (TDD Approach)
- [x] **TASK-005**: Write failing test for JSON with different whitespace
  - **Status**: Completed
  - **Notes**: Test created, but discovered current implementation already handles some whitespace cases
  - **Date**: 2025-09-29
- [x] **TASK-006**: Write failing test for JSON with different indentation
  - **Status**: Completed
  - **Notes**: Test created, but discovered current implementation already handles some indentation cases
  - **Date**: 2025-09-29
- [x] **TASK-007**: Write failing test for JSON with different line breaks
  - **Status**: Completed
  - **Notes**: Test created and failing as expected for complex nested JSON
  - **Date**: 2025-09-29
- [ ] **TASK-008**: Write failing test for mixed JSON/non-JSON scenarios
- [ ] **TASK-009**: Write failing test for JSON with placeholders support

### Phase 3: Implementation
- [ ] **TASK-010**: Implement `isJSONContent()` helper function
- [ ] **TASK-011**: Implement `normalizeJSON()` function
- [ ] **TASK-012**: Implement `compareJSONBodies()` function
- [ ] **TASK-013**: Integrate JSON comparison into main `compareBodies()` function
- [ ] **TASK-014**: Ensure placeholder compatibility with JSON comparison

### Phase 4: Component Tests
- [ ] **TASK-015**: Create test data files with various JSON formatting scenarios
- [ ] **TASK-016**: Write component tests using real HTTP responses
- [ ] **TASK-017**: Test integration with existing placeholder system
- [ ] **TASK-018**: Test error handling for malformed JSON

### Phase 5: Documentation and Refinement
- [ ] **TASK-019**: Update validator documentation with JSON comparison features
- [ ] **TASK-020**: Add JSON comparison examples to testing guidelines
- [ ] **TASK-021**: Performance testing and optimization
- [ ] **TASK-022**: Final code review and cleanup

## Key Findings from Research

### JSON Patterns in .hresp Files
1. **Placeholder Types**:
   - `{{$anyGuid}}` - UUID validation
   - `{{$anyTimestamp}}` - Unix timestamp validation
   - `{{$anyDatetime format}}` - Datetime with format specification
   - `{{$any}}` - Any non-empty value

2. **Formatting Variations**:
   - Single-line: `{"key":"value"}`
   - Multi-line with indentation: 2-space and 4-space patterns
   - Mixed formatting in same files (multiple responses)

3. **Content Structure**:
   - Simple key-value pairs
   - Nested objects (GraphQL responses)
   - Arrays and complex nested structures

### Current Implementation Analysis
- **File**: `validator.go:368-416`
- **Function**: `compareBodies()`
- **Current Logic**: 
  - Normalizes line endings (`\r\n` → `\n`)
  - Trims whitespace
  - Fast path for non-placeholder content (direct string comparison)
  - Placeholder path using regex compilation and matching
- **Issue**: String comparison is whitespace/formatting sensitive

## Design Decisions Needed

### JSON Detection Approach
**Option A: Content-Type Header Check**
- Pros: Fast, simple, no parsing overhead
- Cons: Unreliable (headers can be missing/incorrect)

**Option B: JSON Parsing Attempt**
- Pros: More reliable, works regardless of headers
- Cons: Performance overhead, error handling complexity

**Recommended**: Option B - Attempt JSON parsing on both bodies, fallback to existing logic if either fails

### JSON Normalization Strategy
- Parse JSON using `encoding/json`
- Re-serialize with consistent formatting (no extra whitespace)
- Preserve placeholder compatibility by processing placeholders after normalization

## Next Steps
1. Complete TASK-003: Design JSON detection heuristic
2. Start Phase 2: Write failing unit tests (TDD approach)
3. Implement JSON normalization functions