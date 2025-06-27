# Otsu Obliterator Makefile
# Modern Go project automation following best practices

# Configuration
BINARY_NAME := otsu-obliterator
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := build
CMD_DIR := cmd/$(BINARY_NAME)

# Build flags
LDFLAGS := -s -w -X main.version=$(VERSION)
BUILD_TAGS := matprofile

# Auto-detect platform
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Colors for output
GREEN := \033[0;32m
BLUE := \033[0;34m  
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m

# Default target
.DEFAULT_GOAL := help

# Quality targets
.PHONY: all clean deps format lint test bench audit

## Development workflow
all: clean deps format lint test build ## Complete build pipeline

dev: format lint test run ## Quick development cycle

## Building
build: deps auto-clean ## Build for current platform (use build.sh for cross-platform)
	@echo "$(BLUE)Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -tags $(BUILD_TAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "$(GREEN)✓ Built: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-all: ## Build for all platforms using build.sh
	@./build.sh build all

debug: deps ## Build debug version with race detection
	@echo "$(BLUE)Building debug version$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -tags $(BUILD_TAGS) -race -gcflags=all=-N\ -l -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-debug ./$(CMD_DIR)
	@echo "$(GREEN)✓ Built debug: $(BUILD_DIR)/$(BINARY_NAME)-debug$(NC)"

## Running
run: build ## Build and run application
	@echo "$(BLUE)Running $(BINARY_NAME)$(NC)"
	@./$(BUILD_DIR)/$(BINARY_NAME)

run-debug: debug ## Run with debug logging
	@echo "$(BLUE)Running with debug logging$(NC)"
	@LOG_LEVEL=debug ./$(BUILD_DIR)/$(BINARY_NAME)-debug

run-memory: ## Run with memory debugging
	@echo "$(BLUE)Running with memory debugging$(NC)"
	@./build.sh debug memory

## Testing and Quality
test: deps ## Run tests with coverage
	@echo "$(BLUE)Running tests$(NC)"
	@mkdir -p coverage
	@go test -tags $(BUILD_TAGS) -race -coverprofile=coverage/coverage.out -covermode=atomic -v ./...
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "$(GREEN)✓ Tests passed. Coverage report: coverage/coverage.html$(NC)"

test-short: deps ## Run short tests only
	@echo "$(BLUE)Running short tests$(NC)"
	@go test -tags $(BUILD_TAGS) -short ./...

bench: deps ## Run benchmarks
	@echo "$(BLUE)Running benchmarks$(NC)"
	@go test -tags $(BUILD_TAGS) -bench=. -benchmem ./...

## Code Quality
format: ## Format code and organize imports
	@echo "$(BLUE)Formatting code$(NC)"
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "$(YELLOW)⚠ goimports not found. Install: go install golang.org/x/tools/cmd/goimports@latest$(NC)"; \
	fi
	@echo "$(GREEN)✓ Code formatted$(NC)"

lint: deps ## Run static analysis
	@echo "$(BLUE)Running linters$(NC)"
	@go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "$(YELLOW)⚠ staticcheck not found. Install: go install honnef.co/go/tools/cmd/staticcheck@latest$(NC)"; \
	fi
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)⚠ golangci-lint not found. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi
	@echo "$(GREEN)✓ Linting completed$(NC)"

audit: format lint test ## Complete quality control audit
	@echo "$(BLUE)Running quality audit$(NC)"
	@go mod verify
	@go mod tidy -diff
	@echo "$(GREEN)✓ Quality audit completed$(NC)"

## Dependencies and Cleanup
deps: ## Install and verify dependencies
	@echo "$(BLUE)Installing dependencies$(NC)"
	@go mod download
	@go mod verify
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

deps-tools: ## Install development tools
	@echo "$(BLUE)Installing development tools$(NC)"
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)✓ Development tools installed$(NC)"

clean: ## Remove build artifacts and clean cache
	@echo "$(BLUE)Cleaning build artifacts$(NC)"
	@rm -rf $(BUILD_DIR) coverage
	@go clean -cache -testcache
	@find . -name "*.prof" -delete 2>/dev/null || true
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

auto-clean: ## Auto-clean obsolete builds on version changes
	@if [ -f "$(BUILD_DIR)/.version" ] && [ "$$(cat $(BUILD_DIR)/.version)" != "$(VERSION)" ]; then \
		echo "$(BLUE)Version changed, cleaning obsolete builds$(NC)"; \
		$(MAKE) clean; \
	fi
	@mkdir -p $(BUILD_DIR)
	@echo "$(VERSION)" > $(BUILD_DIR)/.version

## Cross-platform builds (delegates to build.sh)
build-windows: ## Cross-compile for Windows
	@./build.sh build windows

build-macos: ## Cross-compile for macOS Intel  
	@./build.sh build macos

build-macos-arm64: ## Cross-compile for macOS Apple Silicon
	@./build.sh build macos-arm64

build-linux: ## Cross-compile for Linux
	@./build.sh build linux

## Platform-specific run targets
run-windows: build-windows ## Run Windows binary (if on Windows/WSL)
	@if [ "$(GOOS)" = "windows" ] || command -v cmd.exe >/dev/null 2>&1; then \
		./$(BUILD_DIR)/$(BINARY_NAME).exe; \
	else \
		echo "$(RED)✗ Cannot run Windows binary on $(GOOS)$(NC)"; \
	fi

## Deployment and Distribution
package: ## Create distribution package using build.sh
	@./build.sh package

install: build ## Install binary to local system
	@echo "$(BLUE)Installing $(BINARY_NAME)$(NC)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME) 2>/dev/null || \
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME) 2>/dev/null || \
	echo "$(YELLOW)⚠ Could not install to system PATH. Add $(BUILD_DIR) to PATH or copy manually$(NC)"
	@echo "$(GREEN)✓ Installation completed$(NC)"

uninstall: ## Remove installed binary
	@echo "$(BLUE)Uninstalling $(BINARY_NAME)$(NC)"
	@rm -f $(GOPATH)/bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)✓ Uninstallation completed$(NC)"

## Debugging and Diagnostics
check-deps: ## Verify all dependencies and tools
	@echo "$(BLUE)Checking dependencies$(NC)"
	@./build.sh deps
	@echo "$(GREEN)✓ Dependency check completed$(NC)"

debug-build: ## Show build information and environment
	@echo "$(BLUE)Build Information$(NC)"
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Dir: $(BUILD_DIR)"
	@echo "Command Dir: $(CMD_DIR)"
	@echo "Go Version: $$(go version)"
	@echo "GOOS/GOARCH: $(GOOS)/$(GOARCH)"
	@echo "Build Tags: $(BUILD_TAGS)"
	@echo "LDFLAGS: $(LDFLAGS)"

debug-menu: build ## Test menu functionality specifically
	@echo "$(BLUE)Testing menu functionality$(NC)"
	@echo "$(YELLOW)Look for 'MAIN: Starting main function' log to verify proper entry point$(NC)"
	@LOG_LEVEL=debug ./$(BUILD_DIR)/$(BINARY_NAME)

## CI/CD targets
ci: format lint test build ## CI pipeline target
	@echo "$(GREEN)✓ CI pipeline completed$(NC)"

## Help system
help: ## Show this help message
	@echo "$(BLUE)Otsu Obliterator - Make Targets$(NC)"
	@echo ""
	@echo "$(YELLOW)CRITICAL:$(NC) This project uses 'go build' (not 'fyne build') to preserve"
	@echo "proper main() entry points for menu functionality."
	@echo ""
	@echo "$(BLUE)Usage:$(NC) make [target]"
	@echo ""
	@echo "$(BLUE)Quick Start:$(NC)"
	@echo "  make deps     # Install dependencies"
	@echo "  make run      # Build and run"
	@echo "  make test     # Run tests"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "$(BLUE)Targets:$(NC)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(BLUE)Cross-platform building:$(NC)"
	@echo "  ./build.sh build all     # All platforms"
	@echo "  ./build.sh debug memory  # Memory debugging"
	@echo ""
	@echo "$(BLUE)Troubleshooting:$(NC)"
	@echo "  make debug-menu          # Test menu functionality"
	@echo "  make check-deps          # Verify dependencies"
	@echo "  make clean && make build # Clean rebuild"

# Version info
version: ## Show version information
	@echo "$(BLUE)Version Information$(NC)"
	@echo "$(BINARY_NAME) version $(VERSION)"
	@if [ -f "$(BUILD_DIR)/$(BINARY_NAME)" ]; then \
		echo "Binary size: $$(du -h $(BUILD_DIR)/$(BINARY_NAME) | cut -f1)"; \
		echo "Binary path: $(BUILD_DIR)/$(BINARY_NAME)"; \
	else \
		echo "$(YELLOW)Binary not built yet$(NC)"; \
	fi