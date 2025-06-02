**Version:** 1.0
**Date:** 2025-06-02
**Status:** Draft

# Product Requirements Document: JetBrains HTTP Client Compatibility

## 1. Introduction

This document outlines the requirements for enhancing `go-restclient` to achieve compatibility with core features of the JetBrains HTTP client, as described in their official documentation. The goal is to allow users to leverage `.http` files written for the JetBrains environment with `go-restclient`.

## 2. Goals

*   Parse and execute HTTP requests defined in `.http` files that conform to the JetBrains HTTP client syntax.
*   Support multiple requests within a single file.
*   Implement variable substitution using `{{variable_name}}` syntax.
*   Support environment-specific configurations via an environment file.
*   Enable pre-request and post-request scripting using JavaScript.
*   Allow importing requests from other `.http` files.
*   Handle common request body content types.

## 3. Target Audience

*   Developers using `go-restclient` who also use or want to use the JetBrains HTTP client format for defining HTTP requests.
*   Users looking for a CLI tool that can run JetBrains-style `.http` files.

## 4. Key Features & Requirements

### 4.1. File Format and Parsing

*   **FR1.1:** Support for `.http` and `.rest` file extensions.
*   **FR1.2:** Parse multiple requests within a single file, delimited by `###`.
*   **FR1.3:** Support comments using `#` and `//`.
*   **FR1.4:** Support request naming:
    *   Via comment above the request: `### My Request Name`
    *   Via directive: `// @name MyRequestName` or `# @name MyRequestName`
*   **FR1.5:** Parse request method, URL, HTTP version, headers, and body.

### 4.2. Variables and Environments

*   **FR2.1:** Support variable substitution in URL, headers, and body using `{{variable_name}}` syntax.
*   **FR2.2:** Load environment variables from a `http-client.env.json` file located in the same directory as the `.http` file or a project root.
    *   The file should contain a JSON object where keys are environment names and values are objects of key-value pairs.
    *   Example: `{"dev": {"host": "localhost:3000"}, "prod": {"host": "api.example.com"}}`
*   **FR2.3:** Allow specifying an active environment for a run (e.g., via CLI flag).
*   **FR2.4:** Variables defined in scripts (see 4.4) should also be available for substitution.

### 4.3. Request Imports

*   **FR3.1:** Support importing requests or shared components from other `.http` files.
*   **FR3.2:** Assumed import syntax (pending confirmation): `// @import "path/to/another.http"` or `# @import "path/to/another.http"`. The imported file's requests should be usable.

### 4.4. JavaScript Scripting

*   **FR4.1:** Embed a JavaScript interpreter (e.g., `goja`).
*   **FR4.2:** **Pre-request Scripts:**
    *   Execute JS code *before* a request is sent.
    *   Syntax: Inline `< {% script_code %}` or external `< path/to/script.js`.
    *   Provide a `request` object to the script, allowing modification of variables for the current request (e.g., `request.variables.set("key", "value")`).
*   **FR4.3:** **Response Handler (Post-request) Scripts:**
    *   Execute JS code *after* a response is received.
    *   Syntax: Inline `> {% script_code %}` or external `> path/to/script.js`.
    *   Provide a `client` object:
        *   `client.global.set("varName", value)`: Store variables globally for subsequent requests.
        *   `client.test("Test Name", function() { /* ... */ })`: Define a test case.
        *   `client.assert(condition, "Failure message")`: Perform assertions.
        *   `client.log("message")`: Log messages.
    *   Provide a `response` object:
        *   `response.status`: HTTP status code (number).
        *   `response.contentType`: Response content type (string).
        *   `response.body`: Response body (parsed as JSON/XML if applicable, otherwise string).
        *   `response.headers`: Object representing response headers.
*   **FR4.4:** Support ES6 module `import` and `export` within JS scripts for modularity.

### 4.5. Request Body Handling

*   **FR5.1:** Support `application/json` bodies.
*   **FR5.2:** Support `text/plain` and other raw text bodies.
*   **FR5.3:** Support `application/x-www-form-urlencoded` bodies, including correct encoding of special characters.
*   **FR5.4:** Support `multipart/form-data` bodies.

### 4.6. Execution

*   **FR6.1:** Allow execution of a specific named request from a file.
*   **FR6.2:** Allow execution of all requests in a file sequentially.
*   **FR6.3:** Variables set by `client.global.set()` in one request's script must be available to subsequent requests in the same execution run.

## 5. Non-Functional Requirements

*   **NFR1:** Performance: Script execution and request processing should be reasonably performant.
*   **NFR2:** Error Handling: Clear and informative error messages for syntax errors in `.http` files, script execution errors, and failed assertions.
*   **NFR3:** Usability: CLI interface should be intuitive for selecting files, named requests, and environments.

## 6. Out of Scope (Version 1.0)

*   gRPC, WebSocket, GraphQL protocols.
*   Advanced SSL/TLS client certificate configuration.
*   Proxy configuration via `.http` file syntax.
*   UI-specific integrations of the JetBrains client (e.g., run configurations UI, direct browser opening).
*   Response history persistence beyond the current session of `client.global` variables.
*   Automatic conversion from/to cURL or Postman collections.

## 7. Open Questions / Assumptions to Verify

*   **OQ1:** Confirm the exact syntax for request imports (`// @import` is an assumption).
*   **OQ2:** Detailed structure of `http-client.env.json` if it supports more than simple key-value pairs per environment (e.g., nested structures). For now, simple key-value is assumed.
