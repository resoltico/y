# Otsu Obliterator Makefile - Modular Architecture
.PHONY: all build run clean deps test help
.DEFAULT_GOAL := help

BINARY_NAME=otsu-obliterator
VERSION=1.0.0
BUILD_DIR=build
CMD_DIR=cmd/otsu-obliterator

# Build flags
LDFLAGS=-s -w -X main.version=$(VERSION)
BUILD_TAGS=matprofile

all: build

deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Dependencies installed"

build:
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-profile:
	@echo "Building $(BINARY_NAME) with profiling..."
	go build -tags $(BUILD_TAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

run-profile: build-profile
	@echo "Running $(BINARY_NAME) with profiling..."
	@echo "Memory profiler: http://localhost:6060/debug/pprof/"
	@echo "Mat profiling: http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat"
	./$(BUILD_DIR)/$(BINARY_NAME)

run-debug-safe:
	@echo "Running with safe debugging (no pixel analysis)..."
	OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_IMAGE=true OTSU_DEBUG_FORMAT=true go run -tags $(BUILD_TAGS) ./$(CMD_DIR)

run-debug-all:
	@echo "Running with all debugging enabled..."
	OTSU_DEBUG_FORMAT=true OTSU_DEBUG_IMAGE=true OTSU_DEBUG_MEMORY=true OTSU_DEBUG_PERFORMANCE=true OTSU_DEBUG_GUI=true OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_TRICLASS=true go run -tags $(BUILD_TAGS) ./$(CMD_DIR)

test:
	@echo "Running tests..."
	go test -tags $(BUILD_TAGS) ./...

test-race:
	@echo "Running tests with race detector..."
	go test -race -tags $(BUILD_TAGS) ./...

benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -tags $(BUILD_TAGS) ./...

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe $(BINARY_NAME)-*
	@echo "Clean complete"

# Cross-compilation targets
build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -tags $(BUILD_TAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME).exe ./$(CMD_DIR)

build-macos:
	@echo "Building for macOS (Intel)..."
	GOOS=darwin GOARCH=amd64 go build -tags $(BUILD_TAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-macos-amd64 ./$(CMD_DIR)

build-macos-arm64:
	@echo "Building for macOS (Apple Silicon)..."
	GOOS=darwin GOARCH=arm64 go build -tags $(BUILD_TAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-macos-arm64 ./$(CMD_DIR)

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -tags $(BUILD_TAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)

build-all: build-windows build-macos build-macos-arm64 build-linux

# Development helpers
dev: deps build-profile
	@echo "Development environment ready. Run 'make run-profile' to start."

check-memory:
	@echo "Checking for memory leaks..."
	go run -tags $(BUILD_TAGS) ./$(CMD_DIR) &
	@echo "Application started. Check http://localhost:6060/debug/pprof/heap for memory usage"

profile:
	@echo "Starting profiling server..."
	go tool pprof http://localhost:6060/debug/pprof/heap

lint:
	@echo "Running linter..."
	golangci-lint run

format:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

mod-verify:
	@echo "Verifying modules..."
	go mod verify

security:
	@echo "Running security scan..."
	gosec ./...

help:
	@echo "Otsu Obliterator - Modular Architecture"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build Targets:"
	@echo "  build           Build production binary"
	@echo "  build-profile   Build with Mat profiling"
	@echo "  build-all       Build for all platforms"
	@echo ""
	@echo "Run Targets:"
	@echo "  run             Run production build"
	@echo "  run-profile     Run with profiling server"
	@echo "  run-debug-safe  Run with safe debugging"
	@echo "  run-debug-all   Run with all debugging"
	@echo ""
	@echo "Test Targets:"
	@echo "  test            Run tests"
	@echo "  test-race       Run tests with race detector"
	@echo "  benchmark       Run benchmarks"
	@echo ""
	@echo "Development:"
	@echo "  dev             Setup development environment"
	@echo "  deps            Install dependencies"
	@echo "  clean           Clean build artifacts"
	@echo "  lint            Run linter"
	@echo "  format          Format code"
	@echo "  vet             Run go vet"
	@echo ""
	@echo "Cross-Platform:"
	@echo "  build-windows   Build for Windows"
	@echo "  build-macos     Build for macOS Intel"
	@echo "  build-macos-arm64 Build for macOS Apple Silicon"
	@echo "  build-linux     Build for Linux"
	@echo ""
	@echo "Profiling:"
	@echo "  check-memory    Start app and show memory profiling info"
	@echo "  profile         Connect to profiling server"