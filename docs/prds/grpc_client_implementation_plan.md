# gRPC Client Implementation Plan

This document outlines the phased implementation of gRPC client support in the `go-restclient` library.

## Phase 1: Foundations & Dependencies [DONE]
- [x] Add necessary go modules <!-- id: p1-1 -->
    - `google.golang.org/grpc`
    - `google.golang.org/protobuf`
    - `github.com/jhump/protoreflect`
- [x] Define `GRPC` constant and basic types <!-- id: p1-2 -->
    - Update `parser_types.go` or similar to explicitly recognize `GRPC` as a supported method.

## Phase 2: Parser Enhancements [DONE]
- [x] Relax URL validation for `GRPC` method <!-- id: p2-1 -->
    - Modify `parser_state.go` and `variables.go` to allow URLs without `http://` or `https://` schemes when the method is `GRPC` (e.g., `localhost:50051/package.Service/Method`).
- [x] Ensure `GRPC` requests map to the `Request` struct correctly <!-- id: p2-2 -->
    - Verify `RawURLString` captures the full target + method path.
    - Ensure headers are parsed correctly as metadata.

## Phase 3: Core Client Implementation [DONE]
- [x] Create `client_grpc.go` <!-- id: p3-1 -->
    - Implement `executeGRPCRequest(ctx context.Context, req *Request) (*Response, error)`.
- [x] Implement Connection Management <!-- id: p3-2 -->
    - Implement `grpc.Dial` logic.
- [x] Implement Dynamic Invocation <!-- id: p3-3 -->
    - Use `jhump/protoreflect` for dynamic method calling via reflection.

## Phase 4: Integration & Verification [DONE]
- [x] Add Unit Tests for `client_grpc.go` <!-- id: p4-1 -->
- [x] Create E2E Test Suite with `.http` files <!-- id: p4-2 -->
- [x] Verify Variable Substitution with environment files <!-- id: p4-3 -->
