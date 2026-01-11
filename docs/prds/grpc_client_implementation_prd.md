# gRPC Client Implementation PRD

## 1. Overview
The goal is to extend the `go-restclient` library to support gRPC requests while maintaining strict consistency with the existing `.http` file syntax, variable substitution system, and response validation logic. This will allow users to test gRPC services alongside REST APIs using a unified workflow.

## 2. Goals & Non-Goals
### Goals
- Support `GRPC` as a method in `.http` files.
- Enable invoking gRPC methods using standard `host/Package.Service/Method` syntax.
- Support JSON bodies for request payloads (automatically converted to Protobuf messages).
- Support standard variable substitution (`{{var}}`, `.env`, system vars) in gRPC requests.
- Consistent response output (Status => gRPC Status, Body => JSON representation of response).
- Support metadata (headers) in the same format as HTTP headers.

### Non-Goals
- Supporting streaming (Server/Client/Bidirectional) in the first iteration (Unary only).
- Compiling `.proto` files dynamically (we will rely on server reflection or `grpcurl`-style dynamic invocation).
- Binary protobuf body support (only JSON-mapped bodies for now).

## 3. User Experience (UX)
### 3.1 Request Syntax
Users will define gRPC requests in `.http` files using the `GRPC` method keyword.

**Format:**
```http
GRPC host:port/package.Service/Method
Metadata-Key: value

{
  "field": "value"
}
```

**Example:**
```http
### Get User Details (gRPC)
@userId = 12345

GRPC localhost:50051/users.UserService/GetUser
Authorization: Bearer {{token}}
X-Request-ID: {{$guid}}

{
  "id": "{{userId}}"
}
```

### 3.2 Variable Substitution
All variable types (Custom, System, Faker, Environment) must work exactly as they do for REST requests.

### 3.3 Response Format
The response object returned to Go consumers (and printed in logs) should map gRPC concepts to the existing `Response` struct where possible:
- `StatusCode`: Mapped from `grpc.Code` (e.g., `0` for OK, `5` for NotFound).
- `Status`: String representation of the code (e.g., "OK", "NOT_FOUND").
- `Body`: JSON representation of the protobuf response message.
- `Headers`: Response metadata/trailers.

## 4. Technical Design

### 4.1 Parser Changes
- **Status**: minimal changes expected.
- The `parser` currently accepts arbitrary method strings if they match `isValidHTTPToken`. `GRPC` is valid.
- The URL parsing logic needs to ensure it doesn't enforce `http://` or `https://` schemes strictly if `GRPC` is the method (or treat `localhost:50051` correctly).
- **Validation**: Ensure `GRPC` requests are validated to have a body if the method requires it (Unary).

### 4.2 Client Execution Logic
We will modify `Client.executeRequest` to check the method.
- If `Method == "GRPC"`, branch to a new internal method `executeGRPCRequest`.
- **Dependency**: Use `google.golang.org/grpc` and likely `github.com/jhump/protoreflect` (similar to `grpcurl`) to handle dynamic invocation without generated code.
- **Connection**:
    - Establish a `grpc.Dial` connection to the host.
    - Use **Server Reflection** to discover the method signature and message types.
    - If reflection is disabled, consider supporting a local `.proto` file path in a `@proto` directive (Phase 2).
    - Convert the JSON body from the `.http` file into the dynamic protobuf message.
    - Invoke the method.
    - Marshal the response protobuf back to JSON for the `Response.Body`.

### 4.3 Response Handling
The `Response` struct in `response.go` is HTTP-centric (`http.Header`, `StatusCode int`).
- `StatusCode` can store the `grpc.Code` (casted to int).
- `Headers` can store gRPC metadata.
- `Proto` field can be set to "gRPC".

### 4.4 Verification & Testing
- **Unit Tests**: Mock the gRPC server and test parsing/execution logic.
- **Integration Tests**: Spin up a small gRPC server (e.g., Greeter) in the test suite and run a `.http` file against it.

## 5. Migration Strategy
No breaking changes. Existing REST requests will continue to function as before.

## 6. Open Questions
- **TLS/SSL**: How to specify plaintext vs TLS?
    - *Proposal*: Use `GRPC` for plaintext (likely implies `h2c` or standard plaintext) and `GRPCS` (or `@tls` directive) for secure connections, similar to `http` vs `https`. Alternatively, check if the host has `https://` prefix, though gRPC standard format is often just `host:port`.
    - *Decision*: Default to plaintext for `localhost`, TLS otherwise, or use a specific `@grpc-tls` directive? For consistency, maybe `GRPC` is plain and `GRPCS` is TLS, or rely on `http://` vs `https://` prefix in the URL line: `GRPC https://host:port/...`.
