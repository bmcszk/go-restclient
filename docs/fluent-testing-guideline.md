# Fluent Test Design Standards (MANDATORY)

This document provides comprehensive standards for writing fluent E2E tests using the Given/When/Then pattern. All E2E tests MUST follow these exact patterns and rules.

## Fluent Test Builder Pattern
**Based on TVN-cue-crab reference implementation - use fluent interfaces for readable test composition:**

```go
package e2e_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

type parts struct {
    *testing.T
    responseCode int
    responseBody []byte
    videoID      string
    checks       *checks
}

func newParts(t *testing.T) (*parts, *parts, *parts) {
    t.Helper()

    parts := &parts{
        T:      t,
        checks: newChecks(),
    }

    // Setup test environment
    parts.setupTestEnvironment()

    return parts, parts, parts  // given, when, then
}

// Fluent chaining methods
func (p *parts) and() *parts {
    return p
}

func (p *parts) uploadVideoFile(filename, title string) *parts {
    // Implementation
    return p
}

func (p *parts) videoUploadedSuccessfully() *parts {
    assert.Equal(p.T, http.StatusCreated, p.responseCode)
    return p
}

func (p *parts) noError() *parts {
    assert.NoError(p.T, p.checks.repeatRun())
    return p
}
```

### Test Usage Example
**Clean, readable test structure with fluent chaining:**

```go
func TestVideoUploadAndRetrieval(t *testing.T) {
    videoFilePath := "./assets/test-video.mp4"
    given, when, then := newParts(t)

    when.
        uploadVideoFile(videoFilePath, "E2E Test Video")

    then.
        videoUploadedSuccessfully().and().
        getVideoByID(then.videoID).and().
        videoFoundSuccessfully()
}

func TestAssetEventProcessing(t *testing.T) {
    videoFilePath := "./assets/test-video.mp4"
    given, when, then := newParts(t)

    given.
        uploadVideoFile(videoFilePath, "Asset Event E2E Test Video").and().
        videoUploadedSuccessfully().and().
        getVideoByID(given.videoID)

    when.
        sendAssetEvent("ACTIVATE", given.videoID).and().
        messageSentSuccessfully()

    then.
        videoHasStatus("ready").and().
        videoHasAssets(2).and().
        noError()
}
```

## E2E Test Structure Requirements

**ALL E2E tests MUST follow the exact Given/When/Then structure with these specific patterns:**

```go
func TestWorkflow_CriticalUserAction(t *testing.T) {
    given, when, then := newParts(t)

    given.
        setupTestData().and().
        configureServices()

    when.
        executeCriticalAction()

    then.
        actionSucceeded().and().
        noError()
}
```

## MANDATORY Formatting Rules

### 1. Given/When/Then Section Structure
**Every E2E test MUST use this exact pattern:**

```go
// ✅ CORRECT: Proper Given/When/Then structure
func TestAPI_ResourceCreation(t *testing.T) {
    given, when, then := newParts(t)

    given.
        setupResourceScenario()

    when.
        createResourceViaAPI()

    then.
        resourceCreatedSuccessfully().and().
        responseContainsExpectedData()
}

// ❌ FORBIDDEN: Missing then section
func TestAPI_ResourceCreation(t *testing.T) {
    given, when, _ := newParts(t)

    given.
        setupResourceScenario()

    when.
        createResourceViaAPI().and().
        resourceCreatedSuccessfully()  // WRONG: validation in when section
}
```

### 2. Fluent Method Chaining Format
**ALL fluent chains must follow this exact formatting:**

```go
// ✅ CORRECT: Proper fluent formatting
given.
    setupScenario().and().
    configureServices()

when.
    executeAction()

then.
    verifyResult().and().
    cleanupState()
```

### 3. Method Positioning Rules
**Follow these rules for method chaining:**

```go
// ✅ CORRECT: .and() at end of line, continuation on next line
when.
    executeHTTPRequest("/api/resource").and().
    validateResponseCode(201)

// ✅ CORRECT: Multiple validations chained
then.
    responseCodeIs(200).and().
    responseContains("success").and().
    noError()

// ❌ FORBIDDEN: .and() at start of line
when.
    executeHTTPRequest("/api/resource")
        .and().validateResponseCode(201)  // WRONG: .and() mispositioned
```

### 4. Helper Function Usage Rules
**ALL setup operations MUST use fluent methods, NEVER direct helper function calls:**

```go
// ❌ FORBIDDEN: Direct helper function call in test
func TestScenarioFileLoading(t *testing.T) {
    configFile := createTestUnifiedConfigFile(t)  // WRONG: Direct call
    _, when, then := newServerParts(t, configFile)

    when.
        a_get_request_is_made_to("/api/resource")

    then.
        the_response_is_successful()
}

// ❌ FORBIDDEN: Direct config creation with local variable
func TestPerformanceTest(t *testing.T) {
    configFile := createPerformanceTestConfig(t)  // WRONG: Local variable
    when, then := newServerParts(t, configFile)   // WRONG: Passing as parameter

    when.
        a_load_test_is_executed()

    then.
        the_response_time_is_acceptable()
}

// ✅ CORRECT: Use fluent method in given section
func TestScenarioFileLoading(t *testing.T) {
    given, when, then := newParts(t)

    given.
        unifiedConfigFile()  // Fluent method that creates config and stores in parts.configFile

    when.
        a_get_request_is_made_to("/api/resource")

    then.
        the_response_is_successful()
}

// ✅ CORRECT: Config creation via fluent method
func TestPerformanceTest(t *testing.T) {
    given, when, then := newParts(t)

    given.
        performanceTestConfig()  // Creates config, stores in parts.configFile

    when.
        a_load_test_is_executed()  // Accesses parts.configFile internally

    then.
        the_response_time_is_acceptable()
}
```

**Critical Rules:**
- NEVER create config files with direct function calls like `configFile := createXXX(t)`
- NEVER pass config files as parameters to factory functions like `newServerParts(t, configFile)`
- ALWAYS create config files via fluent methods in the `given` section
- ALWAYS store config file paths in `parts.configFile`
- Config creation methods MUST return `*parts` for chaining

**Rationale:**
- Maintains consistent fluent pattern throughout tests
- Improves test readability and maintainability
- Makes test setup explicit and traceable
- Supports method chaining for complex setup scenarios
- Eliminates local variables and parameter passing

### 5. Single Given Chain Rule
**Each test MUST have only ONE given statement that chains all setup steps:**

```go
// ❌ FORBIDDEN: Multiple given statements
func TestComplexSetup(t *testing.T) {
    given, when, then := newParts(t)

    given.setupDatabase()        // WRONG: Separate statements
    given.createTestData()       // WRONG: Separate statements
    given.configureServices()    // WRONG: Separate statements

    when.executeAction()

    then.verifyResult()
}

// ✅ CORRECT: Single given chain
func TestComplexSetup(t *testing.T) {
    given, when, then := newParts(t)

    given.
        setupDatabase().and().
        createTestData().and().
        configureServices()

    when.
        executeAction()

    then.
        verifyResult()
}

// ✅ CORRECT: Single given method (when no chaining needed)
func TestSimpleSetup(t *testing.T) {
    given, when, then := newParts(t)

    given.
        unifiedConfigFile()

    when.
        a_get_request_is_made_to("/api/resource")

    then.
        the_response_is_successful()
}
```

**Rationale:**
- Forces proper fluent chaining pattern
- Ensures all setup methods return `*parts` for chaining
- Makes the setup phase atomic and clear
- Prevents scattered setup statements
- All test state MUST be stored in the `parts` struct

**Critical Rule: All Test State in Parts Struct**
```go
// ❌ FORBIDDEN: Local variables for test state
func TestBadPattern(t *testing.T) {
    given, when, then := newParts(t)

    configFile := createConfigFile(t)  // WRONG: State not in parts
    userID := "123"                    // WRONG: State not in parts

    given.setupWithConfig(configFile)  // Passing state as parameter
    when.createUser(userID)
    then.verifyUser(userID)
}

// ✅ CORRECT: All state in parts struct
type parts struct {
    *testing.T
    require    *require.Assertions
    configFile string  // State stored in parts
    userID     string  // State stored in parts
    response   *http.Response
}

func (p *parts) unifiedConfigFile() *parts {
    // Create config and store in parts
    p.configFile = createAndStoreConfig(p.T)
    p.setupScenarioTestServer(p.configFile)
    return p
}

func (p *parts) createUserWithID(id string) *parts {
    p.userID = id  // Store state in parts
    // ... create user logic
    return p
}

func TestCorrectPattern(t *testing.T) {
    given, when, then := newParts(t)

    given.
        unifiedConfigFile()  // Config stored in parts.configFile

    when.
        createUserWithID("123")  // ID stored in parts.userID

    then.
        verifyUserCreated()  // Accesses parts.userID internally
}
```

**Why This Matters:**
- Enables method chaining without parameter passing
- Makes test state accessible across given/when/then phases
- Supports cleanup and assertions that need access to setup data
- Prevents test state from being scattered across local variables

## Variable Declaration Rules

### 1. Section Variable Declaration
**Declare variables based on what sections are actually used:**

```go
// ✅ CORRECT: Only declare variables that are used
func TestWithGivenAndWhen(t *testing.T) {
    given, when, then := newParts(t)
    // Uses all three sections
}

func TestWithWhenAndThen(t *testing.T) {
    _, when, then := newParts(t)
    // No setup needed
}

func TestWithWhenOnly(t *testing.T) {
    _, when, _ := newParts(t)
    // Simple test with no verification
}
```

### 2. Unused Variable Elimination
**NEVER declare unused variables:**

```go
// ❌ FORBIDDEN: Unused given variable
func TestSimpleAction(t *testing.T) {
    given, when, then := newParts(t)  // given is unused
    when.
        executeAction()
    then.
        verifyResult()
}

// ✅ CORRECT: Only declare what's needed
func TestSimpleAction(t *testing.T) {
    _, when, then := newParts(t)
    when.
        executeAction()
    then.
        verifyResult()
}
```

## Response Handling Patterns

### 1. Response Storage Pattern
**Store responses in the parts struct for better readability:**

```go
// ✅ CORRECT: Store responses in parts struct
func (p *parts) executeHTTPRequest(endpoint string) *parts {
    responses, err := p.httpClient.Get(endpoint)
    p.require.NoError(err, "Failed to execute HTTP request")
    p.responses = make([]any, len(responses))
    for i, resp := range responses {
        p.responses[i] = resp
    }
    return p
}

func (p *parts) validateResponse() *parts {
    // Access stored responses from parts struct
    resp := p.responses[0].(*http.Response)
    p.require.Equal(200, resp.StatusCode)
    return p
}
```

### 2. Method Return Value Pattern
**Fluent methods should return *parts for chaining:**

```go
// ✅ CORRECT: Methods support fluent chaining
func (p *parts) createResource() *parts {
    // Implementation
    return p
}

// ✅ CORRECT: Usage in fluent chain
when.
    createResource().and().
    verifyResource()
```

## Migration Guidelines: Standard to E2E Tests

### 1. Migration Decision Matrix
**Use this guide to determine when to create E2E tests:**

| Test Need | Recommended Approach | Migration Strategy |
|-----------|---------------------|-----------------|
| Single function unit logic | **Unit Test** | Add unit tests if missing |
| Service interaction | **Integration Test** | Create integration tests |
| Complete user workflow | **E2E Test** | Create E2E test with Given/When/Then |
| API contract validation | **E2E Test** | Convert integration tests to E2E |
| Frontend user journey | **Browser Test** | Create browser tests with chromedp |

### 2. Standard Test to E2E Migration Process

**Step 1: Identify Migration Candidates**
```bash
# Find integration tests that cover user workflows
find test/integration -name "*integration_test.go" | \
    xargs grep -l "workflow\|journey\|end-to-end"
```

**Step 2: Analyze Current Test Structure**
```go
// BEFORE: Standard integration test
func TestUserService_CreateUser(t *testing.T) {
    db := setupTestDatabase(t)
    defer db.Close()

    service := NewUserService(db)
    user := &User{Name: "Test User", Email: "test@example.com"}

    created, err := service.CreateUser(user)
    assert.NoError(t, err)
    assert.Equal(t, user.Name, created.Name)
    assert.NotZero(t, created.ID)
}
```

**Step 3: Convert to Given/When/Then Structure**
```go
// AFTER: Migrated to E2E with fluent pattern
func TestUserService_CreateUser(t *testing.T) {
    given, when, then := newParts(t)

    given.
        setupTestDatabase().and().
        initializeUserService()

    when.
        createUserWithName("Test User").and().
        createUserWithEmail("test@example.com")

    then.
        userCreatedSuccessfully().and().
        userHasCorrectName("Test User").and().
        userHasValidID()
}
```

### 3. E2E Test Implementation Template
**Use this template for new E2E tests:**

```go
package e2e_test

import (
    "testing"
    "github.com/stretchr/testify/require"
)

type parts struct {
    *testing.T
    require     *require.Assertions
    // Add test-specific fields
    userID      string
    response    *http.Response
    httpClient  *http.Client
}

func newParts(t *testing.T) (*parts, *parts, *parts) {
    t.Helper()

    parts := &parts{
        T:        t,
        require:  require.New(t),
        // Initialize test-specific fields
    }

    parts.setupTestEnvironment()
    return parts, parts, parts
}

// Fluent helper methods
func (p *parts) and() *parts {
    return p
}

func (p *parts) setupTestEnvironment() *parts {
    // Implementation
    return p
}

// Given section methods
func (p *parts) initializeUserService() *parts {
    // Implementation
    return p
}

// When section methods
func (p *parts) createUserWithDetails(details map[string]string) *parts {
    // Implementation
    return p
}

// Then section methods
func (p *parts) userCreatedSuccessfully() *parts {
    // Implementation
    return p
}

// Test implementation
func TestUserService_CompleteUserCreation(t *testing.T) {
    given, when, then := newParts(t)

    given.
        initializeUserService()

    when.
        createUserWithDetails(map[string]string{
            "name":  "John Doe",
            "email": "john@example.com",
        })

    then.
        userCreatedSuccessfully().and().
        userExistsInDatabase().and().
        userHasCorrectDetails()
}
```

## E2E Test Quality Standards

### 1. Test Completeness Requirements
**Every E2E test must cover:**
- ✅ Setup phase (given)
- ✅ Action phase (when)
- ✅ Verification phase (then)
- ✅ Cleanup (if needed)
- ✅ Error handling validation

### 2. Readability Standards
**Tests must be self-documenting:**
- ✅ Clear test names: `TestComponent_Action_ExpectedResult`
- ✅ Descriptive method names
- ✅ Proper section comments
- ✅ Meaningful assertion messages

### 3. Maintainability Requirements
**Tests must be maintainable:**
- ✅ Reusable helper methods
- ✅ Clear separation of concerns
- ✅ Consistent error handling
- ✅ Environment-independent execution

## Anti-Patterns to Avoid

### ❌ Structural Anti-Patterns
```go
// FORBIDDEN: Missing then section
func TestBad(t *testing.T) {
    given, when, _ := newParts(t)
    when.executeAction()  // No verification
}

// FORBIDDEN: Validation in wrong section
func TestBad(t *testing.T) {
    given, when, then := newParts(t)
    when.executeAction().and().verifyResult()  // Wrong section
}

// FORBIDDEN: Unused variables
func TestBad(t *testing.T) {
    given, when, then := newParts(t)  // given, then unused
    when.executeAction()
    then.verifyResult()
}
```

### ❌ Formatting Anti-Patterns
```go
// FORBIDDEN: Inconsistent chaining
when.
    method1().
        method2()  // Missing .and().
    then.verify()

// FORBIDDEN: Misplaced .and()
when.
    method1().
        .and().method2()  // Wrong indentation

// FORBIDDEN: Long single-line chains
when.method1().and().method2().and().method3().and().method4()  // Too long
```

## Advanced Patterns

### Asynchronous Check Pattern
**Robust pattern for testing eventual consistency with retry logic:**

```go
type checks struct {
    interval time.Duration
    timeout  time.Duration
    checks   []*check
}

type check struct {
    f   func() error
    ok  bool
    err error
}

func newChecks() *checks {
    // CI-aware defaults
    interval := 500 * time.Millisecond
    timeout := 20 * time.Second

    if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
        interval = 2 * time.Second
        timeout = 60 * time.Second
    }

    return &checks{
        interval: interval,
        timeout:  timeout,
        checks:   make([]*check, 0, 10),
    }
}

func (cs *checks) add(f func() error) {
    cs.checks = append(cs.checks, &check{f: f})
}

func (cs *checks) repeatRun() error {
    tick := time.Tick(cs.interval)
    limit := time.After(cs.timeout)

    for {
        select {
        case <-limit:
            return cs.error()
        case <-tick:
            cs.run()
            if cs.ok() {
                return nil
            }
        }
    }
}

// Usage in test methods
func (p *parts) videoHasStatus(expectedStatus string) *parts {
    p.checks.add(func() error {
        resp, err := p.httpClient.Get("http://" + p.envs.ServiceAddress + "/videos/" + p.videoID)
        if err != nil {
            return fmt.Errorf("get video by ID request: %w", err)
        }
        defer resp.Body.Close()

        // Parse and validate status
        var videoResp struct { Status string `json:"status"` }
        if err := json.NewDecoder(resp.Body).Decode(&videoResp); err != nil {
            return fmt.Errorf("parse video response: %w", err)
        }

        if videoResp.Status != expectedStatus {
            return fmt.Errorf("expected status %s, got %s", expectedStatus, videoResp.Status)
        }

        return nil
    })
    return p
}
```

### Template-Based Test Data Pattern
**Clean approach for managing complex test data with templates:**

```go
// Template files in templates/ directory
// templates/asset_event.json
{
    "Type": "{{.Type}}",
    "TenantUUID": "{{.TenantUUID}}",
    "ExternalUUID": "{{.ExternalUUID}}",
    "Name": "{{.Name}}",
    "Active": {{.Active}},
    "Since": "{{.Since}}",
    "TechnicalTags": "{{.TechnicalTags}}"
}

// Test method using templates
func (p *parts) sendAssetEvent(eventType, videoID string) *parts {
    eventTime := time.Now().Add(1 * time.Second).Format(time.RFC3339)
    return p.sendMessageFromTemplate("asset_event.json", map[string]any{
        "Type":          eventType,
        "TenantUUID":    "test-tenant-uuid",
        "ExternalUUID":  videoID,
        "Name":          "E2E Test Video",
        "Active":        true,
        "Since":         eventTime,
        "TechnicalTags": "",
    })
}

func (p *parts) sendMessageFromTemplate(templateFile string, data map[string]any) *parts {
    tmpl, err := template.ParseFiles(filepath.Join("templates", templateFile))
    assert.NoError(p.T, err, "parse template file")

    var buf bytes.Buffer
    err = tmpl.Execute(&buf, data)
    assert.NoError(p.T, err, "execute template")

    return p.sendSQSMessage(buf.String())
}
```

### 6. t.Run Promotion Rule
**Prefer separate top-level Test funcs over t.Run subtests.** Only use t.Run when a subtest genuinely needs state isolation (fresh parts, fresh server, fresh env). t.Run that exists only to share a "name with a twist" is an anti-pattern — split into separate Test funcs with descriptive names.

```go
// ❌ FORBIDDEN: t.Run wrapper only to share a name
func TestCookieJarHandling(t *testing.T) {
    t.Run("default_behavior", func(t *testing.T) {
        given, when, then := newParts(t)
        // ...
    })
    t.Run("directive_no_cookie_jar", func(t *testing.T) {
        given, when, then := newParts(t)
        // ...
    })
}

// ✅ CORRECT: two top-level Test funcs, one per scenario
func TestCookieJarHandling_DefaultBehavior(t *testing.T) {
    given, when, then := newParts(t)
    // ...
}

func TestCookieJarHandling_NoCookieJarDirective(t *testing.T) {
    given, when, then := newParts(t)
    // ...
}
```

**Rationale:**
- Simpler test func bodies; complexity stays in the DSL
- `go test -run TestCookieJarHandling_NoCookieJarDirective` works for targeted runs
- No wrapper Test funcs — every test is directly runnable
- Junit reports one clean entry per test

### 7. Fixtures Over Inline Strings
**aHttpFile(content) accepts SHORT inline HTTP requests only.** For multi-line request bodies with JSON templates, multi-line bodies with @vars, or `<@` / `<` external-file references, commit a fixture in `test/data/` and use `aHttpFileFromTemplate(name)`. Tests must not encode request bodies with sprintf + newlines + escapes.

```go
// ❌ FORBIDDEN: inline sprintf'd .http body
anExternalFile("test_vars.json", `...long inline JSON...`).and().
aHttpFile(fmt.Sprintf(`@userId = user123
POST %s/post
<@ ./test_vars.json`, given.serverURL))

// ✅ CORRECT: committed fixture + one DSL call
anExternalFile("test_vars.json", `...`).and().
aHttpFileFromTemplate("external_file_with_vars.http")
```

**Rationale:**
- Fixture files diff cleanly and are reviewable
- No escaping pitfalls in Go raw strings
- Template variables ([[.ServerURL]]) keep the server URL dynamic

### 8. Handler Assertions via DSL
**aHttpServer's handler must NOT contain inline `if r.Method != X { t.Errorf(...) }` cascades.** Keep the handler short (echo back / 200 OK). Assertions belong in the then-chain via `serverReceivedMethodAndPath` / `serverReceivedHeader*` / `serverReceivedBody*` / `serverReceivedJSONBody` reading capturedRequests.

```go
// ❌ FORBIDDEN: handler with inline t.Errorf cascade
aHttpServer(func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        t.Errorf("method = %v", r.Method)
    }
    if got := r.Header.Get("Content-Type"); got != "application/json" {
        t.Errorf("Content-Type = %q", got)
    }
    // ...
})

// ✅ CORRECT: short echo handler + DSL assertions in then
aJsonEchoServer().and().
    // ... other given

then.
    serverReceivedMethodAndPath(0, http.MethodPost, "/post").and().
    serverReceivedHeaderValue(0, "Content-Type", "application/json").and().
    serverReceivedBodyParsesAsJSON(0)
```

**Rationale:**
- Handler stays a plain test-data function (no *testing.T dependency)
- Assertion failures report through the fluent chain, not buried t.Errorf
- Captured requests can be asserted multiple ways without re-serving

### 9. Faker/Rule Data Structures Belong in DSL
**When a test needs a list of validation rules (key, pattern, fieldCount, containsCheck), expose ONE DSL method taking a small struct slice — not per-fixture helper functions that build the slice from a separate file.**

```go
// ❌ FORBIDDEN: per-fixture rule builders in the test file
func fakerPersonDataHeaders() (vsCode, jetBrains []fakerHeaderRule) { /* ... */ }
func fakerContactInternetHeaders() (vsCode, jetBrains []fakerHeaderRule) { /* ... */ }

// Test:
vsCode, jetBrains := fakerPersonDataHeaders()
then.
    assertFakerHeaders(0, vsCode).and().
    assertFakerHeaders(1, jetBrains)

// ✅ CORRECT: one DSL method + inline rule literal
then.
    serverReceivedFakerHeaders(0, []fakerHeaderRule{
        {key: "X-Random-First-Name", pattern: `^\S+$`, fieldCount: 1},
        {key: "X-Random-Job-Title", pattern: `^.+$`, fieldCount: -1},
        {key: "X-User-Agent", containsCheck: "Mozilla"},
    }).and().
    serverReceivedFakerHeaders(1, []fakerHeaderRule{ /* ... */ })
```

**Rationale:**
- Rules visible inline next to the test that owns them
- No indirection through builder functions
- One DSL entry point; rule types shared across faker tests

## Test Helper Functions
**ONLY helper functions MUST use `t.Helper()` (NOT test functions).** All helpers should follow Given/When/Then structure:

```go
func setupTestData(t *testing.T) TestData {
    t.Helper()

    data := TestData{
        Field1: "value1",
        Field2: "value2",
    }

    return data
}

// Enhanced fluent helper
func (p *parts) cleanupDatabase() *parts {
    if p.db == nil {
        p.Log("Warning: database connection is nil, skipping cleanup")
        return p
    }

    queries := []string{
        "DELETE FROM crab.assets",
        "DELETE FROM crab.transcodings",
        "DELETE FROM crab.videos",
    }

    for _, query := range queries {
        _, err := p.db.Exec(query)
        if err != nil {
            p.Logf("Warning: failed to cleanup table with query '%s': %v", query, err)
        }
    }

    return p
}
```

---

## Test Environment Management (ENHANCED)

### CI-Aware Test Configuration
**Automatically adapt test behavior based on execution environment:**

```go
func newChecks() *checks {
    // Default values for local development
    interval := 500 * time.Millisecond
    timeout := 20 * time.Second

    // Use longer intervals for CI environments
    if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
        interval = 2 * time.Second
        timeout = 60 * time.Second
    }

    return &checks{
        interval: interval,
        timeout:  timeout,
        checks:   make([]*check, 0, 10),
    }
}
```

### Environment-Specific Test Behavior
**Gracefully handle different execution environments:**

```go
func (p *parts) cleanupDatabase() {
    if p.db == nil {
        p.Log("Warning: database connection is nil, skipping cleanup")
        return
    }

    queries := []string{
        "DELETE FROM crab.assets",
        "DELETE FROM crab.transcodings",
        "DELETE FROM crab.videos",
    }

    for _, query := range queries {
        _, err := p.db.Exec(query)
        if err != nil {
            p.Logf("Warning: failed to cleanup table with query '%s': %v", query, err)
        }
    }
}

func (p *parts) cleanupSQSQueue() {
    if p.sqsClient == nil {
        p.Log("Warning: SQS client is nil, skipping SQS cleanup")
        return
    }

    queueUrl, err := url.JoinPath(p.sqsConfig.QueueUrl, p.sqsConfig.QueueName)
    if err != nil {
        p.Logf("Warning: failed to build queue URL for cleanup: %v", err)
        return
    }

    _, err = p.sqsClient.PurgeQueue(p.ctx, &awssqs.PurgeQueueInput{
        QueueUrl: aws.String(queueUrl),
    })
    if err != nil {
        p.Logf("Warning: failed to purge SQS queue: %v", err)
    }
}
```

### Test Asset Management
**Organize test files and assets cleanly:**

```
test/
├── e2e/
│   ├── assets/           # Test data files (videos, images, etc.)
│   │   └── test-video.mp4
│   ├── templates/        # Template files for test data generation
│   │   ├── asset_event.json
│   │   └── encode_task_event.json
│   ├── .env.test        # Test environment configuration
│   ├── docker-compose.test.yml
│   ├── e2e_test.go      # Main test scenarios
│   ├── e2e_parts_test.go # Test helpers and fluent interface
│   └── e2e_checks_test.go # Asynchronous check implementation
└── integration/
    └── ...
```

### Configuration Management
**Clean separation of test configurations:**

```go
// .env.test
TEST_DB_URL=postgres://test:test@localhost:5433/testdb
TEST_SQS_QUEUE_URL=http://localhost:4566/000000000000/test-queue
TEST_SERVICE_ADDRESS=http://localhost:18082

// Test setup with environment loading
func newParts(t *testing.T) (*parts, *parts, *parts) {
    t.Helper()
    var envs config.Environment

    // Load test-specific environment
    err := godotenv.Load(".env.test")
    assert.NoError(t, err, "load test config")

    err = env.Parse(&envs)
    assert.NoError(t, err, "parse test config")

    // Continue with test setup...
}
```

## Test Readability and Maintainability (ENHANCED)

### Clean Test Structure Principles
**Based on TVN-cue-crab reference implementation - prioritize readability and maintainability:**

#### 1. Fluent Interface Pattern
**Use method chaining for readable test workflows:**

```go
// Clean, readable test with fluent chaining
func TestVideoUploadAndRetrieval(t *testing.T) {
    videoFilePath := "./assets/test-video.mp4"
    given, when, then := newParts(t)

    when.
        uploadVideoFile(videoFilePath, "E2E Test Video")

    then.
        videoUploadedSuccessfully().and().
        getVideoByID(then.videoID).and().
        videoFoundSuccessfully()
}

// Complex workflow with multiple steps
func TestAssetEventProcessing(t *testing.T) {
    videoFilePath := "./assets/test-video.mp4"
    given, when, then := newParts(t)

    given.
        uploadVideoFile(videoFilePath, "Asset Event E2E Test Video").and().
        videoUploadedSuccessfully().and().
        getVideoByID(given.videoID)

    when.
        sendAssetEvent("ACTIVATE", given.videoID).and().
        messageSentSuccessfully()

    then.
        videoHasStatus("ready").and().
        videoHasAssets(2).and().
        noError()
}
```

#### 2. Given/When/Then Structure Enforcement
**ALL tests must follow the Given/When/Then pattern:**

```go
func TestEndpointHealth(t *testing.T) {
    _, when, then := newParts(t)

    when.
        checkHealth()

    then.
        responseCodeIs(200)
}

// Tests with setup (given) use all three parts
func TestComplexWorkflow(t *testing.T) {
    given, when, then := newParts(t)

    given.
        setupTestData().and().
        configureServices()

    when.
        executeWorkflow()

    then.
        verifyExpectedResult().and().
        noError()
}
```

#### 3. Self-Documenting Test Methods
**Method names should clearly describe their purpose:**

```go
// ✅ Clear, descriptive method names
func (p *parts) videoUploadedSuccessfully() *parts
func (p *parts) videoHasStatus(expectedStatus string) *parts
func (p *parts) sendAssetEvent(eventType, videoID string) *parts
func (p *parts) messageSentSuccessfully() *parts

// ❌ Vague or unclear method names
func (p *parts) checkVideo() *parts
func (p *parts) doEvent() *parts
func (p *parts) verify() *parts
```

#### 4. Graceful Error Handling
**Provide meaningful error messages and handle failures gracefully:**

```go
func (p *parts) cleanupDatabase() {
    if p.db == nil {
        p.Log("Warning: database connection is nil, skipping cleanup")
        return
    }

    queries := []string{
        "DELETE FROM crab.assets",
        "DELETE FROM crab.transcodings",
        "DELETE FROM crab.videos",
    }

    for _, query := range queries {
        _, err := p.db.Exec(query)
        if err != nil {
            p.Logf("Warning: failed to cleanup table with query '%s': %v", query, err)
        }
    }
}

func (p *parts) videoHasStatus(expectedStatus string) *parts {
    p.checks.add(func() error {
        resp, err := p.httpClient.Get("http://" + p.envs.ServiceAddress + "/videos/" + p.videoID)
        if err != nil {
            return fmt.Errorf("get video by ID request: %w", err)
        }
        defer resp.Body.Close()

        // ... validation logic with detailed error messages
        if videoResp.Status != expectedStatus {
            return fmt.Errorf("expected status %s, got %s", expectedStatus, videoResp.Status)
        }

        return nil
    })
    return p
}
```

#### 5. Test Data Management
**Organize test data cleanly with templates and fixtures:**

```go
// Template-based test data generation
func (p *parts) sendAssetEvent(eventType, videoID string) *parts {
    eventTime := time.Now().Add(1 * time.Second).Format(time.RFC3339)
    return p.sendMessageFromTemplate("asset_event.json", map[string]any{
        "Type":          eventType,
        "TenantUUID":    "test-tenant-uuid",
        "ExternalUUID":  videoID,
        "Name":          "E2E Test Video",
        "Active":        true,
        "Since":         eventTime,
        "TechnicalTags": "",
    })
}

// Template file: templates/asset_event.json
{
    "Type": "{{.Type}}",
    "TenantUUID": "{{.TenantUUID}}",
    "ExternalUUID": "{{.ExternalUUID}}",
    "Name": "{{.Name}}",
    "Active": {{.Active}},
    "Since": "{{.Since}}",
    "TechnicalTags": "{{.TechnicalTags}}"
}
```

#### 6. Asynchronous Testing Patterns
**Handle eventual consistency with robust retry logic:**

```go
// CI-aware retry configuration
func newChecks() *checks {
    interval := 500 * time.Millisecond
    timeout := 20 * time.Second

    // Use longer intervals for CI environments
    if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
        interval = 2 * time.Second
        timeout = 60 * time.Second
    }

    return &checks{
        interval: interval,
        timeout:  timeout,
        checks:   make([]*check, 0, 10),
    }
}

// Accumulate checks and execute with retry
func (p *parts) videoHasAssets(expectedCount int) *parts {
    p.checks.add(func() error {
        query := `SELECT COUNT(*) FROM crab.assets WHERE video_id = $1`

        var count int
        err := p.db.QueryRow(query, p.videoID).Scan(&count)
        if err != nil {
            return fmt.Errorf("query assets count: %w", err)
        }

        if count != expectedCount {
            return fmt.Errorf("expected %d assets, got %d", expectedCount, count)
        }

        return nil
    })
    return p
}
```

#### 7. Test Organization Best Practices
**Structure tests for maximum maintainability:**

```
test/
└── e2e/
    ├── .env.test              # Test environment configuration
    ├── docker-compose.test.yml
    ├── e2e_test.go            # Main test scenarios (readable workflows)
    ├── e2e_parts_test.go      # Test helpers and fluent interface
    └── e2e_checks_test.go     # Asynchronous check implementation
```

#### 8. Environment Configuration Management
**Separate test configurations cleanly:**

```go
// Separate test configs with environment-specific behavior
func setupTestEnvironment(t *testing.T) *TestEnvironment {
    t.Helper()

    // Load test-specific config
    err := godotenv.Load(".env.test")
    require.NoError(t, err, "load test environment")

    env := &TestEnvironment{
        DBURL:     os.Getenv("TEST_DB_URL"),
        SQSQueue:  os.Getenv("TEST_SQS_QUEUE_URL"),
        ServiceAddr: os.Getenv("TEST_SERVICE_ADDRESS"),
    }

    return env
}
```

## Test Anti-Patterns to Avoid

### ❌ Anti-Pattern: Complex Test Setup
```go
// BAD: Complex setup mixed with test logic
func TestVideoProcessing(t *testing.T) {
    // 50 lines of complex setup here
    db := setupDatabase(t, "complex-config.sql")
    sqsClient := setupSQS(t)
    s3Client := setupS3(t)
    videoProcessor := NewVideoProcessor(db, sqsClient, s3Client)

    // Test logic buried in setup
    // ... actual test logic
}
```

### ✅ Correct Pattern: Clean Separation
```go
// GOOD: Clean setup separation
func TestVideoProcessing(t *testing.T) {
    given, when, then := newParts(t)

    given.
        setupVideoProcessingEnvironment().and().
        uploadTestVideo()

    when.
        startVideoProcessing()

    then.
        videoProcessedSuccessfully().and().
        assetsCreated()
}

func (p *parts) setupVideoProcessingEnvironment() *parts {
    // All complex setup encapsulated in helper method
    p.db = setupDatabase(p.T, "test-config.sql")
    p.sqsClient = setupSQS(p.T)
    p.s3Client = setupS3(p.T)
    p.videoProcessor = NewVideoProcessor(p.db, p.sqsClient, p.s3Client)
    return p
}
```

## Testing Commands Reference

### E2E Test Execution Commands
```bash
# E2E tests (HTTP-based) with docker-compose.test.yml
just test-e2e-up      # Start docker-compose.test.yml services
just test-e2e-run     # Run `go test ./test/e2e/...` against running services
just test-e2e-down    # Stop docker-compose.test.yml services
just test-e2e         # Complete workflow (up → run → down)

# All tests (unit + integration + e2e + browser)
just test-all

# Pre-commit check (unit tests + lint only - fast)
just check

# IMPORTANT: Browser tests are MANDATORY for frontend features
# HTTP-based E2E tests DO NOT validate frontend functionality
```

### Test Environment Variables
- `TEST_DB_URL` - Database connection string for tests
- `TEST_SQS_QUEUE_URL` - SQS queue URL for async tests
- `TEST_SERVICE_ADDRESS` - Service base URL for HTTP tests
- `CI` - Automatically set in CI environments
- `GITHUB_ACTIONS` - Automatically set in GitHub Actions

### 11. Minimal Comments Rule
**Test source carries near-zero comments. The fluent chain IS the documentation.** Delete: `// PRD-COMMENT:` / requirement-tracker blocks (that context belongs in the PR description), `// Given / // When / // Then` section headers (the chain shows it), `// X was captured before...` / `// suppress unused-var` meta-comments, and anything that paraphrases the next line. Keep only: TODO/FIXME with an issue reference, one-line godoc on exported symbols, `// nolint:` directives. If a test needs a paragraph to be understandable, the fix is better names or a DSL method — not a comment.

```go
// ❌ FORBIDDEN: comment-driven test
// PRD-COMMENT: FR10.1 - Client Core Execution: Single Request
// This test verifies that the client can execute one request...
func TestExecuteFile_SingleRequest(t *testing.T) {
    given, when, then := newParts(t)
    // Given
    given.aClient()
    // When
    when.executeFile()
    // Then
    then.noError()
}

// ✅ CORRECT: names + chain speak for themselves
func TestExecuteFile_SingleRequest(t *testing.T) {
    given, when, then := newParts(t)

    given.
        aRequestFixture("single_request.http").and().
        aClient()

    when.
        executeFile()

    then.
        noError()
}
```

### 12. YAGNI Pass Before Every PR (ponytail)
**Before opening a PR, delete everything that has no caller.** Migration/refactor work breeds orphaned helpers — grep and remove: DSL methods with zero call sites, struct types whose table tests were split, per-fixture builder functions, unused fixture files, dead `_ = x` lines. A helper "we might need later" is a liability; re-adding it later costs less than maintaining it now. Prove the pass: grep each deleted symbol returns 0 hits, then run the full gate (build/vet/lint/tests).

**Never appease the `unused` linter with reference lists (PR#35 round 3).** `var _ = []any{ (*parts).foo, ... }` blocks in DSL files keep dead methods lint-clean forever — 5 dead DSL methods survived 3 review rounds behind them. Delete the blocks and let `unused` name the dead methods, then delete them. Same for `var _ = time.Time{}`-style "silence unused import" lines: fix the import, not the symptom.

**DSL domain split when one file bloats (PR#35 round 3).** fluent_parts_test.go core (parts/newParts/and/lifecycle/base assertions); ext (fixtures/templates/external files/env/servers); capture (tracking/captured*); validator; graphql; per-feature files as needed. File moves only — no renames, no behavior changes; one package so all files share `parts`. Reviewer-triggered splits are exactly this shape.

**Migration-history godoc is deleted, not shrunk (PR#35 rounds 2-3).** "The legacy tests never exercised this path... gate-mandated count..." belongs in the PR description / bean PoW, never in code. This includes 1-line shrunk survivors and mid-sentence `//` fragments above Test funcs. Delegate contracts for review rounds must state: comments max 1 line across the ENTIRE diff (DSL files included — workers otherwise read "minimal comments" as scoping only Test funcs), only non-obvious WHY, no `// --- moved from X ---` banners. Census: `awk '/^\/\//{c++;next} {if(c>2) print FILENAME": "c" before "NR; c=0}' *_test.go`.

**Repeated mock-infrastructure in Test bodies = missing DSL given (PR#35 rounds 1-3).** The same handler shape appearing 3+ times (GraphQL JSON encoding, canned routes, echo) becomes an `a*Server` method. Likewise multi-line inline content strings (external-file JSON payloads) become committed fixtures under `test/data/<domain>/` with a `*Fixture` DSL variant — single-line payloads may stay inline. Keep fixture layout canonical: one `test/data/` tree; dissolve stray dirs (test/fixtures/) and grep `*.go *.http *.md` for stale paths in the same batch.

---

**These standards are MANDATORY for all E2E tests. No exceptions.**