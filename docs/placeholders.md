# HTTP Request Placeholders

This document consolidates placeholder variables from JetBrains HTTP Client and VS Code REST Client extension. These placeholders can be used in HTTP request files (`.http` or `.rest`) to dynamically generate values or reference stored data.

## Variable Types

### Custom Variables

Custom variables are defined directly in HTTP files and provide reusable values.

**File Variables**

Define at the top of your HTTP file:
```
@variableName = variableValue
```

- Example: `@baseUrl = https://api.example.com`
- Reference: `{{baseUrl}}`
- File variables can contain references to other variables
- File variables are only accessible within the current file

### Environment Variables

Environment variables enable environment-specific configurations (e.g., development, production).

**Usage**:
- Defined in configuration files or settings
- Referenced with `{{variableName}}`
- Typically stored outside the HTTP file in:
  - `http-client.env.json` (JetBrains)
  - VS Code settings.json (REST Client extension)

### Dynamic Variables (System Variables)

Dynamic variables generate values at runtime and are referenced using the `{{$variableName}}` syntax.

## Supported Placeholders

### UUID/GUID Generation

| Placeholder | Description | Example Output |
|-------------|-------------|----------------|
| `{{$guid}}` | Generates a UUID v4 | `123e4567-e89b-12d3-a456-426614174000` |
| `{{$uuid}}` | Generates a UUID v4 (JetBrains) | `123e4567-e89b-12d3-a456-426614174000` |
| `{{$random.uuid}}` | Generates a UUID v4 (JetBrains) | `123e4567-e89b-12d3-a456-426614174000` |

### Date and Time

| Placeholder | Description | Example Output |
|-------------|-------------|----------------|
| `{{$timestamp}}` | Current Unix timestamp (seconds) | `1685123456` |
| `{{$isoTimestamp}}` | Current timestamp in ISO-8601 format (UTC) | `2025-06-06T10:19:19Z` |
| `{{$datetime format}}` | Current UTC datetime with specified format | `2025-06-06` |
| `{{$localDatetime format}}` | Current local datetime with specified format | `2025-06-06` |

Supported formats for `$datetime` and `$localDatetime`:
- `"rfc1123"`: RFC 1123 format
- `"iso8601"`: ISO 8601 format
- Go layout string (e.g., `"2006-01-02"`)

### Random Values

| Placeholder | Description | Example Output |
|-------------|-------------|----------------|
| `{{$randomInt}}` | Random integer between 0-1000 | `345` |
| `{{$randomInt min max}}` | Random integer between min and max | `42` |
| `{{$random.integer(min, max)}}` | Random integer between min (inclusive) and max (exclusive) | `42` |
| `{{$random.float(min, max)}}` | Random float between min (inclusive) and max (exclusive) | `42.5` |

### Text and String Generation

| Placeholder | Description | Example Output |
|-------------|-------------|----------------|
| `{{$random.alphabetic(length)}}` | Random string of alphabetic characters | `abcDEFghi` |
| `{{$random.alphanumeric(length)}}` | Random string of alphanumeric characters and underscores | `a1b2C3_d` |
| `{{$random.hexadecimal(length)}}` | Random hexadecimal string | `a1b2c3` |
| `{{$random.email}}` | Random email address | `user@example.com` |

### Environment Access

| Placeholder | Description | Example Output |
|-------------|-------------|----------------|
| `{{$processEnv NAME}}` | Value of environment variable NAME | `api_token_123` |
| `{{$processEnv %NAME}}` | Indirect environment lookup (VS Code) | Variable value |
| `{{$env.NAME}}` | Value of environment variable NAME (JetBrains) | `api_token_123` |
| `{{$dotenv NAME}}` | Value of NAME from .env file | `api_token_123` |

### Additional Random Data Types (JetBrains Only)

JetBrains HTTP Client also supports many additional data generation functions from the Java Faker library:

| Placeholder | Description | Example Output |
|-------------|-------------|----------------|
| `{{$random.address}}` | Random address | `123 Main St` |
| `{{$random.name}}` | Random name | `John Smith` |
| `{{$random.company}}` | Random company name | `Acme Corp` |
| `{{$random.phoneNumber}}` | Random phone number | `555-123-4567` |
| `{{$random.internet}}` | Random internet-related value | Various |
| `{{$random.finance}}` | Random finance-related value | Various |

And many more. Full list includes: address, beer, bool, business, ChuckNorris.fact, code, color, commerce, company, crypto, educator, finance, hacker, idNumber, internet, lorem, name, number, phoneNumber, shakespeare, superhero, team, university.

### Response References (Chained Requests)

For chained requests, you can reference values from previous requests in the same file:

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{requestName.response.body.field}}` | Access a field from a previous request's response body | `{{login.response.body.token}}` |
| `{{requestName.response.headers.header}}` | Access a header from a previous request's response | `{{login.response.headers.Authorization}}` |

## Examples

### Basic Variable Usage

```http
@baseUrl = https://api.example.com
@version = v1

### Get user profile
GET {{baseUrl}}/{{version}}/users/123
Authorization: Bearer {{token}}
```

### System Variables Example

```http
### Create a new record with random ID
POST https://api.example.com/records
Content-Type: application/json

{
  "id": "{{$guid}}",
  "name": "Test Record",
  "timestamp": {{$timestamp}},
  "randomValue": {{$randomInt}}
}
```

### Environment Variables Example

```http
### Get data using environment credentials
GET https://{{host}}/api/data
Authorization: Bearer {{$env.API_TOKEN}}
```

### Chained Request Example

```http
### Login to get token (named request)
# @name login
POST https://api.example.com/login
Content-Type: application/json

{
  "username": "user",
  "password": "pass"
}

### Use token from previous request
GET https://api.example.com/secure-data
Authorization: Bearer {{login.response.body.token}}
```
