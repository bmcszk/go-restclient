SHELL := /bin/bash

.PHONY: all build clean fmt lint test test-unit test-e2e check help

BINARY_NAME=go-restclient-lib-checker # Example, not really a binary for a lib

# Go parameters
GOBASE := $(shell pwd)
GOPATH := $(GOBASE)/vendor
GOBIN := $(GOBASE)/bin
GOFILES := $(wildcard *.go)

# Tools
GOLANGCI_LINT := $(GOBIN)/golangci-lint

all: check

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Build
build: fmt lint ## Build the Go application (not typical for a library)
	@echo "Building... (Note: Libraries are typically not "built" into a binary like this)"
	# @go build -o $(GOBIN)/$(BINARY_NAME) $(GOFILES)
	@go build ./...

# Format, Lint, Test
fmt: ## Format Go source files
	@go fmt ./...

lint: ## Lint Go source files using golangci-lint
	@echo "Linting..."
	@golangci-lint run ./...

check: build lint test-unit ## Run all pre-commit checks (build, lint, unit tests)
	@echo "Checks completed."

test: test-unit test-e2e ## Run all tests

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	@go test -tags=unit -cover ./...

test-e2e: ## Run end-to-end tests
	@echo "Running E2E tests..."
	@go test -tags=e2e ./e2e/...

# Dependencies
install-lint: ## Install golangci-lint
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@mv $(shell go env GOPATH)/bin/golangci-lint $(GOBIN)/

# Clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(GOBIN)/$(BINARY_NAME)
	@go clean -cache -testcache -modcache

# Go Mod
tidy: ## Tidy go.mod file
	@go mod tidy

deps: tidy ## Install/update dependencies
	@go get -u ./...

.PHONY: tidy deps install-lint 
