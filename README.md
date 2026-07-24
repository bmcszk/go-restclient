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

### Response Chaining

Reference responses from other requests:

```go
parsedFile, _ := client.ParseFile("requests.http")
// Execute first request
resp1, _ := client.ExecuteRequest(ctx, parsedFile, 0)
// {{authenticate.response.body.token}} is now available for second request
resp2, _ := client.ExecuteRequest(ctx, parsedFile, 1)
```

### Response Validation

Validate responses against `.hresp` files:

```go
responses, _ := client.ExecuteFile(ctx, "requests.http")
err := client.ValidateResponses("expected.hresp", responses...)
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
restclient -f requests.http --define token=abc123 --define env=prod
```

### Prerequisite requests

Run a request before the target (for auth token chaining):
```bash
restclient -f requests.http -n "get protected" -A authenticate
restclient -f requests.http -n "get protected" --after authenticate
```

### Fail on errors

Exit with code 1 on HTTP 4xx/5xx responses:
```bash
restclient -f requests.http -E
restclient -f requests.http --fail-on-error
```

### Output formats

```bash
# Body only
restclient -f requests.http -o body

# JSON path extraction
restclient -f requests.http -o jsonpath "data.users[0].name"

# Environment variable format
restclient -f requests.http -o env "token"
# Output: token=eyJhbGciOiJIUzI1NiIs...
```

### Assert responses

Validate against an expected-response file (`.hresp` format):
```bash
restclient -f requests.http -e expected.hresp
restclient -f requests.http -n "login" -e login_expected.hresp

# Validate specific response by name
restclient -f requests.http -n "login" -e responses.hresp --e-name "success"

# Validate specific response by index
restclient -f requests.http -n "login" -e responses.hresp --e-index 0
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

`-n` and `-i` are mutually exclusive.

Exits with code `1` if any request fails or assertion fails, `0` on success, `2` on usage error.

## Quick Start

### 1. Create a `.http` file

```http
### get user
GET https://httpbin.org/get
Authorization: Bearer {{$dotenv TOKEN}}

### create user
POST https://httpbin.org/post
Content-Type: application/json

{
  "name": "John",
  "email": "john@example.com"
}
```

### 2. Set environment variables

```bash
export TOKEN=my-secret-token
```

### 3. Run with CLI

```bash
restclient -f requests.http
```

### 4. Or use as library

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

## Testing

### Run all tests

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
