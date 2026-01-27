.PHONY: build clean test lint coverage help all check env version-info

# Version information
VERSION ?= v0.1.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Default target
all: build

# Build the application
build:
	@echo "Building web3signer $(VERSION)..."
	@mkdir -p build
	go build $(LDFLAGS) -o build/web3signer ./cmd/web3signer
	go build -o build/test-kms ./cmd/test-kms
	@echo "Build complete: build/web3signer, build/test-kms"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf build/
	@echo "Clean complete"

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test ./... -cover

# Run code quality checks
lint:
	$(GOPATH)/bin/golangci-lint run --timeout=5m

# Run tests and code quality checks
check: test-coverage lint

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Tidy dependencies
tidy:
	go mod tidy

# Run integration tests
integration-test:
	go test ./test -v

# Install development tools
install-tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest

# Generate coverage report
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Check development environment
env:
	@echo "Checking development environment..."
	@./scripts/check-env.sh

# Show version information
version-info:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# Help target
help:
	@echo "Available targets:"
	@echo "  build            - Build the application"
	@echo "  clean            - Clean build artifacts"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  lint             - Run code quality checks"
	@echo "  check            - Run tests and code quality checks"
	@echo "  fmt              - Format code"
	@echo "  vet              - Vet code"
	@echo "  tidy             - Tidy dependencies"
	@echo "  integration-test - Run integration tests"
	@echo "  install-tools    - Install development tools"
	@echo "  coverage         - Generate HTML coverage report"
	@echo "  env              - Check development environment"
	@echo "  version-info     - Show version information"
	@echo "  help             - Show this help message"
