# HTTP Request Syntax Documentation

This document consolidates the HTTP request syntax from both JetBrains HTTP Client and VS Code REST Client extension. It provides a comprehensive reference for writing HTTP request files.

## Feature Comparison

| Feature | JetBrains HTTP Client | VS Code REST Client | go-restclient Support |
|---------|----------------------|-------------------|----------------------|
| Basic HTTP Methods | ✅ | ✅ | ✅ |
| Multiple Requests per File | ✅ | ✅ | ✅ |
| Request Names | ✅ | ✅ | ✅ |
| Environment Variables | ✅ | ✅ | ✅ |
| File Variables | ✅ | ✅ | ✅ |
| System Variables | ✅ | ✅ | ✅ |
| Response Handling | ✅ | ✅ | ✅ |
| Response References | ✅ | ✅ | ✅ |
| GraphQL Support | ✅ | ✅ | ✅ |
| File Upload | ✅ | ✅ | ✅ |
| Cookie Management | ✅ | ✅ | ✅ |
| Response Validation | ✅ | ✅ | ✅ |
| Pre-request Scripts | ✅ | ✅ | ❌ |
| Post-response Scripts | ✅ | ✅ | ❌ |
| cURL Import/Export | ✅ | ✅ | ❌ |
| Authentication Helpers | ✅ | ✅ | ✅ |
| Request History | ✅ | ✅ | ❌ |

## Placeholders Reference

This section provides a comprehensive reference for all placeholders supported by both JetBrains HTTP Client and VS Code REST Client, as well as those unique to each client.

### Common Placeholders

| Placeholder | Description | Example | Supported By |
|-------------|-------------|---------|-------------|
| `{{$uuid}}` / `{{$guid}}` | Generates UUID v4 | `123e4567-e89b-12d3-a456-426614174000` | Both |
| `{{$timestamp}}` | Current Unix timestamp (seconds) | `1654321098` | Both |
| `{{$datetime format}}` | UTC datetime with specified format | `2025-06-06T11:06:52Z` | Both |
| `{{$localDatetime format}}` | Local datetime with specified format | `2025-06-06 13:06:52` | Both |
| `{{$randomInt}}` | Random integer (0-1000 by default) | `123` | Both |

### Environment Access Placeholders

| Placeholder | Description | Example | Supported By |
|-------------|-------------|---------|-------------|
| `{{$env.VAR_NAME}}` | System environment variable | `api-key-123` | JetBrains |
| `{{$processEnv VAR_NAME}}` | System environment variable | `api-key-123` | VS Code |
| `{{$processEnv %VAR_NAME}}` | Indirect environment lookup | Value from another env var | VS Code |
| `{{$dotenv VAR_NAME}}` | Value from .env file | `secret-123` | VS Code |

### JetBrains-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|--------|
| `{{$isoTimestamp}}` | ISO-8601 format timestamp (UTC) | `2025-06-06T11:06:52Z` |
| `{{$random.integer(min, max)}}` | Random integer in range | `42` |
| `{{$random.float(min, max)}}` | Random float in range | `42.5` |
| `{{$random.alphabetic(length)}}` | Random alphabetic string | `ABcDef` |
| `{{$random.alphanumeric(length)}}` | Random alphanumeric string | `A1b2C3` |
| `{{$random.hexadecimal(length)}}` | Random hexadecimal string | `1a2b3c` |
| `{{$random.email}}` | Random email | `user@example.com` |

#### JetBrains Faker Library Variables

JetBrains HTTP Client supports the following classes from the Java Faker library:

```
$random.address          $random.educator         $random.number
$random.beer             $random.finance          $random.phoneNumber
$random.bool             $random.hacker           $random.shakespeare
$random.business         $random.idNumber         $random.superhero
$random.ChuckNorris.fact $random.internet         $random.team
$random.code             $random.lorem            $random.university
$random.color            $random.name
$random.commerce         $random.company
$random.crypto
```

Examples:
```
{{$random.name.firstName}} - Random first name
{{$random.address.city}} - Random city name
{{$random.finance.creditCard}} - Random credit card number
```

### VS Code-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|--------|
| `{{$aadToken [options]}}` | Azure Active Directory token | OAuth token |
| `{{$aadV2Token [options]}}` | Azure AD v2 token | OAuth token |

#### Azure AD Token Options (VS Code)

`{{$aadToken [new] [public|cn|de|us|ppe] [<domain|tenantId>] [aud:<domain|tenantId>]}}`

- `new`: Optional. Force re-authentication.
- `public|cn|de|us|ppe`: Optional. Specify top-level domain.
- `<domain|tenantId>`: Optional. Domain or tenant ID.
- `aud:<domain|tenantId>`: Optional. Target Azure AD audience.

`{{$aadV2Token [new] [appOnly] [scopes:<scope[,]>] [tenantid:<domain|tenantId>] [clientid:<clientId>]}}`

- `new`: Optional. Force re-authentication.
- `appOnly`: Optional. Use client credentials flow.
- `scopes:<scope[,]>`: Optional. Comma-delimited scopes.
- `tenantId:<domain|tenantId>`: Optional. Domain or tenant ID.
- `clientId:<clientId>`: Optional. Application registration ID.

## Dynamic Variables Comparison

| Variable | JetBrains HTTP Client | VS Code REST Client | go-restclient Support |
|----------|----------------------|-------------------|----------------------|
| UUID/GUID | `{{$uuid}}`, `{{$random.uuid}}` | `{{$guid}}` | ✅ |
| Timestamp | `{{$timestamp}}` | `{{$timestamp}}` | ✅ |
| ISO Timestamp | `{{$isoTimestamp}}` | N/A | ✅ |
| Random Integer | `{{$randomInt}}`, `{{$random.integer()}}` | `{{$randomInt}}` | ✅ |
| Random Float | `{{$random.float()}}` | N/A | ✅ |
| Random String | `{{$random.alphabetic()}}`, `{{$random.alphanumeric()}}` | N/A | ✅ |
| Random Email | `{{$random.email}}` | N/A | ✅ |
| Date/Time Formatting | `{{$datetime}}` | `{{$datetime}}` | ✅ |
| Local Date/Time | `{{$localDatetime}}` | `{{$localDatetime}}` | ✅ |
| Environment Variables | `{{$env.VAR_NAME}}` | `{{$processEnv VAR_NAME}}` | ✅ |
| .env File Variables | N/A | `{{$dotenv VAR_NAME}}` | ✅ |
| Faker Library Variables | `{{$random.address}}`, `{{$random.name}}`, etc. | N/A | ⚠️ Partial |
| Azure AD Token | N/A | `{{$aadToken}}` | ❌ |

*Note: ✅ = Supported, ❌ = Not supported, ⚠️ = Partially supported*

## Request Structure Basics

### Request Line

A minimal HTTP request consists of a method and URL:

```
GET https://example.com/api/users
```

#### HTTP Methods

All standard HTTP methods are supported:
- `GET`
- `POST`
- `PUT`
- `DELETE`
- `PATCH`
- `HEAD`
- `OPTIONS`

#### Short Form for GET Requests

For GET requests, you can use a short form that omits the method:

```
https://example.com/api/users
```

### Request Headers

Headers follow the request line with `Name: Value` format:

```
GET https://example.com/api/users
Accept: application/json
Authorization: Bearer token123
```

### Request Body

For methods that support bodies (POST, PUT, PATCH), add an empty line after headers before specifying the body:

```
POST https://example.com/api/users
Content-Type: application/json

{
  "name": "User",
  "email": "user@example.com"
}
```

### HTTP Version

Optionally specify the HTTP version after the URL:

```
GET https://example.com/api/users HTTP/1.1
```

## Multiple Requests in a Single File

Use triple hash marks (`###`) to separate multiple requests in the same file:

```
GET https://example.com/api/users

###

POST https://example.com/api/users
Content-Type: application/json

{
  "name": "User",
  "email": "user@example.com"
}
```

## Request Names

You can name requests for later reference:

```
### Get Users
GET https://example.com/api/users

### Create User
POST https://example.com/api/users
```

For chained requests, use the special syntax to reference previous responses:

```
### Login Request
# @name login
POST https://example.com/api/login
Content-Type: application/json

{
  "username": "test",
  "password": "password"
}

### Use token from login
GET https://example.com/api/secured-resource
Authorization: Bearer {{login.response.body.token}}
```

## Comments

Use `#` for single-line comments:

```
# This is a comment
GET https://example.com/api/users
```

## Variables

### File Variables

Define variables at the top of the file:

```
@baseUrl = https://example.com
@apiVersion = v1
@token = abc123

### Get users
GET {{baseUrl}}/{{apiVersion}}/users
Authorization: Bearer {{token}}
```

### Environment Variables

Environment variables are defined in a JSON configuration file named `http-client.env.json` placed in the same directory as your HTTP request files. This approach consolidates both the JetBrains and VS Code implementations into a single standard.

#### Environment File Structure

```json
{
  "$shared": {
    "commonVar": "value-for-all-environments"
  },
  "development": {
    "host": "dev.example.com",
    "version": "v1",
    "token": "dev-token"
  },
  "production": {
    "host": "api.example.com",
    "version": "v2",
    "token": "prod-token"
  }
}
```

The `$shared` section contains variables accessible across all environments. Other sections define named environments that can be selected when running requests.

#### Using Environment Variables

```
GET {{host}}/api/{{version}}/users
Authorization: Bearer {{token}}
```

#### Environment-specific Files

For environment-specific variables (especially for different environments like dev, test, prod), you can use:

```
http-client.env.json       # For shared/default environments
http-client.private.env.json  # For sensitive data (should be git-ignored)
```

#### Accessing Shared Variables

You can reference shared variables within environment definitions:

```json
{
  "$shared": {
    "apiVersion": "v2",
    "defaultToken": "base-token"
  },
  "development": {
    "host": "dev.example.com",
    "version": "{{$shared apiVersion}}",
    "token": "{{$shared defaultToken}}-dev"
  }
}
```

### Dynamic System Variables

These generate values at runtime using the `{{$variableName}}` syntax:

#### UUID/GUID Generation
- `{{$guid}}` or `{{$uuid}}` or `{{$random.uuid}}`: Generates a UUID v4

#### Date and Time
- `{{$timestamp}}`: Current Unix timestamp (seconds)
- `{{$isoTimestamp}}`: ISO-8601 formatted timestamp (UTC)
- `{{$datetime format}}`: UTC datetime with format
- `{{$localDatetime format}}`: Local datetime with format

Format options:
- `rfc1123`: RFC 1123 format
- `iso8601`: ISO 8601 format
- Custom Go layout string (e.g., `"2006-01-02"`)

#### Random Values
- `{{$randomInt}}`: Random integer (0-1000)
- `{{$randomInt min max}}`: Random integer in range
- `{{$random.integer(min, max)}}`: Random integer in range (JetBrains)
- `{{$random.float(min, max)}}`: Random float in range (JetBrains)
- `{{$random.alphabetic(length)}}`: Random alphabetic string
- `{{$random.alphanumeric(length)}}`: Random alphanumeric string
- `{{$random.hexadecimal(length)}}`: Random hexadecimal string

#### Environment Access
- `{{$processEnv NAME}}`: OS environment variable
- `{{$env.NAME}}`: OS environment variable (JetBrains)
- `{{$dotenv NAME}}`: Value from .env file

### Response References
- `{{requestName.response.body.field}}`: Access a field from a previous response
- `{{requestName.response.headers.header}}`: Access a header from a previous response

## Request Body Types

### JSON

```
POST https://example.com/api/users
Content-Type: application/json

{
  "name": "User",
  "email": "user@example.com"
}
```

### File as Request Body

To read the request body from a file, type the `<` symbol followed by the path to the file:

```
POST https://example.com/api/users
Content-Type: application/json

< ./path/to/payload.json
```

This works for any content type (JSON, XML, binary data, etc.). The file content is read as-is and sent as the request body.

### Form Data

```
POST https://example.com/api/users
Content-Type: application/x-www-form-urlencoded

name=User&email=user@example.com
```

### Multipart Form Data

```
POST https://example.com/api/users
Content-Type: multipart/form-data; boundary=WebAppBoundary

--WebAppBoundary
Content-Disposition: form-data; name="name"

User
--WebAppBoundary
Content-Disposition: form-data; name="email"

user@example.com
--WebAppBoundary--
```

### File Upload

```
POST https://example.com/api/upload
Content-Type: multipart/form-data; boundary=WebAppBoundary

--WebAppBoundary
Content-Disposition: form-data; name="file"; filename="image.jpg"
Content-Type: image/jpeg

< ./path/to/local/image.jpg
--WebAppBoundary--
```

## HTTP Authentication

### Basic Authentication

```
GET https://example.com/api/secure
Authorization: Basic dXNlcjpwYXNzd29yZA==
```

Or using variables:

```
@username = user
@password = password

GET https://{{username}}:{{password}}@example.com/api/secure
```

### Bearer Token

```
GET https://example.com/api/secure
Authorization: Bearer token123
```

## Request Settings

### Request-Specific Options

Request settings can be specified using special comment lines with the `@` prefix. These settings can be placed either before or after the request line, as long as they're part of the same request block:

```
# @name getUsersList
# @no-redirect
# @no-cookie-jar
# @no-log
GET https://example.com/api/users
```

Common request settings include:

| Setting | Description |
|---------|-------------|
| `@name requestName` | Names the request for reference in chained requests |
| `@no-redirect` | Prevents following HTTP redirects |
| `@no-cookie-jar` | Prevents storing/sending cookies for this request |
| `@no-log` | Excludes this request from history logs |
| `@timeout 5000` | Sets request timeout in milliseconds |

### Request Timeouts

```
# @timeout 5000
GET https://example.com/api/slow-resource
```

## Response Handling

### Expected Response (for Testing)

Define expected responses after the actual request:

```
GET https://example.com/api/users

HTTP/1.1 200 OK
Content-Type: application/json

{
  "users": []
}
```

### Response References

Access data from previous responses for chained requests:

```
### Get authentication token
# @name getToken
POST https://example.com/api/login
Content-Type: application/json

{
  "username": "test",
  "password": "password"
}

### Use token for authenticated request
GET https://example.com/api/secure
Authorization: Bearer {{getToken.response.body.token}}
```

## Response Body Validation Placeholders

For expected response validation (applicable in `.hresp` files):

- `{{$any}}`: Matches any sequence of characters
- `{{$regexp 'pattern'}}`: Matches text against a regular expression
- `{{$anyGuid}}`: Matches a UUID string
- `{{$anyTimestamp}}`: Matches a Unix timestamp
- `{{$anyDatetime 'format'}}`: Matches datetime with specified format

## Additional Features

### cURL Import/Export

Both clients support importing from and exporting to cURL format.

### GraphQL Support

GraphQL requests are supported with special syntax:

```
POST https://example.com/graphql
Content-Type: application/json

{
  "query": "query { users { id name } }"
}
```

### Cookies Management

Both clients automatically manage cookies between requests in the same file.

### Redirects

By default, both clients follow redirects. This can be disabled:

```
# @no-redirect
GET https://example.com/redirect
```

## Examples

### Complete Basic Example

```http
@baseUrl = https://api.example.com
@apiVersion = v1

### Get all users
# @name getUsers
GET {{baseUrl}}/{{apiVersion}}/users
Accept: application/json

### Get specific user
GET {{baseUrl}}/{{apiVersion}}/users/{{$randomInt 1 100}}
Accept: application/json

### Create new user
POST {{baseUrl}}/{{apiVersion}}/users
Content-Type: application/json
X-Request-ID: {{$guid}}

{
  "name": "John Doe",
  "email": "john.doe{{$randomInt}}@example.com",
  "created_at": "{{$isoTimestamp}}"
}

### Update specific user
PUT {{baseUrl}}/{{apiVersion}}/users/{{getUsers.response.body.users[0].id}}
Content-Type: application/json

{
  "name": "Updated Name",
  "email": "updated.email@example.com"
}
```

### Authentication Examples

```http
### Basic Auth
GET https://httpbin.org/basic-auth/user/pass
Authorization: Basic dXNlcjpwYXNz

### Bearer Token Auth
GET https://api.example.com/secure
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

### OAuth Authentication Flow Example
# @name getToken
POST https://oauth.provider/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&client_id={{clientId}}&client_secret={{clientSecret}}

###
# Using token from previous request
GET https://api.example.com/secure
Authorization: Bearer {{getToken.response.body.access_token}}
```
