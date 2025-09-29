# JSON Comparison Whitespace Agnostic PRD

## Problem Statement

The current JSON comparison in `validator.go` is sensitive to whitespace and formatting differences. When comparing JSON responses, differences in indentation, line breaks, and spacing cause validation failures even when the JSON content is semantically identical. This creates false positives in response validation and reduces test reliability.

## Current State Analysis

Looking at `validator.go:368-416`, the `compareBodies` function performs basic string normalization:
- Trims whitespace from both ends
- Replaces `\r\n` with `\n`
- For non-JSON content: Uses direct string comparison or regex with placeholders
- No JSON-specific normalization or parsing

## Proposed Solution

Implement JSON-aware comparison that normalizes formatting before comparison:

### 1. JSON Detection and Normalization
- Detect if content is JSON (by checking `Content-Type` header or attempting to parse)
- If JSON: Parse, then re-serialize with consistent formatting
- If not JSON: Use existing comparison logic

### 2. Comparison Strategy
```go
func compareBodies(responseFilePath string, responseIndex int, expectedBody, actualBody string) error {
    // Check if both bodies are JSON
    if isJSONContent(expectedBody) && isJSONContent(actualBody) {
        return compareJSONBodies(responseFilePath, responseIndex, expectedBody, actualBody)
    }
    
    // Fall back to existing logic for non-JSON content
    return compareBodiesExisting(responseFilePath, responseIndex, expectedBody, actualBody)
}
```

### 3. JSON Normalization Function
```go
func normalizeJSON(jsonStr string) (string, error) {
    var data interface{}
    if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
        return "", err
    }
    
    normalized, err := json.Marshal(data)
    if err != nil {
        return "", err
    }
    
    return string(normalized), nil
}
```

## Task List

### Phase 1: Research and Planning
- [x] **TASK-001**: Analyze existing test cases for JSON comparison scenarios
- [x] **TASK-002**: Review existing `.hresp` files to understand JSON validation patterns
- [x] **TASK-003**: Design JSON detection heuristic (Content-Type vs content parsing)
- [x] **TASK-004**: Create task tracking document

### Phase 2: Unit Tests (TDD Approach)
- [x] **TASK-005**: Write failing test for JSON with different whitespace
- [x] **TASK-006**: Write failing test for JSON with different indentation
- [x] **TASK-007**: Write failing test for JSON with different line breaks
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

## Success Criteria

### Functional Requirements
- [ ] JSON responses with different whitespace are considered equal
- [ ] JSON responses with different indentation are considered equal  
- [ ] JSON responses with different line breaks are considered equal
- [ ] Non-JSON responses continue to work with existing logic
- [ ] Placeholder system (`{{$regexp}}`, `{{$anyGuid}}`, etc.) works with JSON
- [ ] Error handling for malformed JSON is robust

### Non-Functional Requirements
- [ ] Performance impact is minimal (<10% slowdown for JSON comparisons)
- [ ] Backward compatibility is maintained
- [ ] Code follows existing patterns and conventions
- [ ] Test coverage >90% for new functionality
- [ ] No breaking changes to existing API

## Testing Strategy

Following the testing guidelines:

### Unit Tests
- Location: `validator_test.go` (same package)
- Focus: Individual functions for JSON detection and normalization
- Mock external dependencies when needed
- Use Given/When/Then structure

### Component Tests  
- Location: `test/validator_json_test.go`
- Focus: End-to-end JSON comparison with real HTTP responses
- Use `.http` and `.hresp` files for test data
- Test various JSON formatting scenarios

### Test Data Examples
```json
// Expected (formatted)
{
  "name": "test",
  "value": 42
}

// Actual (minified) - should match
{"name":"test","value":42}

// Actual (different whitespace) - should match
{  "name"  :  "test"  ,  "value"  :  42  }
```

## Implementation Notes

### Key Considerations
1. **Placeholder Compatibility**: Must work with existing `{{$regexp}}`, `{{$anyGuid}}` placeholders
2. **Error Handling**: Graceful fallback for malformed JSON
3. **Performance**: JSON parsing has cost, use only when beneficial
4. **Backward Compatibility**: Non-JSON content must continue working

### Technical Approach
- Use Go's `encoding/json` for parsing and serialization
- Detect JSON by attempting to parse (more reliable than Content-Type)
- Maintain existing placeholder logic by applying it before JSON normalization
- Preserve existing error messages and diff formatting

## Risks and Mitigations

### Risks
- **Performance Impact**: JSON parsing adds overhead
- **Complexity**: Additional code paths increase maintenance burden
- **Edge Cases**: Malformed JSON handling may be tricky

### Mitigations
- **Performance**: Only attempt JSON parsing when both bodies appear to be JSON
- **Complexity**: Keep JSON logic isolated and well-tested
- **Edge Cases**: Comprehensive test coverage for malformed scenarios
- **Fallback**: Always fall back to existing logic on JSON errors

## Timeline Estimate

- **Phase 1**: 2 hours (research and planning)
- **Phase 2**: 4 hours (unit tests)
- **Phase 3**: 6 hours (implementation)
- **Phase 4**: 4 hours (component tests)
- **Phase 5**: 2 hours (documentation)

**Total Estimated Effort**: 18 hours