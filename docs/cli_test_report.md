# CLI Test Report

## Test Commands

```bash
# Build
go build -o restclient ./cmd/restclient

# Run all tests
make check

# Run specific test
go test -run TestCLI_DefineFlag ./cmd/restclient/
```

## Test Results

```
DONE 231 tests in 9.551s
0 issues
Coverage: 77.1%
```

## Manual Test Files

### test_basic.http
```http
GET https://echo.free.beeceptor.com/api/users
Authorization: Bearer {{$dotenv TOKEN}}
```

### test_chaining.http
```http
### authenticate
POST https://echo.free.beeceptor.com/auth
Content-Type: application/json

{"user":"admin","pass":"secret"}

### get protected
GET https://echo.free.beeceptor.com/protected
Authorization: Bearer {{authenticate.response.body.token}}
```

### test_prereq.http
```http
### login
POST https://echo.free.beeceptor.com/login
Content-Type: application/json

{"user":"admin"}
```

## CLI Flags

| Flag | Long | Description |
|------|------|-------------|
| `-f` | `--file` | Request file path |
| `-n` | `--name` | Run request by name |
| `-i` | `--index` | Run request by index |
| `-e` | `--expected` | Expected response file |
| `-l` | `--list` | List requests |
| `-E` | `--fail-on-error` | Fail on 4xx/5xx |
| `-o` | `--output` | Output format |
| `-A` | `--after` | Prerequisite request |
| `-D` | `--define` | Define variable |

## Examples

```bash
# List requests
restclient -f test.http --list

# Run with variables
restclient -f test.http -D token=abc123 -D env=prod

# Run specific request
restclient -f test.http -n "get user"

# Run after prerequisite
restclient -f test.http -n "get protected" -A authenticate
```
