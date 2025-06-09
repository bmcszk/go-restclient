**Version:** 2.0
**Date:** 2025-06-09
**Status:** In Progress

# Product Requirements Document: HTTP Client Syntax Compatibility

## 1. Introduction

This document outlines the requirements for enhancing `go-restclient` to achieve comprehensive compatibility with the HTTP request syntax documented in `docs/http_syntax.md`. This document consolidates both JetBrains HTTP Client and VS Code REST Client syntax. The goal is to allow users to leverage `.http` files with this common syntax in the `go-restclient` library.

## 2. Goals

* Parse and execute HTTP requests defined in `.http` files that conform to the common HTTP client syntax.
* Support all placeholders, variables, and dynamic expressions documented in `docs/http_syntax.md`.
* Implement environment management via `http-client.env.json` files.
* Support all documented request components (headers, authentication, body types, etc.).
* Provide response handling and validation capabilities with `.hresp` files.
* Maintain backward compatibility with existing `go-restclient` features.
* Achieve 90%+ compatibility with JetBrains HTTP Client and VS Code REST Client syntax.

## 3. Target Audience

* Developers using `go-restclient` who also use JetBrains HTTP client or VS Code REST client tools.
* Users looking for a CLI tool that can run `.http` files with the common syntax.
* Go developers needing to integrate HTTP request files into their applications.

## 4. Key Features & Requirements

### 4.1 Request Structure Basics ✅ IMPLEMENTED

* **FR1.1:** Support `.http` and `.rest` file extensions. ✅
* **FR1.2:** Support parsing multiple requests within a single file, delimited by `###`. ✅
* **FR1.3:** Support request naming via `### Request Name` syntax or `# @name requestName` directive. ✅
* **FR1.4:** Support comments using `#` and `//`. ✅
* **FR1.5:** Support all major HTTP methods (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE, CONNECT). ✅
* **FR1.6:** Support request line format: `[METHOD] [URL] [HTTP_VERSION]` (HTTP version optional). ✅
* **FR1.7:** Parse headers as `[Header-Name]: [Header-Value]`. ✅
* **FR1.8:** Handle standard body formats after headers separated by blank line. ✅
* **FR1.9:** Support short form for GET requests (URL only without method). ✅
* **FR1.10:** Support query parameters on multiple lines with `?` and `&` prefixes. ✅

### 4.2 Environment Variables and Placeholders ✅ IMPLEMENTED

* **FR2.1:** Support variable substitution in URL, headers, and body using `{{variable_name}}` syntax. ✅
* **FR2.2:** Load environment variables from `http-client.env.json` and `http-client.private.env.json`. ✅
* **FR2.3:** Support in-place variables defined with `@name = value` syntax. ✅
* **FR2.4:** Support special `$shared` environment that applies across all environments. ✅
* **FR2.5:** Support variable fallback syntax `{{variable | fallback_value}}`. ✅
* **FR2.6:** Support environment selection via client configuration. ✅
* **FR2.7:** Support accessing shared variables within environment definitions. ✅

### 4.3 Dynamic System Variables ✅ IMPLEMENTED

* **FR3.1:** Support common placeholders available in both clients: ✅
  * `{{$uuid}}` / `{{$guid}}` - Generate UUID v4 ✅
  * `{{$timestamp}}` - Current Unix timestamp ✅
  * `{{$isoTimestamp}}` - ISO-8601 format timestamp ✅
  * `{{$datetime format}}` - UTC datetime with format ✅
  * `{{$localDatetime format}}` - Local datetime with format ✅
  * `{{$randomInt}}` - Random integer (0-1000) ✅
  * `{{$randomInt min max}}` - Random integer in range ✅

* **FR3.2:** Support JetBrains-specific placeholders: ✅
  * `{{$random.integer(min, max)}}` - Random integer in range ✅
  * `{{$random.float(min, max)}}` - Random float in range ✅
  * `{{$random.alphabetic(length)}}` - Random alphabetic string ✅
  * `{{$random.alphanumeric(length)}}` - Random alphanumeric string ✅
  * `{{$random.hexadecimal(length)}}` - Random hexadecimal string ✅
  * `{{$random.email}}` - Random email ✅
  * `{{$randomBoolean}}`, `{{$randomColor}}`, `{{$randomEmail}}` - Extended random values ✅
  
* **FR3.3:** Support system environment access: ✅
  * `{{$env.VAR_NAME}}` - System environment variable (JetBrains) ✅
  * `{{$processEnv VAR_NAME}}` - System environment variable (VS Code) ✅
  * `{{$dotenv VAR_NAME}}` - Value from `.env` file (VS Code) ✅

* **FR3.4:** Support format options for datetime variables: ✅
  * `rfc1123`, `iso8601` predefined formats ✅
  * Custom Go layout strings ✅

### 4.4 Request Bodies ✅ IMPLEMENTED

* **FR4.1:** Support `application/json` bodies. ✅
* **FR4.2:** Support `application/x-www-form-urlencoded` bodies with proper encoding. ✅
* **FR4.3:** Support `multipart/form-data` bodies including file uploads. ✅
* **FR4.4:** Support `text/plain` and other raw text bodies. ✅
* **FR4.5:** Support GraphQL request format (as JSON). ✅
* **FR4.6:** Support external file references `< ./path/to/file`. ✅
* **FR4.7:** Support variable substitution in external files `<@ ./path/to/file`. ✅ IMPLEMENTED
* **FR4.8:** Support encoding specification for external files. ✅ IMPLEMENTED
* **FR4.9:** Support form data on multiple lines with `&` prefixes. ✅

### 4.5 Authentication ✅ IMPLEMENTED

* **FR5.1:** Support Basic Authentication (`Authorization: Basic`) and URL-based (`user:password@domain`). ✅
* **FR5.2:** Support Bearer token authentication (`Authorization: Bearer`). ✅
* **FR5.3:** Support OAuth authentication flow using request references. ✅
* **FR5.4:** Support custom header-based authentication schemes. ✅

### 4.6 Request Settings ✅ IMPLEMENTED

* **FR6.1:** Support request-specific options via `@name`, `@no-redirect`, `@no-cookie-jar` directives. ✅
* **FR6.2:** Support request timeout setting via `@timeout` directive. ✅
* **FR6.3:** Support `@no-log` directive for excluding requests from history. ✅
* **FR6.4:** Support settings placed before or after request line. ✅

### 4.7 Response Handling and Validation ✅ IMPLEMENTED

* **FR7.1:** Support defining expected responses in `.hresp` files for testing. ✅
* **FR7.2:** Support response references for chained requests (`{{requestName.response.body.field}}`). ✅
* **FR7.3:** Support response validation placeholders: ✅
  * `{{$any}}` - Matches any sequence ✅
  * `{{$regexp 'pattern'}}` - Regex pattern matching ✅
  * `{{$anyGuid}}` - UUID string matching ✅
  * `{{$anyTimestamp}}` - Unix timestamp matching ✅
  * `{{$anyDatetime 'format'}}` - Datetime matching ✅
* **FR7.4:** Support variable substitution in `.hresp` files. ✅
* **FR7.5:** Support status code, headers, and body validation. ✅
* **FR7.6:** Support multiple expected responses in single `.hresp` file. ✅

### 4.8 Request Imports ✅ IMPLEMENTED

* **FR8.1:** Support importing requests from other `.http` files. ✅
* **FR8.2:** Support correct variable scoping and overriding in imports. ✅
* **FR8.3:** Detect and prevent circular imports. ✅
* **FR8.4:** Support relative and absolute import paths. ✅

### 4.9 Cookies and Redirect Handling ✅ IMPLEMENTED

* **FR9.1:** Support automatic cookie management between requests in the same file. ✅
* **FR9.2:** Support redirect following with option to disable (`@no-redirect`). ✅
* **FR9.3:** Support `@no-cookie-jar` directive to disable cookie handling per request. ✅

## 5. Non-Functional Requirements

* **NFR1:** Performance - Script execution and request processing should be reasonably performant.
* **NFR2:** Error Handling - Clear and informative error messages for syntax errors and failed assertions.
* **NFR3:** Usability - CLI interface should be intuitive for selecting files, named requests, and environments.
* **NFR4:** Compatibility - Maintain backward compatibility with existing `go-restclient` features.

## 6. Out of Scope

* Pre-request and post-request scripting ❌ NOT IMPLEMENTED.
* Request history management ❌ NOT IMPLEMENTED.
* UI-specific integrations of the JetBrains or VS Code clients ❌ OUT OF SCOPE.
* cURL import/export functionality ❌ NOT IMPLEMENTED.
* VS Code-specific Azure AD token placeholders (`{{$aadToken}}`, `{{$aadV2Token}}`) ❌ NOT IMPLEMENTED.
* Full JetBrains Faker library variables ⚠️ PARTIALLY IMPLEMENTED (basic categories only).

## 7. Implementation Status Summary

**Overall Compatibility Status: 93% Complete**

### ✅ Fully Implemented Features (29/31)
- All basic HTTP request structure and parsing
- Complete variable substitution system with precedence
- All common system variables (UUID, timestamp, random values)
- JetBrains-style random generators and environment access
- All request body types, file uploads, and external file processing with variable substitution and encoding support
- Complete authentication support (Basic, Bearer, OAuth flow)
- Request settings and directives
- Response validation with placeholders
- Request imports with circular detection
- Cookie and redirect handling
- Environment file management

### ⚠️ Partially Implemented (1/31)
- JetBrains Faker library (basic categories implemented)

### ❌ Not Implemented (1/31)
- Pre-request/post-request scripting (intentionally out of scope)

### Current Implementation Gaps
1. **Advanced Faker Variables**: While basic random generators are implemented, the full Java Faker library categories are not complete
2. **Azure AD Integration**: VS Code-specific Azure AD token acquisition is not implemented
3. **cURL Integration**: Import/export functionality is not available

The library successfully implements the core functionality needed for JetBrains HTTP Client and VS Code REST Client compatibility, with comprehensive support for the most commonly used features.
