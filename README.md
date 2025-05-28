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
	"os"

	"github.com/bmcszk/go-restclient"
)

func main() {
	// Example: Set an environment variable for $processEnv (if used in file)
	// os.Setenv("ORDER_STATUS_FILTER", "pending")
	// Example: Create a .env file for $dotenv (if used in file)
	// _ = os.WriteFile(".env", []byte("USER_ID_FROM_ENV=user_from_dotenv_abc"), 0644)
	// defer os.Remove(".env")


	client, err := restclient.NewClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Programmatic variables to pass to ExecuteFile
	// These will override any 'userId' or 'authToken' defined in the .http file or environment.
	programmaticAPIVars := map[string]string{
		"userId":        "prog_user_override_123",
		"authToken":     "prog_auth_token_xyz",
		"productSuffix": "Deluxe", // This var might only be used programmatically
	}

	requestFilePath := "api_requests.http" // Your .http file

	// Pass programmatic vars as the optional third argument
	responses, err := client.ExecuteFile(context.Background(), requestFilePath, programmaticAPIVars)
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
		// Note: resp.Request.URL will show the URL *after* all substitutions
		fmt.Printf("Request #%d (%s %s): Status %s\n  Body: %s\n",
			i+1, resp.Request.Method, resp.Request.URL, resp.Status, resp.BodyString)
	}

	// Optional: Validate against an expected response file
	// ... (validation logic remains the same) ...
	expectedResponseFilePath := "api_expected.hresp" // Assuming this file exists and matches executed requests
	if _, statErr := os.Stat(expectedResponseFilePath); statErr == nil {
		validationErr := restclient.ValidateResponses(expectedResponseFilePath, responses...)
		if validationErr != nil {
			log.Fatalf("Validation failed: %v", validationErr)
		}
		fmt.Println("All responses validated successfully against " + expectedResponseFilePath + "!")
	} else {
		fmt.Println("Expected response file not found or error stating: " + statErr.Error() + ", skipping validation.")
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
