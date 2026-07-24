# Go REST Client Library

A Go library for executing HTTP requests from `.http` files and validating responses. Write once, use everywhere - for both manual testing and automated E2E tests.

## Why This Library?

**Problem**: I wanted to use the same `.http` files for both manual testing (JetBrains HTTP Client, VS Code REST Client) and automated E2E testing in Go. No existing library offered full compatibility with both environments.

**Solution**: This library parses `.http` files exactly like popular IDE extensions, enabling seamless workflow between manual and automated testing.

## Key Features

- **Full JetBrains/VS Code compatibility** - Same `.http` syntax, variables, and behaviors
- **Variable substitution** - Custom variables, environment variables, system variables (`{{$guid}}`, `{{$randomInt}}`, etc.)
- **Response chaining** - Reference responses from other requests: `{{name.response.body.field}}`
- **Command-line variables** - Override values with `-D key=value`
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

## HTTP File Syntax

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
### Create User
POST https://api.example.com/users
Content-Type: application/json
Authorization: Bearer {{$dotenv TOKEN}}

{
  "name": "{{username}}",
  "email": "{{email}}"
}
```

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

### System Variables

| Variable | Description |
|----------|-------------|
| `{{$guid}}` | UUID v4 |
| `{{$randomInt}}` | Random integer |
| `{{$timestamp}}` | Current Unix timestamp |
| `{{$dotenv KEY}}` | Environment variable from .env file |
| `{{$base64encode value}}` | Base64 encode a value |

### Multiple Requests

Separate requests with `###`:
```http
### Get Users
GET https://api.example.com/users

### Get User
GET https://api.example.com/users/1

### Create User
POST https://api.example.com/users
Content-Type: application/json

{"name":"John"}
```

## Testing

```bash
# Run all tests
make check

# Run specific test
go test -run TestCLI_DefineFlag ./cmd/restclient/
```

## License

MIT
