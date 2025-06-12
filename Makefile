SHELL := /bin/bash

.PHONY: all build clean fmt lint test test-unit check help

all: check

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Build
build: fmt lint ## Build the Go application (not typical for a library)
	@echo "Building... (Note: Libraries are typically not "built" into a binary like this)"
	@go build ./...

# Format, Lint, Test
fmt: ## Format Go source files
	@go fmt ./...

lint: ## Lint Go source files using golangci-lint
	@echo "Linting..."
	@golangci-lint run ./...

check: lint test-unit ## Run all pre-commit checks (lint, unit tests)
	@echo "Checks completed."

test: test-unit ## Run all tests

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	@gotestsum --junitfile unit-tests.xml ./... -- -cover -v
	@echo "Check unit-tests.xml for results."

# Dependencies
install-lint: ## Install golangci-lint
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6

install-gotestsum: ## Install gotestsum
	@echo "Installing gotestsum..."
	@go install gotest.tools/gotestsum@latest

# Clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@go clean -cache -testcache -modcache

# Go Mod
tidy: ## Tidy go.mod file
	@go mod tidy

deps: tidy ## Install/update dependencies
	@go get -u ./...

.PHONY: tidy deps install-lint install-gotestsum
