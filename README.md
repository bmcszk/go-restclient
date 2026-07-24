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

### Variables

```http
@baseUrl = https://api.example.com
@userId = 123

### Get User
GET {{baseUrl}}/users/{{userId}}
Authorization: Bearer {{$dotenv TOKEN}}
```

### System Variables

| Variable | Description |
|----------|-------------|
| `{{$guid}}` | UUID v4 |
| `{{$randomInt}}` | Random integer |
| `{{$timestamp}}` | Unix timestamp |
| `{{$dotenv KEY}}` | Environment variable from .env file |
| `{{$base64encode value}}` | Base64 encode a value |

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

Use `go-restclient` as a Go library in your code:

```bash
go get github.com/bmcszk/go-restclient
```

```go
package main

import (
    "context"
    "fmt"
    "github.com/bmcszk/go-restclient"
)

func main() {
    client, _ := restclient.NewClient()
    responses, err := client.ExecuteFile(context.Background(), "requests.http")
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    for _, r := range responses {
        fmt.Println(r.StatusCode, r.Body)
    }
}
```

## CLI

Install the `restclient` CLI to run `.http` files from command line:

```bash
go install github.com/bmcszk/go-restclient/cmd/restclient@latest
```

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

Validate responses against `.hresp` files:

```bash
restclient -f requests.http -e expected.hresp
restclient -f requests.http -n "login" -e login_expected.hresp
```

### .hresp File Format

```http
### Request Name
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "{{$any}}",
  "id": "{{$anyGuid}}"
}
```

### Placeholders

| Placeholder | Description |
|-------------|-------------|
| `{{$any}}` | Match any value |
| `{{$anyGuid}}` | Match any UUID |
| `{{$regexp pattern}}` | Match regular expression |

## Development

### Run tests

```bash
make check
```

### Run specific test

```bash
go test -run TestCLI_DefineFlag ./cmd/restclient/
```

### Test report

See [docs/cli_test_report.md](docs/cli_test_report.md) for comprehensive CLI flag testing.

## License

MIT
