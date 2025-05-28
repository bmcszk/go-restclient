# Go REST Client Library (`go-restclient`)

Last Updated: 2025-05-28

## Overview

`go-restclient` is a Go library designed to simplify HTTP request execution and validation. It allows developers to define HTTP requests in simple text files (`.rest` or `.http` format, similar to RFC 2616 and popular REST client tools), send these requests, and validate the responses against expected outcomes.

This library is suitable for programmatic use within Go applications.

## Features

- **Parse `.rest`/`.http` files:** 
    - Supports multiple requests per file, separated by `###`.
    - Handles request method, URL, HTTP version, headers, and body.
    - Supports comments (`#`) and named requests (e.g., `### My Request Name`).
- **HTTP Request Execution:**
    - Create a `Client` with options (custom `http.Client`, `BaseURL`, default headers).
    - Execute all requests from a `.rest` or `.http` file using `ExecuteFile(ctx context.Context, requestFilePath string)`.
    - Captures detailed response information: status, headers, body, duration, TLS details.
    - Handles errors during request execution and stores them within the `Response` object.
- **Response Validation:**
    - Compare actual `Response` against an `ExpectedResponse` struct.
    - Validate status code and status string.
    - Validate headers: exact match for specified keys, or check if actual headers contain specified key-substring pairs.
    - Validate body: exact match, contains substrings, not-contains substrings.
    - Validate JSON body content using JSONPath expressions.
- **Extensible:** Designed with clear structs and functions for further extension.

## Getting Started

### Prerequisites

- Go 1.21 or higher (developed with Go 1.24.3).

### Installation

To use `go-restclient` in your Go project:

```bash
go get github.com/bmcszk/go-restclient
```

## Usage

### 1. Create a `.rest` file

**Example: `requests.rest`**
```http
### Get User
GET https://api.example.com/users/123
Accept: application/json
X-API-Key: your-api-key

### Create Product
POST https://api.example.com/products
Content-Type: application/json

{
  "name": "Awesome Gadget",
  "price": 99.99
}
```

### 2. Execute requests using the client

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
		// Example: Set a default header for all requests
		// restclient.WithDefaultHeader("User-Agent", "MyTestApp/1.0"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Pass a context, e.g., context.Background() or context.TODO()
	responses, err := client.ExecuteFile(context.Background(), "requests.rest")
	if err != nil {
		// This error is for file-level issues (e.g., file not found, parse error for whole file)
		log.Fatalf("Failed to execute request file: %v", err)
	}

	for _, resp := range responses {
		if resp.Error != nil {
			fmt.Printf("Request to %s failed: %v\n", resp.Request.URL, resp.Error)
			continue
		}

		fmt.Printf("Response for %s %s:\n", resp.Request.Method, resp.Request.URL)
		fmt.Printf("Status: %s\n", resp.Status)
		fmt.Printf("Body:\n%s\n", resp.BodyString)
		fmt.Println("----------")

		// Example Validation (if you have an ExpectedResponse)
		// expected := &restclient.ExpectedResponse{ ... }
		// validationResult := restclient.ValidateResponse(resp, expected)
		// if !validationResult.Passed {
		// 	 fmt.Printf("Validation Failed: %v\n", validationResult.Mismatches)
		// }
	}
}

```

### 3. Validating with Request and Response Files

A common use case is to define your HTTP requests in one file and their corresponding expected responses in another. `go-restclient` supports this workflow using `.http` (or `.rest`) files for requests and `.hresp` files for expected responses.

**Example: `my_requests.http`**

This file defines the actual HTTP requests to be sent. Variables like `{{.ServerURL}}` can be used if you process the file through a template engine before parsing, or you can use absolute URLs directly.

```http
GET {{.ServerURL}}/req1

###
POST {{.ServerURL}}/req2
Content-Type: application/json

{"key": "value"}
```

**Example: `my_expected_responses.hresp`**

This file defines the expected outcomes for the requests in `my_requests.http`, in the same order. Each expected response is separated by `###`.

```http
HTTP/1.1 200 OK

response1

###

HTTP/1.1 201 Created

response2
```

**Go Code Example:**

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bmcszk/go-restclient"
)

// Assume startMockServer() sets up a test server and returns its URL.
// In a real scenario, ServerURL might come from config or be hardcoded if static.
var mockServerURL string 

func main() {
	// For this example, let's assume my_requests.http needs a ServerURL.
	// In a real test, you might dynamically create this file content.
	// For simplicity, we'll assume my_requests.http directly contains usable URLs
	// or you pre-process it. For this example, let's use placeholder paths.
	
	// Effective request file path (can be the direct path if no templating is needed)
	requestFilePath := "my_requests.http" // Or path to your pre-processed file
	expectedResponseFilePath := "my_expected_responses.hresp"

	client, err := restclient.NewClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	responses, err := client.ExecuteFile(context.Background(), requestFilePath)
	if err != nil {
		// This error is for file-level issues (e.g., file not found, parse error for whole file)
		// or if any request within the file critically fails during execution setup.
		log.Fatalf("Failed to execute request file: %v", err)
	}

	fmt.Printf("Executed %d requests from %s\n", len(responses), requestFilePath)

	// Validate all responses against the expected responses file
	validationErr := restclient.ValidateResponses(expectedResponseFilePath, responses...)
	if validationErr != nil {
		log.Fatalf("Validation failed: %v", validationErr)
	}

	fmt.Println("All requests executed and validated successfully!")

	// Individual response details are still available in the 'responses' slice
	for i, resp := range responses {
		if resp.Error != nil { // This error is for individual request execution issues (e.g. network error)
			fmt.Printf("Request #%d to %s failed during execution: %v\n", i+1, resp.Request.URL, resp.Error)
			continue
		}
		fmt.Printf("Details for Request #%d (%s %s): Status %s\n", i+1, resp.Request.Method, resp.Request.URL, resp.Status)
	}
}

```

### 4. Advanced: Programmatic Validation and JSONPath

(Details on loading `ExpectedResponse` from files and full validation flow to be expanded based on chosen file format for expected responses e.g. JSON, YAML, or a simplified `.httpresponse` format)

Currently, `ExpectedResponse` can be defined programmatically or loaded from a JSON file using `LoadExpectedResponseFromJSONFile`.

```go
// Programmatic example
expected := &restclient.ExpectedResponse{
    StatusCode: restclient.IntPtr(200),
    BodyContains: []string{"success"},
    HeadersContain: map[string]string{"Content-Type": "application/json"},
    JSONPathChecks: map[string]interface{}{"$.data.id": 100},
}

// Assuming 'resp' is the *restclient.Response from ExecuteFile/ExecuteRequest
validationResult := restclient.ValidateResponse(resp, expected)
if !validationResult.Passed {
    fmt.Printf("Validation failed for %s:\n", resp.Request.Name)
    for _, mismatch := range validationResult.Mismatches {
        fmt.Printf(" - %s\n", mismatch)
    }
}
```

## Development

### Running Tests

- **Unit Tests:** `make test-unit` (or `go test -cover ./...`)

### Linting and Checks

- **Lint:** `make lint`
- **All pre-commit checks (lint, build, unit tests):** `make check`

## Contributing

Contributions are welcome! Please follow standard Go best practices and ensure `make check` passes before submitting pull requests.

(Further contribution guidelines to be added if the project grows).

## License

MIT License (To be formally added - assuming MIT for now). 
