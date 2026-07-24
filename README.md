# Go REST Client Library

A Go library for executing HTTP requests from `.http` files and validating responses. Write once, use everywhere - for both manual testing and automated E2E tests.

## Why This Library?

**Problem**: I wanted to use the same `.http` files for both manual testing (JetBrains HTTP Client, VS Code REST Client) and automated E2E testing in Go. No existing library offered full compatibility with both environments.

**Solution**: This library parses `.http` files exactly like popular IDE extensions, enabling seamless workflow between manual and automated testing.

## Key Features

- **Full JetBrains/VS Code compatibility** - Same `.http` syntax, variables, and behaviors
- **Variable substitution** - Custom variables, environment variables, system variables (`{{$guid}}`, `{{$randomInt}}`, etc.)
- **Response chaining** - Reference responses from other requests: `{{name.response.body.field}}`
- **Response validation** - Compare responses against `.hresp` files with placeholders (`{{$any}}`, `{{$regexp}}`, `{{$anyGuid}}`)
- **Multiple requests per file** - Separated by `###`
- **E2E testing ready** - Perfect for automated integration tests

## HTTP Files

`.http` files are plain text files for defining HTTP requests. They were popularized by JetBrains IDEs (IntelliJ, PyCharm, GoLand) and are now supported by VS Code (REST Client extension) and other tools.

### Request Format

```http
### Request Name
METHOD URL
Header1: value1
Header2: value2

body content
```

## Variable Types

### Custom Variables

```http
@baseUrl = https://api.example.com
@userId = 123

### Get User
GET {{baseUrl}}/users/{{userId}}
Authorization: Bearer {{$dotenv TOKEN}}
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

### Response Chaining

Reference responses from other requests in the same file:

```http
### Authenticate
POST https://api.example.com/auth
Content-Type: application/json

{"user":"admin","pass":"secret"}

### Get Protected
GET https://api.example.com/protected
Authorization: Bearer {{authenticate.response.body.token}}
```

## Library

### Lib installation

```bash
go get github.com/bmcszk/go-restclient
```

### Execute in Go

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

### Programmatic Variables (highest precedence)

```go
client, err := restclient.NewClient(
    restclient.WithVars(map[string]interface{}{
        "userId": "override-value",
        "authToken": "secret-token",
    }),
)
```

## CLI

TODO

### CLI installation

Install the `restclient` CLI to run `.http` files from command line:

```bash
go install github.com/bmcszk/go-restclient/cmd/restclient@latest
```

### CLI usage

```bash
restclient -f requests.http
```

### List requests

```bash
restclient -f requests.http --list
```

### Run a single request

By name (case-insensitive):

```bash
restclient -f requests.http -n "create user"
```

By 0-based index:

```bash
restclient -f requests.http -i 0
```

### Command-line variables

Override variables from the command line:

```bash
restclient -f requests.http -D token=abc123 -D env=prod
```

### Prerequisite requests

Run a request before the target (for auth token chaining):

```bash
restclient -f requests.http -n "get protected" -A authenticate
```

### Fail on errors

Exit with code 1 on HTTP 4xx/5xx responses:

```bash
restclient -f requests.http -E
```

### Output formats

```bash
# Body only
restclient -f requests.http -o body

# JSON path extraction
restclient -f requests.http -o jsonpath "data.users[0].name"

# Environment variable format
restclient -f requests.http -o env "token"
```

### CLI Flags

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

## Development

### Prerequisites
- Go 1.21+

### Commands
```bash
make check          # Run all checks (lint, test, build)
```

## License

MIT License
