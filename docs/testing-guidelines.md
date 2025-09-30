# Testing Guidelines

This document defines testing rules and patterns for the go-restclient project.

## Test Types

### 1. Unit Tests
- **Location**: Same package as code being tested (`*_test.go` files)
- **Package Name**: Same as source package
- **Dependencies**: Mock external dependencies when needed
- **Scope**: Test individual functions and methods
- **Speed**: Fast execution
- **Isolation**: Minimal external dependencies

### 2. Component Tests
- **Location**: `test/` directory with test data files
- **Package Name**: Same as source package
- **Dependencies**: Real HTTP requests to test servers
- **Scope**: Test complete client functionality with real HTTP interactions
- **Speed**: Moderate execution
- **Data**: Use `.http` and `.hresp` files for request/response testing

## Test Organization

### File Naming
- Unit tests: `<component>_test.go` in same directory as code
- Component tests: `test/<component>_test.go` with supporting data files
- Test data: `test/data/` directory with `.http` and `.hresp` files

### Test Data Structure
```
test/
├── data/
│   ├── http_request_files/     # HTTP request test files
│   ├── http_response_files/    # HTTP response validation files
│   ├── graphql/               # GraphQL test scenarios
│   ├── system_variables/      # System variable tests
│   └── request_body/          # Test request body files
├── client_execute_*.go        # Component test files
├── validator_*.go            # Validation test files
└── test_helpers.go           # Shared test utilities
```

## Running Tests

### Commands
```bash
# Unit tests only (fast feedback)
make test-unit

# All tests (unit + component)
make test

# Pre-commit check (unit tests + lint only - fast)
make check

# Run single test
go test -run TestName ./...
```

### Test Execution
- Unit tests: `go test ./...` (includes all test files)
- Component tests: Same command, includes test data files
- Test data: Automatically loaded from `test/data/` directory

## Test Structure

### Unit Test Structure
```go
func TestFunction_Scenario_ExpectedResult(t *testing.T) {
    // given
    setup := setupTestData()
    
    // when
    result, err := functionUnderTest(setup.input)
    
    // then
    assert.NoError(t, err)
    assert.Equal(t, setup.expected, result)
}
```

### Component Test Structure
```go
func TestClient_ExecuteHTTPRequest(t *testing.T) {
    // given
    client := NewClient()
    testData := loadHTTPFile("simple_get.http")
    
    // when
    result, err := client.ExecuteFile(testData)
    
    // then
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, result.StatusCode)
}
```

## Test Data Management

### HTTP Request Files
- Use `.http` extension for request definitions
- Include all request methods, headers, and bodies
- Support variable substitution and system variables

### HTTP Response Files
- Use `.hresp` extension for expected responses
- Include status codes, headers, and body validation
- Support validation placeholders and patterns

### Test Helpers
- Use `t.Helper()` in helper functions only
- Create reusable setup/teardown functions
- Share common test utilities in `test/test_helpers.go`

## Mocking Guidelines

### Mock Creation
Use `testify/mock` for external dependencies:
```go
type MockHTTPClient struct {
    mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    args := m.Called(req)
    return args.Get(0).(*http.Response), args.Error(1)
}
```

### Mock Assertions
Always verify mock expectations:
```go
mockClient.On("Do", mock.Anything).Return(expectedResponse, nil)
// ... test code ...
mockClient.AssertExpectations(t)
```

## Quality Standards

### Coverage
- Unit tests: Aim for >80% coverage of public APIs
- Component tests: Cover critical HTTP client workflows
- Focus on core functionality over edge cases

### Performance
- Unit tests: Fast execution (<100ms each)
- Component tests: Moderate execution (<1s each)
- Avoid unnecessary delays in test setup

### Reliability
- Tests must be deterministic (no flaky tests)
- Use fixed test data, not random generation
- Handle network timeouts gracefully in component tests
- Clean up test resources properly

## TDD Process

### Red-Green-Refactor
1. **Red**: Write failing test first
2. **Green**: Write minimal code to pass test
3. **Refactor**: Improve code while keeping tests green
4. **Repeat**: Continue cycle for all features

### Test-First Development
- Write unit test before implementation
- Write component test for HTTP interactions
- Use test data files to define expected behavior
- Never skip tests - they are part of the implementation

## Continuous Integration

### Pre-commit Requirements
**`make check` is the main development validation command:**

```bash
make check  # Runs: lint + test-unit
```

**Components of `make check`:**
- **`make lint`** - golangci-lint comprehensive code quality checks
- **`make test-unit`** - Fast unit tests with coverage

### Test Execution Order
1. Unit tests (fastest)
2. Component tests (moderate)
Fail fast - stop on first test failure in CI.

## Zero Tolerance Policy

### Unacceptable Test States
- **❌ FAILING TESTS**: Any test that returns non-zero exit code
- **❌ FLAKY TESTS**: Tests that pass/fail inconsistently across runs
- **❌ TIMEOUT TESTS**: Tests that exceed timeout limits

### Enforcement Rules
- **BLOCKING**: No commits allowed with failing tests
- **IMMEDIATE FIX**: All failing tests must be resolved before continuing development
- **100% PASS RATE**: Only tests that consistently pass are acceptable

This policy ensures production-ready code quality and reliable automated testing pipelines.