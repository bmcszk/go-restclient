# Go REST Client Library (`go-restclient`)

Last Updated: 2025-05-27

## Overview

`go-restclient` is a Go library designed to simplify HTTP request testing in End-to-End (E2E) test suites. It allows developers to define HTTP requests in simple text files (`.rest` or `.http` format) and validate responses against expected outcomes also defined in files.

This library is inspired by tools like the VSCode REST Client extension but is intended for programmatic use within Go applications, particularly for testing.

## Features (Planned)

- Parse `.rest`/`.http` files for request details (method, URL, headers, body).
- Send HTTP requests.
- Capture HTTP responses.
- Compare actual responses against expected responses (status code, headers, body).
- Support for variables in request files.

## Getting Started

_(Instructions to be added once the library is more mature)_.

### Prerequisites

- Go (version X.Y.Z or higher - to be specified)

### Installation

```bash
# To be determined (e.g., go get github.com/bmcszk/go-restclient)
```

## Usage

_(Code examples and usage instructions to be added)_.

### Request File Format

The request files (`.rest` or `.http`) should follow a simple format:

```http
### Example Request
# Comment: This is a GET request to a test API
GET https://httpbin.org/get
Accept: application/json

```

Multiple requests can be separated by `###`.

### Expected Response Format

_(Details on how expected responses will be defined to be added)_.

## Development

### Running Tests

```bash
make test-unit
make test-e2e
```

### Linting

```bash
make lint
```

### Building

```bash
make build
```

## Contributing

_(Contribution guidelines to be added)_.

## License

_(License information to be added - likely MIT)_. 
