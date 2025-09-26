# Hosts Manager Makefile
# Cross-platform hosts file manager

BINARY_NAME=hosts-manager
PACKAGE=hosts-manager
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOFMT=gofmt

# Build flags
LDFLAGS=-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commitHash=$(COMMIT_HASH)
BUILD_FLAGS=-ldflags "$(LDFLAGS)" -trimpath

# Directories
SRC_DIR=./cmd/hosts-manager
BUILD_DIR=build
DIST_DIR=dist

# Platforms for cross-compilation
PLATFORMS=windows/amd64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: all build clean test coverage lint fmt vet deps help install uninstall release dist

all: clean fmt vet test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	GO111MODULE=on $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)

# Build for development (with debug info)
build-dev:
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	GO111MODULE=on $(GOBUILD) -race -o $(BUILD_DIR)/$(BINARY_NAME)-dev $(SRC_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)

# Run tests
test:
	@echo "Running tests..."
	GO111MODULE=on $(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	GO111MODULE=on $(GOTEST) -v -race -coverprofile=$(BUILD_DIR)/coverage.out ./...
	GO111MODULE=on $(GOCMD) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report generated at $(BUILD_DIR)/coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	GO111MODULE=on $(GOTEST) -bench=. -benchmem ./...

# Lint code
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping..."; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	@$(GOFMT) -s -w .

# Vet code
vet:
	@echo "Vetting code..."
	GO111MODULE=on $(GOVET) ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	GO111MODULE=on $(GOMOD) download
	GO111MODULE=on $(GOMOD) tidy

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	GO111MODULE=on $(GOGET) -u ./...
	GO111MODULE=on $(GOMOD) tidy

# Install locally
install: build
	@echo "Installing $(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "$(BINARY_NAME) installed to /usr/local/bin/"

# Uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(BINARY_NAME) uninstalled"

# Cross-compile for all platforms
release: clean
	@echo "Building release binaries..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os_arch=$$(echo $$platform | tr '/' '-'); \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		echo "Building for $$os/$$arch..."; \
		output_name=$(BINARY_NAME)-$(VERSION)-$$os_arch; \
		if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
		env GOOS=$$os GOARCH=$$arch GO111MODULE=on $(GOBUILD) $(BUILD_FLAGS) -o $(DIST_DIR)/$$output_name $(SRC_DIR); \
	done
	@echo "Release binaries built in $(DIST_DIR)/"

# Create distribution packages
dist: release
	@echo "Creating distribution packages..."
	@cd $(DIST_DIR) && \
	for binary in $(BINARY_NAME)-$(VERSION)-*; do \
		if [[ $$binary == *".exe" ]]; then \
			zip $$binary.zip $$binary; \
		else \
			tar -czf $$binary.tar.gz $$binary; \
		fi; \
	done
	@echo "Distribution packages created in $(DIST_DIR)/"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

# Run with specific command
run-tui: build
	@echo "Running $(BINARY_NAME) in TUI mode..."
	@$(BUILD_DIR)/$(BINARY_NAME) tui

run-list: build
	@echo "Listing hosts entries..."
	@$(BUILD_DIR)/$(BINARY_NAME) list

run-help: build
	@echo "Showing help..."
	@$(BUILD_DIR)/$(BINARY_NAME) --help

# Docker support
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) .

docker-run: docker-build
	@echo "Running Docker container..."
	@docker run --rm -it $(BINARY_NAME):$(VERSION)

# Security check
security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found, skipping security check"; \
	fi

# Generate documentation
docs:
	@echo "Generating documentation..."
	@mkdir -p docs
	@$(BUILD_DIR)/$(BINARY_NAME) --help > docs/help.txt 2>&1 || true
	@echo "Documentation generated in docs/"

# Initialize project (install dev dependencies)
init:
	@echo "Initializing development environment..."
	@GO111MODULE=on $(GOMOD) download
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	fi
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		GO111MODULE=on $(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	@echo "Development environment initialized"

# Check for updates
check-updates:
	@echo "Checking for dependency updates..."
	@GO111MODULE=on $(GOCMD) list -u -m all

# Validate the project
validate: fmt vet lint test
	@echo "Project validation complete"

# Quick development cycle
dev: clean fmt vet build-dev
	@echo "Development build complete"

# CI/CD pipeline simulation
ci: deps fmt vet lint test coverage security build
	@echo "CI pipeline complete"

# Show help
help:
	@echo "Hosts Manager Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build the binary"
	@echo "  build-dev    Build with debug info"
	@echo "  clean        Clean build artifacts"
	@echo "  test         Run tests"
	@echo "  coverage     Run tests with coverage report"
	@echo "  bench        Run benchmarks"
	@echo "  lint         Run linters"
	@echo "  fmt          Format code"
	@echo "  vet          Vet code"
	@echo "  deps         Download dependencies"
	@echo "  deps-update  Update dependencies"
	@echo "  install      Install binary locally"
	@echo "  uninstall    Uninstall binary"
	@echo "  release      Build for all platforms"
	@echo "  dist         Create distribution packages"
	@echo "  run          Run the application"
	@echo "  run-tui      Run in TUI mode"
	@echo "  run-list     List hosts entries"
	@echo "  run-help     Show application help"
	@echo "  docker-build Build Docker image"
	@echo "  docker-run   Run Docker container"
	@echo "  security     Run security checks"
	@echo "  docs         Generate documentation"
	@echo "  init         Initialize development environment"
	@echo "  validate     Full project validation"
	@echo "  dev          Quick development build"
	@echo "  ci           Simulate CI pipeline"
	@echo "  help         Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION      Set version (default: git describe or 'dev')"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test"
	@echo "  make VERSION=v1.0.0 release"
	@echo "  make install"