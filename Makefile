# Makefile for finance-forecast Go project

# Variables
BINARY_NAME := finance-forecast
TEST_DIR := test
BUILD_DIR := build
GO_MODULE := github.com/iwvelando/finance-forecast

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt

# Build flags
BUILD_FLAGS := -v
LDFLAGS := -ldflags "-X main.version=$(shell git describe --tags --always --dirty)"
TEST_FLAGS := -race
COVERAGE_FILE := $(TEST_DIR)/logs/coverage.out
COVERAGE_HTML := $(TEST_DIR)/logs/coverage.html

# Default target
.PHONY: all
all: clean build test

# Help target
.PHONY: help
help:
	@echo "Finance Forecast Build System"
	@echo "============================="
	@echo ""
	@echo "Available targets:"
	@echo "  all                  - Clean, build, and test (default)"
	@echo "  build                - Build the application"
	@echo "  test                 - Run unit and integration tests"
	@echo "  test-all             - Run all tests including performance"
	@echo "  test-unit            - Run unit tests only"
	@echo "  test-integration     - Run integration tests only"
	@echo "  test-performance     - Run performance tests only"
	@echo "  test-coverage        - Run tests with coverage report"
	@echo "  test-verbose         - Run all tests with verbose output"
	@echo "  clean                - Clean build artifacts and test logs"
	@echo "  clean-logs           - Clean test logs only"
	@echo "  deps                 - Download and verify dependencies"
	@echo "  fmt                  - Format Go source code"
	@echo "  vet                  - Run go vet"
	@echo "  lint                 - Run golangci-lint (if available)"
	@echo "  run                  - Build and run with example config"
	@echo "  run-pretty           - Build and run with pretty output"
	@echo "  install              - Install the binary to GOPATH/bin"
	@echo "  dev-setup            - Set up development environment"
	@echo "  check                - Run all quality checks"
	@echo "  pre-commit           - Run pre-commit checks"
	@echo "  status               - Show project status"
	@echo "  check-organization   - Verify project organization"

# Build targets
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/finance-forecast

.PHONY: build-all
build-all: build
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/finance-forecast
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/finance-forecast
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/finance-forecast

# Test targets
.PHONY: test
test: test-unit test-integration

.PHONY: test-all
test-all: test-unit test-integration test-performance
	@echo "All tests completed!"

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) $(TEST_FLAGS) ./internal/config ./internal/forecast

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) $(TEST_FLAGS) ./test/integration

.PHONY: test-performance
test-performance:
	@echo "Running performance benchmarks..."
	@mkdir -p $(TEST_DIR)/logs
	$(GOTEST) -v -run "^TestBasic|^TestPerformance|^TestMemory|^TestData" ./test/integration 2>&1 | tee $(TEST_DIR)/logs/benchmark_output.log

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(TEST_DIR)/logs
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

.PHONY: test-verbose
test-verbose:
	@echo "Running all tests with verbose output..."
	@mkdir -p $(TEST_DIR)/logs
	@echo "Testing config package..."
	$(GOTEST) -v ./internal/config 2>&1 | tee $(TEST_DIR)/logs/config_test_output.log
	@echo "Testing forecast package..."
	$(GOTEST) -v ./internal/forecast 2>&1 | tee $(TEST_DIR)/logs/forecast_test_output.log
	@echo "Running integration tests..."
	$(GOTEST) -v ./test/integration 2>&1 | tee $(TEST_DIR)/logs/integration_test_output.log

.PHONY: test-scripts
test-scripts:
	@echo "Running test scripts..."
	@chmod +x $(TEST_DIR)/scripts/run_tests.sh
	@chmod +x $(TEST_DIR)/scripts/run_all_tests.sh
	$(TEST_DIR)/scripts/run_tests.sh

# Quality targets
.PHONY: fmt
fmt:
	@echo "Formatting Go source code..."
	$(GOFMT) ./...

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping lint check"; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Dependency management
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify
	$(GOMOD) tidy

# Run targets
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME) with example configuration..."
	./$(BINARY_NAME) -config=./config.yaml.example -output-format=csv

.PHONY: run-pretty
run-pretty: build
	@echo "Running $(BINARY_NAME) with pretty output..."
	./$(BINARY_NAME) -config=./config.yaml.example -output-format=pretty

# Installation
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(LDFLAGS) .

# Cleanup targets
.PHONY: clean
clean: clean-logs
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)

.PHONY: clean-logs
clean-logs:
	@echo "Cleaning test logs..."
	@rm -f $(TEST_DIR)/logs/*.log
	@rm -f $(TEST_DIR)/logs/*.out
	@rm -f $(TEST_DIR)/logs/*.html

# Development targets
.PHONY: dev-setup
dev-setup: deps
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@echo "Development environment ready!"

.PHONY: check
check: fmt vet lint test
	@echo "All checks passed!"

# Git hooks
.PHONY: pre-commit
pre-commit: fmt vet test-unit
	@echo "Pre-commit checks passed!"

# Release targets
.PHONY: release-check
release-check: clean build-all test-coverage
	@echo "Release validation completed!"

# File organization targets
.PHONY: organize-tests
organize-tests:
	@echo "Test artifacts are already organized in $(TEST_DIR)/"
	@echo "Structure:"
	@echo "  $(TEST_DIR)/baseline/  - Test baseline files"
	@echo "  $(TEST_DIR)/docs/      - Test documentation"
	@echo "  $(TEST_DIR)/logs/      - Test output logs"
	@echo "  $(TEST_DIR)/scripts/   - Test execution scripts"

.PHONY: check-organization
check-organization:
	@echo "Running project organization check..."
	@chmod +x $(TEST_DIR)/scripts/check_organization.sh
	@$(TEST_DIR)/scripts/check_organization.sh

# Show project status
.PHONY: status
status:
	@echo "Finance Forecast Project Status"
	@echo "==============================="
	@echo "Go version: $(shell $(GOCMD) version)"
	@echo "Module: $(GO_MODULE)"
	@echo "Binary: $(BINARY_NAME)"
	@echo "Test directory: $(TEST_DIR)/"
	@echo ""
	@echo "Files:"
	@echo "  Source files: $(shell find . -name '*.go' -not -path './vendor/*' | wc -l | tr -d ' ')"
	@echo "  Test files: $(shell find . -name '*_test.go' -not -path './vendor/*' | wc -l | tr -d ' ')"
	@echo "  Dependencies: $(shell $(GOCMD) list -m all | wc -l | tr -d ' ')"
	@echo ""
	@if [ -f $(BINARY_NAME) ]; then \
		echo "Binary: $(BINARY_NAME) ($(shell ls -lh $(BINARY_NAME) | awk '{print $$5}'))"; \
	else \
		echo "Binary: Not built"; \
	fi
