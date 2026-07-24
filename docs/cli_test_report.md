# CLI Test Report

## Test Results

```
DONE 231 tests in 9.551s
0 issues
Coverage: 77.1%
```

## Manual Test Commands

### -f/--file (required)

```bash
restclient -f test/fixtures/test_basic.http
```

### -n/--name (run by name)

```bash
restclient -f test/fixtures/test_multi.http -n "get user"
```

### -i/--index (run by index)

```bash
restclient -f test/fixtures/test_multi.http -i 0
```

### -l/--list (list requests)

```bash
restclient -f test/fixtures/test_multi.http --list
```

### -D/--define (command-line variables)

```bash
restclient -f test/fixtures/test_basic.http -D token=abc123 -D env=prod
restclient -f test/fixtures/test_basic.http --define token=abc123 --define env=prod
```

### -A/--after (prerequisite requests)

```bash
restclient -f test/fixtures/test_chaining.http -n "get protected" -A authenticate
```

### -E/--fail-on-error (fail on 4xx/5xx)

```bash
restclient -f test/fixtures/test_basic.http -E
```

### -o/--output (output formats)

```bash
# Body only
restclient -f test/fixtures/test_basic.http -o body

# JSON path extraction
restclient -f test/fixtures/test_basic.http -o jsonpath "headers.X-Custom"

# Environment variable format
restclient -f test/fixtures/test_basic.http -o env "token"
```

### -e/--expected (response validation)

```bash
restclient -f test/fixtures/test_basic.http -e test/fixtures/expected.hresp
```

### --e-name (validate specific response by name)

```bash
restclient -f test/fixtures/test_multi.http -n "get user" -e test/fixtures/responses.hresp --e-name "success"
```

### --e-index (validate specific response by index)

```bash
restclient -f test/fixtures/test_multi.http -n "get user" -e test/fixtures/responses.hresp --e-index 0
```

### -h/--help

```bash
restclient --help
```

## Error Cases

### Missing file

```bash
restclient
# Exit code: 80
# Output: missing flags: --file=STRING
```

### Nonexistent file

```bash
restclient -f nonexistent.http
# Exit code: 1
# Output: error: open nonexistent.http: no such file or directory
```

### Mutually exclusive -n and -i

```bash
restclient -f test_multi.http -n "get user" -i 0
# Exit code: 2
# Output: error: -n and -i are mutually exclusive
```

### Request not found

```bash
restclient -f test_multi.http -n "nonexistent"
# Exit code: 1
# Output: error: request name "nonexistent" not found
```

### Invalid -D format

```bash
restclient -f test_basic.http -D invalid
# Exit code: 1
# Output: error: invalid -D/--define format: "invalid" (expected key=value)
```

### Invalid output format

```bash
restclient -f test_basic.http -o invalid
# Exit code: 1
# Output: error: output format must be one of: body, jsonpath, env
```
