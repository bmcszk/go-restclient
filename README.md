# Go REST Client Library

A Go library for executing HTTP requests from `.http` files and validating responses. Write once, use everywhere - for both manual testing and automated E2E tests.

## Why This Library?

**Problem**: I wanted to use the same `.http` files for both manual testing (JetBrains HTTP Client, VS Code REST Client) and automated E2E testing in Go. No existing library offered full compatibility with both environments.

**Solution**: This library parses `.http` files exactly like popular IDE extensions, enabling seamless workflow between manual and automated testing.

## Key Features

- **Full JetBrains/VS Code compatibility** - Same `.http` syntax, variables, and behaviors
- **Variable substitution** - Custom variables, environment variables, system variables (`{{$guid}}`, `{{$randomInt}}`, etc.)
- **Response validation** - Compare responses against `.hresp` files with placeholders (`{{$any}}`, `{{$regexp}}`, `{{$anyGuid}}`)
- **Multiple requests per file** - Separated by `###`
- **E2E testing ready** - Perfect for automated integration tests

## Installation

```bash
go get github.com/bmcszk/go-restclient
```

### CLI

Install the `restclient` CLI (runs a `.http` file and prints each response):

```bash
go install github.com/bmcszk/go-restclient/cmd/restclient@latest
```

```bash
restclient -f requests.http
```

#### List requests

```bash
restclient -f requests.http --list
```

#### Run a single request

By name (case-insensitive):
```bash
restclient -f requests.http -n "create user"
```

By 0-based index:
```bash
restclient -f requests.http -i 0
```

#### Command-line variables

Override variables from the command line:
```bash
restclient -f requests.http -D token=abc123 -D env=prod
restclient -f requests.http --define token=abc123 --define env=prod
```

#### Prerequisite requests

Run a request before the target (for auth token chaining):
```bash
restclient -f requests.http -n "get protected" -A authenticate
restclient -f requests.http -n "get protected" --after authenticate
```

#### Fail on errors

Exit with code 1 on HTTP 4xx/5xx responses:
```bash
restclient -f requests.http -E
restclient -f requests.http --fail-on-error
```

#### Output formats

```bash
# Body only
restclient -f requests.http -o body

# JSON path extraction
restclient -f requests.http -o jsonpath "data.users[0].name"

# Environment variable format
restclient -f requests.http -o env "token"
# Output: token=eyJhbGciOiJIUzI1NiIs...
```

#### Assert responses

Validate against an expected-response file (`.hresp` format):
```bash
restclient -f requests.http -e expected.hresp
restclient -f requests.http -n "login" -e login_expected.hresp

# Validate specific response by name
restclient -f requests.http -n "login" -e responses.hresp --e-name "success"

# Validate specific response by index
restclient -f requests.http -n "login" -e responses.hresp --e-index 0
```

#### CLI Flags

| Short | Long | Description |
|-------|------|-------------|
| `-f` | `--file` | Request file path (required) |
| `-n` | `--name` | Run request by name |
| `-i` | `--index` | Run request by index |
| `-e` | `--expected` | Expected response file |
| | `--e-name` | Expected response name |
| | `--e-index` | Expected response index |
| `-l` | `--list` | List requests |
| `-E` | `--fail-on-error` | Fail on 4xx/5xx |
| `-o` | `--output` | Output format |
| `-A` | `--after` | Prerequisite request |
| `-D` | `--define` | Define variable (repeatable) |

`-n` and `-i` are mutually exclusive.

Exits with code `1` if any request fails or assertion fails, `0` on success, `2` on usage error.

## Quick Start

### 1. Create a `.http` file

```http
@baseUrl = https://api.example.com
@userId = 123

### Get user profile
GET {{baseUrl}}/users/{{userId}}
Authorization: Bearer {{authToken}}
X-Request-ID: {{$guid}}

### Create new user  
POST {{baseUrl}}/users
Content-Type: application/json

{
  "id": "{{$randomInt 1000 9999}}",
  "name": "Test User",
  "createdAt": "{{$timestamp}}"
}
```

### 2. Execute in Go

```go
package main

import (
    "context"
    "log"
    "github.com/bmcszk/go-restclient"
)

func main() {
    client, err := restclient.NewClient(
        restclient.WithVars(map[string]interface{}{
            "authToken": "your-token-here",
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    responses, err := client.ExecuteFile(context.Background(), "requests.http")
    if err != nil {
        log.Fatal(err)
    }

    for i, resp := range responses {
        if resp.Error != nil {
            log.Printf("Request %d failed: %v", i+1, resp.Error)
        } else {
            log.Printf("Request %d: %d %s", i+1, resp.StatusCode, resp.Status)
        }
    }
}
```

## Variable Types

### Custom Variables
```http
@baseUrl = https://api.example.com
@userId = 123

GET {{baseUrl}}/users/{{userId}}
```

### System Variables
- `{{$guid}}` - UUID (e.g., `123e4567-e89b-12d3-a456-426614174000`)
- `{{$randomInt}}` or `{{$randomInt 1 100}}` - Random integer
- `{{$timestamp}}` - Unix timestamp
- `{{$datetime}}` or `{{$datetime "2006-01-02"}}` - Current datetime
- `{{$processEnv VAR_NAME}}` - Environment variable
- `{{$dotenv VAR_NAME}}` - From `.env` file

### JetBrains Faker Variables
- `{{$randomFirstName}}`, `{{$randomLastName}}`
- `{{$randomPhoneNumber}}`, `{{$randomStreetAddress}}`
- `{{$randomUrl}}`, `{{$randomUserAgent}}`

### Programmatic Variables (highest precedence)
```go
client, err := restclient.NewClient(
    restclient.WithVars(map[string]interface{}{
        "userId": "override-value",
        "authToken": "secret-token",
    }),
)
```

## Response Validation

Create `.hresp` files to validate responses:

**responses.hresp:**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "{{$anyGuid}}",
  "name": "{{$any}}",
  "createdAt": "{{$anyTimestamp}}"
}

###

HTTP/1.1 201 Created
Content-Type: application/json

{
  "id": "{{$regexp `\d{4}`}}",
  "status": "created"
}
```

**Validate in Go:**
```go
err := client.ValidateResponses("responses.hresp", responses...)
if err != nil {
    log.Fatal("Validation failed:", err)
}
```

### Validation Placeholders
- `{{$any}}` - Matches any text
- `{{$regexp `pattern`}}` - Regex pattern (in backticks)
- `{{$anyGuid}}` - UUID format
- `{{$anyTimestamp}}` - Unix timestamp
- `{{$anyDatetime 'format'}}` - Datetime (rfc1123, iso8601, or custom)

## Client Options

```go
client, err := restclient.NewClient(
    restclient.WithBaseURL("https://api.example.com"),
    restclient.WithDefaultHeader("X-API-Key", "secret"),
    restclient.WithHTTPClient(customHTTPClient),
    restclient.WithVars(variables),
)
```

## Compatible Syntax

Works with files created for:
- [JetBrains HTTP Client](https://www.jetbrains.com/help/idea/http-client-in-product-code-editor.html)
- [VS Code REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client)

📚 **[Complete HTTP Syntax Reference](docs/http_syntax.md)** - Comprehensive documentation of all supported HTTP request syntax, variables, and features.

## Use Cases

### Manual Testing
Use your favorite IDE extension to test APIs during development.

### Automated E2E Testing
```go
func TestUserAPI(t *testing.T) {
    client, _ := restclient.NewClient(
        restclient.WithBaseURL(testServer.URL),
    )
    
    responses, err := client.ExecuteFile(context.Background(), "user_tests.http")
    require.NoError(t, err)
    
    err = client.ValidateResponses("user_expected.hresp", responses...)
    require.NoError(t, err)
}
```

### CI/CD Integration
```bash
go test ./tests/e2e/... # Runs tests using .http files
```

## Development

### Prerequisites
- Go 1.21+

### Commands
```bash
make check          # Run all checks (lint, test, build)
make test-unit      # Run unit tests only
go test .           # Quick test
```

### Test Coverage
Current coverage: 78.4% (187 tests passing)

## License

MIT License