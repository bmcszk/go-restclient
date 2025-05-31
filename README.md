# Go REST Client Library (`go-restclient`)

Last Updated: 2025-05-29

## Overview

`go-restclient` is a Go library designed to simplify HTTP request execution and validation. It allows developers to define HTTP requests in simple text files (`.rest` or `.http` format, similar to RFC 2616 and popular REST client tools), send these requests, and validate the responses against expected outcomes.

This library is suitable for programmatic use within Go applications, particularly for integration and end-to-end testing.

## Features

- **Parse `.rest`/`.http` files:**
    - Supports multiple requests per file, separated by `###`.
    - Handles request method, URL, HTTP version, headers, and body.
    - Supports comments (`#`) and named requests (e.g., `### My Request Name`).
- **Dynamic Variable Substitution:**
    - **Custom Variables (File-defined):** Define and use variables within request files (e.g., `@baseUrl = https://api.example.com`, `{{baseUrl}}`).
    - **System Variables:**
        - `{{$guid}}`: Generates a new UUID.
        - `{{$randomInt [min max]}}`: Generates a random integer. Optional `min` `max` (e.g., `{{$randomInt 1 10}}`). Defaults to 0-100.
        - `{{$timestamp}}`: Current UTC Unix timestamp (seconds).
        - `{{$datetime format}}`: Current UTC datetime. `format` can be `rfc1123`, `iso8601`, or a Go layout string (e.g., `"2006-01-02"`).
        - `{{$localDatetime format}}`: Current local datetime, same `format` options as `$datetime`.
        - `{{$processEnv VAR_NAME}}`: Value of environment variable `VAR_NAME`.
        - `{{$dotenv VAR_NAME}}`: Value of `VAR_NAME` from a `.env` file in the request file's directory.
    - **Programmatic Variables:** Pass a `map[string]string` of variables directly to `ExecuteFile`. These override file-defined and environment variables of the same name.
- **HTTP Request Execution:**
    - Create a `Client` with options (custom `http.Client`, `BaseURL`, default headers).
    - Execute requests from a `.http` file using `ExecuteFile(ctx, filePath, [programmaticVars])`.
    - Captures detailed response information: status, headers, body, duration.
    - Handles errors during request execution and stores them within the `Response` object.
- **Response Validation:**
    - Compare actual `Response` objects against an expected response file (`.hresp`).
    - Validate status code and status string.
    - Validate headers: exact match for specified keys, or check if actual headers contain specified key-substring pairs.
    - Validate body: exact match or contains substrings.

### Response Body Placeholders (`.hresp` files)

When defining expected responses in `.hresp` files, the body section can use placeholders to match dynamic content. This is particularly useful for values that change with each request, like generated IDs, timestamps, or any arbitrary text.

The following placeholders are supported in the expected response body:

- **`{{$any}}`**: Matches any sequence of characters (including an empty string). This is useful for parts of the response body that are dynamic or not relevant to the current test. It's non-greedy.
  *Example:* `{"message": "Processed item {{$any}} successfully"}`

- **`{{$regexp 'pattern'}}`**: Matches text against the provided Go regular expression `pattern`. The pattern **must be enclosed in backticks (` `)**.
  *Example:* `{"id": "{{$regexp `^[a-f0-9-]+$`}}"}` (matches a UUID-like string)
  *Example:* `{"value": "{{$regexp `\d{3,5}`}}` (matches 3 to 5 digits)

- **`{{$anyGuid}}`**: Matches a standard UUID string (e.g., `123e4567-e89b-12d3-a456-426614174000`).
  *Example:* `{"correlationId": "{{$anyGuid}}"}`

- **`{{$anyTimestamp}}`**: Matches a Unix timestamp (an integer representing seconds).
  *Example:* `{"createdAt": "{{$anyTimestamp}}"}`

- **`{{$anyDatetime 'format'}}`**: Matches a datetime string based on the provided `format`.
    - Predefined formats: `'rfc1123'`, `'iso8601'`.
    - Custom Go layout: If `format` is a Go `time.Parse` layout string, it must be **enclosed in double quotes** (e.g., `{{$anyDatetime ""2006-01-02""}}`).
  *Example (RFC1123):* `{"lastModified": "{{$anyDatetime 'rfc1123'}}"}`
  *Example (ISO8601):* `{"timestamp": "{{$anyDatetime 'iso8601'}}"}`
  *Example (Custom Layout):* `{"eventDate": "{{$anyDatetime ""2006-01-02""}}"}`

**Note on Placeholder Evaluation:** When a response body contains placeholders, the library constructs a single regular expression from the expected body. Each placeholder is converted into its corresponding regex pattern. Literals parts of the expected body are automatically escaped for regex. The entire actual response body is then matched against this master regex.

## Getting Started

### Prerequisites

- Go 1.21 or higher.

### Installation

```bash
go get github.com/bmcszk/go-restclient
```

## Usage

### Client Configuration Options

The `restclient.NewClient()` function accepts functional options to customize the client's behavior:

- **`restclient.WithHTTPClient(customClient *http.Client)`**:
  Allows you to provide your own `*http.Client` instance. This is useful for configuring custom transport, timeouts, redirects, TLS settings, etc. If not provided, a default `*http.Client` is used.

  *Example:*
  ```go
  customTransport := &http.Transport{
      TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
  }
  httpClient := &http.Client{
      Transport: customTransport,
      Timeout:   15 * time.Second,
  }
  client, err := restclient.NewClient(restclient.WithHTTPClient(httpClient))
  ```

- **`restclient.WithBaseURL(baseURL string)`**:
  Sets a base URL that will be prepended to all relative request URLs defined in `.http` files. If a request URL is absolute, the base URL is not used.

  *Example:*
  ```go
  client, err := restclient.NewClient(restclient.WithBaseURL("https://api.example.com/v1"))
  // A request to "/users" in an .http file will become "https://api.example.com/v1/users"
  ```

- **`restclient.WithDefaultHeader(key string, value string)`**:
  Adds a single default header that will be included in every request sent by the client. If a request itself defines a header with the same key, the request's header value will typically take precedence (standard HTTP client behavior may vary slightly based on canonicalization, but usually the last one set for a key is used, or multiple are sent if the header key allows).

  *Example:*
  ```go
  client, err := restclient.NewClient(restclient.WithDefaultHeader("X-Client-ID", "my-app-123"))
  ```

- **`restclient.WithDefaultHeaders(headers http.Header)`**:
  Adds multiple default headers from an `http.Header` map. Similar precedence rules apply as with `WithDefaultHeader`.

  *Example:*
  ```go
  defaultHeaders := make(http.Header)
  defaultHeaders.Add("X-App-Name", "MyApplication")
  defaultHeaders.Add("X-App-Version", "2.0.1")
  client, err := restclient.NewClient(restclient.WithDefaultHeaders(defaultHeaders))
  ```

These options can be combined when creating a new client:
```go
client, err := restclient.NewClient(
    restclient.WithBaseURL("https://api.myapp.com"),
    restclient.WithDefaultHeader("Authorization", "Bearer default-token"),
    // ... other options
)
```

### 1. Define HTTP Requests

Create a `.http` (or `.rest`) file with your requests.

**Example: `api_requests.http`**
```http
@baseUrl = https://api.example.com
@defaultUser = file_user_id

### Get a specific user, potentially overridden by programmatic var
GET {{baseUrl}}/users/{{userId}}
X-Request-ID: {{$guid}}
Authorization: Bearer {{authToken}}

### Create a new product
POST {{baseUrl}}/products
Content-Type: application/json

{
  "productId": "{{$randomInt 1000 9999}}",
  "name": "Super Widget {{productSuffix}}" 
}
```

### 2. Define Expected Responses (Optional)

Create an `.hresp` file to specify expected responses for validation. Each response corresponds to a request in your `.http` file, in the same order, separated by `###`.

**Example: `api_expected.hresp`** (for the first two requests in `api_requests.http`)
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "message": "User details retrieved"
}

###

HTTP/1.1 201 Created
Content-Type: application/json

{
  "message": "Product created successfully"
}
```

### 3. Execute and Validate in Go

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bmcszk/go-restclient"
)

func main() {
	client, err := restclient.NewClient(
		// Example client options:
		// restclient.WithBaseURL("https://api.global.com"),
		// restclient.WithDefaultHeader("X-App-Version", "1.2.3"),
		// restclient.WithHTTPClient(&http.Client{Timeout: 15 * time.Second}),
		// Programmatic variables are now set via WithVars
		restclient.WithVars(map[string]interface{}{
			"userId":        "prog_user_override_123",
			"authToken":     "prog_auth_token_xyz",
			"productSuffix": "Deluxe",
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	requestFilePath := "api_requests.http"       // Your .http file
	expectedResponseFilePath := "api_expected.hresp" // Your .hresp file for validation

	responses, err := client.ExecuteFile(context.Background(), requestFilePath) // No programmaticAPIVars here
	if err != nil {
		log.Fatalf("Failed to execute request file: %v", err)
	}

	fmt.Printf("Executed %d requests from %s.\n", len(responses), requestFilePath)

	// Validate the responses
	// The client instance is used for variable substitution within the .hresp file (e.g., for {{$uuid}})
	validationErr := client.ValidateResponses(expectedResponseFilePath, responses) // Pass the slice directly
	if validationErr != nil {
		log.Fatalf("Response validation failed: %v", validationErr)
	}

	fmt.Println("All requests executed and validated successfully!")

	// Optional: Iterate through responses for manual checks or logging.
	fmt.Println("\nIndividual response details (optional logging):")
	for i, resp := range responses {
		if resp.Error != nil {
			fmt.Printf("Request #%d (%s %s) had an execution error: %v\n",
				i+1, resp.Request.Method, resp.Request.RawURLString, resp.Error)
			continue
		}
		fmt.Printf("Request #%d (%s %s): Status %s\n  Body: %s\n",
			i+1, resp.Request.Method, resp.Request.URL, resp.Status, resp.BodyString)
	}
}
```

## Development

### Running Tests

- **Unit Tests:** `make test-unit`

### Linting and Checks

- **All pre-commit checks (lint, build, unit tests):** `make check`

## Contributing

Contributions are welcome! Please follow standard Go best practices and ensure `make check` passes before submitting pull requests.

## License

MIT License (To be formally added - assuming MIT for now). 
