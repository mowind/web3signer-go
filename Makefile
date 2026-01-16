# Makefile for web3signer-go
# Provides build, test, and deployment targets

# Binary name and version
BINARY_NAME=web3signer
VERSION?=$(shell git describe --tags --always --abbrev=0 2>/dev/null || echo "dev")
BUILD_TIME?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt
GOLINT=golangci-lint

# Build directories
BUILD_DIR=build
CMD_DIR=cmd/web3signer
MAIN_GO=$(CMD_DIR)/main.go

# Cross-compilation targets
LINUX_AMD64=linux-amd64
LINUX_ARM64=linux-arm64
DARWIN_AMD64=darwin-amd64
DARWIN_ARM64=darwin-arm64
WINDOWS_AMD64=windows-amd64
WINDOWS_ARM64=windows-arm64

.PHONY: all
all: build

.PHONY: build
build: fmt vet
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -v \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME) \
		$(MAIN_GO)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-$(LINUX_AMD64) $(MAIN_GO)
	GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-$(LINUX_ARM64) $(MAIN_GO)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-$(DARWIN_AMD64) $(MAIN_GO)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-$(DARWIN_ARM64) $(MAIN_GO)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-$(WINDOWS_AMD64) $(MAIN_GO)
	@echo "Cross-platform builds complete"

.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Format complete"

.PHONY: vet
vet:
	@echo "Vetting code..."
	$(GOVET) ./...
	@echo "Vet complete"

.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Tests complete. Coverage: coverage.out"

.PHONY: test-short
test-short:
	@echo "Running short tests..."
	$(GOTEST) -short -v ./...

.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

.PHONY: lint
lint:
	@if command -v $(GOLINT) >/dev/null; then \
		echo "Linting code..."; \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not installed. Skipping lint."; \
	fi

.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $$(GOPATH)/bin/$(BINARY_NAME)
	@echo "Install complete"

.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME) --help

.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOCMD) mod download
	$(GOCMD) mod tidy
	@echo "Dependencies complete"

.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go version: $$($(GOCMD) version)"

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make all          - Build the project"
	@echo "  make build        - Build the binary"
	@echo "  make build-all    - Build for all platforms"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make fmt          - Format the code"
	@echo "  make vet          - Run go vet"
	@echo "  make test         - Run tests"
	@echo "  make test-short   - Run short tests"
	@echo "  make benchmark    - Run benchmarks"
	@echo "  make lint         - Run linter"
	@echo "  make install       - Install the binary"
	@echo "  make run          - Run the binary"
	@echo "  make deps          - Download dependencies"
	@echo "  make version       - Show version information"
	@echo "  make help         - Show this help message"