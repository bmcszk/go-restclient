# Go REST Client Library (`go-restclient`)

Last Updated: 2025-05-28

## Overview

`go-restclient` is a Go library designed to simplify HTTP request execution and validation. It allows developers to define HTTP requests in simple text files (`.rest` or `.http` format, similar to RFC 2616 and popular REST client tools), send these requests, and validate the responses against expected outcomes.

This library is suitable for programmatic use within Go applications, particularly for integration and end-to-end testing.

## Features

- **Parse `.rest`/`.http` files:**
    - Supports multiple requests per file, separated by `###`.
    - Handles request method, URL, HTTP version, headers, and body.
    - Supports comments (`#`) and named requests (e.g., `### My Request Name`).
- **Dynamic Variable Substitution:**
    - **Custom Variables:** Define and use variables within request files (e.g., `@baseUrl = https://api.example.com`).
    - **System Variables:**
        - `{{$guid}}`: Generates a new UUID.
        - `{{$randomInt min max}}`: Generates a random integer within the range [min, max] (inclusive). If no arguments, defaults to [0, 100].
        - `{{$timestamp}}`: Current UTC Unix timestamp (seconds).
        - `{{$datetime format}}`: Current UTC datetime formatted according to `format` (e.g., `rfc1123`, `iso8601`, or a custom Go layout string like `"2006-01-02T15:04:05Z07:00"`).
        - `{{$localDatetime format}}`: Current local datetime formatted similarly to `$datetime`.
        - `{{$processEnv variableName}}`: Substitutes with the value of the environment variable `variableName`.
        - `{{$dotenv variableName}}`: Substitutes with the value of `variableName` from a `.env` file in the same directory as the request file.
- **HTTP Request Execution:**
    - Create a `Client` with options (custom `http.Client`, `BaseURL`, default headers).
    - Execute all requests from a `.rest` or `.http` file using `ExecuteFile(ctx context.Context, requestFilePath string)`.
    - Captures detailed response information: status, headers, body, duration.
    - Handles errors during request execution and stores them within the `Response` object.
- **Response Validation:**
    - Compare actual `Response` objects against an expected response file (`.hresp`).
    - Validate status code and status string.
    - Validate headers: exact match for specified keys, or check if actual headers contain specified key-substring pairs.
    - Validate body: exact match or contains substrings.

## Getting Started

### Prerequisites

- Go 1.21 or higher.

### Installation

```bash
go get github.com/bmcszk/go-restclient
```

## Usage

### 1. Define HTTP Requests

Create a `.http` (or `.rest`) file with your requests.

**Example: `api_requests.http`**
```http
@baseUrl = https://api.example.com
@authToken = your-secret-token

### Get a specific user
GET {{baseUrl}}/users/{{$guid}}
X-Request-ID: {{$guid}}
Authorization: Bearer {{authToken}}
Accept: application/json

### Create a new product
# Example using a system variable in the body
POST {{baseUrl}}/products
Content-Type: application/json
X-Correlation-ID: {{$guid}}

{
  "productId": "{{$randomInt 1000 9999}}",
  "name": "Super Widget",
  "timestamp": "{{$timestamp}}"
}

### Get orders using an environment variable for the filter
GET {{baseUrl}}/orders?status={{$processEnv ORDER_STATUS_FILTER}}

### Load user from .env file
# Assuming .env file in the same directory has: USER_ID_FROM_ENV=user_from_dotenv_123
GET {{baseUrl}}/users/{{$dotenv USER_ID_FROM_ENV}}
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
	"os"

	"github.com/bmcszk/go-restclient"
)

func main() {
	// Example: Set an environment variable for $processEnv
	os.Setenv("ORDER_STATUS_FILTER", "pending")
	// Example: Create a .env file for $dotenv
	_ = os.WriteFile(".env", []byte("USER_ID_FROM_ENV=user_from_dotenv_abc"), 0644)
	defer os.Remove(".env")


	client, err := restclient.NewClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	requestFilePath := "api_requests.http" // Your .http file
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)
	if err != nil {
		// This error covers file-level issues or critical request setup failures.
		log.Fatalf("Failed to execute request file: %v", err)
	}

	fmt.Printf("Executed %d requests from %s\n", len(responses), requestFilePath)

	// Print basic info for each response
	for i, resp := range responses {
		if resp.Error != nil {
			fmt.Printf("Request #%d (%s %s) failed during execution: %v\n",
				i+1, resp.Request.Method, resp.Request.RawURLString, resp.Error)
			continue
		}
		fmt.Printf("Request #%d (%s %s): Status %s, Body: %s\n",
			i+1, resp.Request.Method, resp.Request.URL, resp.Status, resp.BodyString)
	}

	// Optional: Validate against an expected response file
	expectedResponseFilePath := "api_expected.hresp"
	if _, err := os.Stat(expectedResponseFilePath); err == nil {
		validationErr := restclient.ValidateResponses(expectedResponseFilePath, responses...)
		if validationErr != nil {
			log.Fatalf("Validation failed: %v", validationErr)
		}
		fmt.Println("All responses validated successfully against " + expectedResponseFilePath + "!")
	} else {
		fmt.Println("Expected response file not found, skipping validation.")
	}
}
```

## Development

### Running Tests

- **Unit Tests:** `make test-unit` (or `go test -tags=unit ./...`)

### Linting and Checks

- **All pre-commit checks (lint, build, unit tests):** `make check`

## Contributing

Contributions are welcome! Please follow standard Go best practices and ensure `make check` passes before submitting pull requests.

## License

MIT License (To be formally added - assuming MIT for now). 
