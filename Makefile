BINARY  := hosts-manager
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
PREFIX  ?= /usr/local

# Tool versions. CI pins the same ones in .github/workflows/ci.yml; keep them
# in sync. golangci-lint must be v2 (.golangci.yml uses the v2 schema).
GOLANGCI_VERSION := v2.11.4
GOSEC_VERSION    := v2.28.0

.PHONY: build test lint sec check install uninstall tools snapshot clean

build: ## Build ./build/hosts-manager
	go build -trimpath -ldflags "$(LDFLAGS)" -o build/$(BINARY) ./cmd/hosts-manager

test: ## Run tests with the race detector
	go test -race ./...

lint: ## Run golangci-lint
	golangci-lint run --timeout=5m

sec: ## Run gosec and govulncheck
	gosec -quiet ./...
	govulncheck ./...

check: lint test sec ## Everything CI checks

install: build ## Install to $(PREFIX)/bin (use sudo)
	install -m 0755 build/$(BINARY) $(PREFIX)/bin/$(BINARY)

uninstall: ## Remove from $(PREFIX)/bin (use sudo)
	rm -f $(PREFIX)/bin/$(BINARY)

tools: ## Install the pinned lint and security tools
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@latest

snapshot: ## Build release archives locally into dist/ without publishing
	goreleaser release --snapshot --clean

clean:
	rm -rf build dist

help:
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'
