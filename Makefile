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

.PHONY: all build build-dev clean test coverage bench lint lint-fast lint-fix fmt vet deps deps-update help install uninstall release dist run run-tui run-list run-help docker-build docker-run security security-gosec security-nancy security-govulncheck security-semgrep security-license security-sbom security-container security-audit docs install-linters install-security-tools install-dev-tools init check-updates validate validate-fast validate-full pre-commit quality-gate pre-release dev ci ci-full

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
	GO111MODULE=on $(GOTEST) -v -coverprofile=$(BUILD_DIR)/coverage.out ./...
	GO111MODULE=on $(GOCMD) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report generated at $(BUILD_DIR)/coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	GO111MODULE=on $(GOTEST) -bench=. -benchmem ./...

# Comprehensive linting
lint:
	@echo "Running comprehensive linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --no-config --enable=govet,errcheck,staticcheck,unused,ineffassign,unconvert,misspell --timeout=5m ./...; \
	else \
		echo "golangci-lint not found, install with: make install-linters"; \
		exit 1; \
	fi

# Fast linting (subset of checks)
lint-fast:
	@echo "Running fast linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --no-config --enable=govet,errcheck,staticcheck --timeout=2m ./...; \
	else \
		echo "golangci-lint not found, install with: make install-linters"; \
		exit 1; \
	fi

# Lint with fixes
lint-fix:
	@echo "Running linters with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --no-config --enable=unconvert --fix ./...; \
	else \
		echo "golangci-lint not found, install with: make install-linters"; \
		exit 1; \
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

# Comprehensive security analysis
security:
	@echo "Running comprehensive security analysis..."
	@$(MAKE) security-gosec
	@$(MAKE) security-nancy
	@$(MAKE) security-govulncheck
	@$(MAKE) security-semgrep
	@echo "Security analysis complete"

# gosec - Go security checker
security-gosec:
	@echo "Running gosec security analysis..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -fmt json -out $(BUILD_DIR)/gosec-report.json -stdout -verbose ./...; \
	else \
		echo "gosec not found, install with: make install-security-tools"; \
	fi

# nancy - Vulnerability scanner for Go dependencies
security-nancy:
	@echo "Running nancy dependency vulnerability scan..."
	@if command -v nancy >/dev/null 2>&1; then \
		go list -json -deps ./... | nancy sleuth --output json > $(BUILD_DIR)/nancy-report.json || true; \
	else \
		echo "nancy not found, install with: make install-security-tools"; \
	fi

# govulncheck - Official Go vulnerability scanner
security-govulncheck:
	@echo "Running govulncheck vulnerability scan..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck -json ./... > $(BUILD_DIR)/govulncheck-report.json 2>&1 || true; \
	else \
		echo "govulncheck not found, install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

# semgrep - Static analysis security scanner
security-semgrep:
	@echo "Running semgrep security analysis..."
	@if command -v semgrep >/dev/null 2>&1; then \
		semgrep --config=auto --json --output=$(BUILD_DIR)/semgrep-report.json . || true; \
	else \
		echo "semgrep not found, install with: pip install semgrep"; \
	fi

# License compliance check
security-license:
	@echo "Running license compliance check..."
	@if command -v go-licenses >/dev/null 2>&1; then \
		go-licenses report ./... --template $(BUILD_DIR)/licenses.tpl > $(BUILD_DIR)/licenses-report.txt; \
	else \
		echo "go-licenses not found, install with: go install github.com/google/go-licenses@latest"; \
	fi

# SBOM (Software Bill of Materials) generation
security-sbom:
	@echo "Generating Software Bill of Materials..."
	@if command -v syft >/dev/null 2>&1; then \
		syft packages dir:. -o json=$(BUILD_DIR)/sbom.json; \
		syft packages dir:. -o spdx-json=$(BUILD_DIR)/sbom.spdx.json; \
	else \
		echo "syft not found, install with: curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin"; \
	fi

# Container security scanning (if using Docker)
security-container:
	@echo "Running container security scan..."
	@if command -v grype >/dev/null 2>&1; then \
		grype $(BINARY_NAME):$(VERSION) -o json > $(BUILD_DIR)/grype-report.json; \
	else \
		echo "grype not found, install with: curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin"; \
	fi

# Full security audit
security-audit: security security-license security-sbom
	@echo "Full security audit complete"
	@echo "Reports generated in $(BUILD_DIR)/"

# Generate documentation
docs:
	@echo "Generating documentation..."
	@mkdir -p docs
	@$(BUILD_DIR)/$(BINARY_NAME) --help > docs/help.txt 2>&1 || true
	@echo "Documentation generated in docs/"

# Install linting tools
install-linters:
	@echo "Installing linting tools..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2; \
	else \
		echo "golangci-lint already installed"; \
	fi
	@if ! command -v gofumpt >/dev/null 2>&1; then \
		echo "Installing gofumpt..."; \
		GO111MODULE=on go install mvdan.cc/gofumpt@latest; \
	else \
		echo "gofumpt already installed"; \
	fi
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		GO111MODULE=on go install golang.org/x/tools/cmd/goimports@latest; \
	else \
		echo "goimports already installed"; \
	fi

# Install security tools
install-security-tools:
	@echo "Installing security tools..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		GO111MODULE=on go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	else \
		echo "gosec already installed"; \
	fi
	@if ! command -v nancy >/dev/null 2>&1; then \
		echo "Installing nancy..."; \
		GO111MODULE=on go install github.com/sonatypecommunity/nancy@latest; \
	else \
		echo "nancy already installed"; \
	fi
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		GO111MODULE=on go install golang.org/x/vuln/cmd/govulncheck@latest; \
	else \
		echo "govulncheck already installed"; \
	fi
	@if ! command -v go-licenses >/dev/null 2>&1; then \
		echo "Installing go-licenses..."; \
		GO111MODULE=on go install github.com/google/go-licenses@latest; \
	else \
		echo "go-licenses already installed"; \
	fi
	@echo "Note: semgrep requires Python: pip install semgrep"
	@echo "Note: syft and grype can be installed with:"
	@echo "  curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin"
	@echo "  curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin"

# Install all development tools
install-dev-tools: install-linters install-security-tools
	@echo "All development tools installed"

# Initialize project (install dev dependencies)
init: deps install-dev-tools
	@echo "Development environment initialized"
	@echo ""
	@echo "Available tools:"
	@echo "  Linting: golangci-lint, gofumpt, goimports"
	@echo "  Security: gosec, nancy, govulncheck, go-licenses"
	@echo "  Optional: semgrep (Python), syft, grype"
	@echo ""
	@echo "Run 'make validate' to run full validation pipeline"

# Check for updates
check-updates:
	@echo "Checking for dependency updates..."
	@GO111MODULE=on $(GOCMD) list -u -m all

# Validate the project (comprehensive)
validate: clean fmt vet lint test coverage security-gosec
	@echo "Project validation complete"

# Fast validation (for development)
validate-fast: fmt vet lint-fast test
	@echo "Fast validation complete"

# Pre-commit validation
pre-commit: fmt vet lint-fast
	@echo "Pre-commit validation complete"

# Quality gate for CI/CD
quality-gate:
	@echo "Running quality gate checks..."
	@mkdir -p $(BUILD_DIR)
	@$(MAKE) test
	@$(MAKE) lint
	@$(MAKE) security-gosec
	@$(MAKE) coverage
	@echo "Quality gate passed"

# Full validation with security
validate-full: clean fmt vet lint test coverage security
	@echo "Full validation with security complete"

# Quick development cycle
dev: clean fmt vet build-dev
	@echo "Development build complete"

# Pre-release validation
pre-release: clean validate-full release
	@echo "Pre-release validation complete"

# CI/CD pipeline simulation
ci: deps quality-gate build
	@echo "CI pipeline complete"

# Enhanced CI/CD with security
ci-full: deps validate-full security-audit build dist
	@echo "Full CI pipeline with security complete"

# Show help
help:
	@echo "Hosts Manager Makefile"
	@echo ""
	@echo "🏗️  Build Targets:"
	@echo "  build         Build the binary"
	@echo "  build-dev     Build with debug info and race detection"
	@echo "  clean         Clean build artifacts"
	@echo "  release       Build for all platforms"
	@echo "  dist          Create distribution packages"
	@echo ""
	@echo "🧪 Testing Targets:"
	@echo "  test          Run tests"
	@echo "  coverage      Run tests with coverage report"
	@echo "  bench         Run benchmarks"
	@echo ""
	@echo "🔍 Linting Targets:"
	@echo "  lint          Run comprehensive linters"
	@echo "  lint-fast     Run fast linters (subset)"
	@echo "  lint-fix      Run linters with auto-fix"
	@echo "  fmt           Format code"
	@echo "  vet           Vet code"
	@echo ""
	@echo "🔒 Security Targets:"
	@echo "  security           Run all security checks"
	@echo "  security-gosec     Run gosec security analysis"
	@echo "  security-nancy     Run nancy vulnerability scan"
	@echo "  security-govulncheck Run official Go vulnerability check"
	@echo "  security-semgrep   Run semgrep static analysis"
	@echo "  security-license   Check license compliance"
	@echo "  security-sbom      Generate Software Bill of Materials"
	@echo "  security-container Container security scan"
	@echo "  security-audit     Full security audit"
	@echo ""
	@echo "🛠️  Development Tools:"
	@echo "  install-linters       Install linting tools"
	@echo "  install-security-tools Install security tools"
	@echo "  install-dev-tools     Install all development tools"
	@echo "  init                  Initialize development environment"
	@echo ""
	@echo "📦 Dependencies:"
	@echo "  deps         Download dependencies"
	@echo "  deps-update  Update dependencies"
	@echo "  check-updates Check for dependency updates"
	@echo ""
	@echo "✅ Validation Targets:"
	@echo "  validate      Comprehensive validation"
	@echo "  validate-fast Fast validation (development)"
	@echo "  validate-full Full validation with security"
	@echo "  pre-commit    Pre-commit validation"
	@echo "  quality-gate  Quality gate for CI/CD"
	@echo "  pre-release   Pre-release validation"
	@echo ""
	@echo "🚀 CI/CD Targets:"
	@echo "  ci            Standard CI pipeline"
	@echo "  ci-full       Full CI pipeline with security"
	@echo "  dev           Quick development build"
	@echo ""
	@echo "💻 Installation:"
	@echo "  install       Install binary locally"
	@echo "  uninstall     Uninstall binary"
	@echo ""
	@echo "🏃 Execution:"
	@echo "  run           Run the application"
	@echo "  run-tui       Run in TUI mode"
	@echo "  run-list      List hosts entries"
	@echo "  run-help      Show application help"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  docker-build  Build Docker image"
	@echo "  docker-run    Run Docker container"
	@echo ""
	@echo "📚 Documentation:"
	@echo "  docs          Generate documentation"
	@echo "  help          Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION       Set version (default: git describe or 'dev')"
	@echo ""
	@echo "Examples:"
	@echo "  make init                    # Initialize development environment"
	@echo "  make validate               # Full validation"
	@echo "  make validate-fast          # Quick validation for development"
	@echo "  make security               # Run security checks"
	@echo "  make ci                     # Simulate CI pipeline"
	@echo "  make VERSION=v1.0.0 release # Build release"