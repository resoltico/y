# Otsu Obliterator Makefile
.PHONY: build run clean test profile build-profile run-profile check-leaks deps check-deps

# Binary name
BINARY_NAME=otsu-obliterator

# MAIN TARGETS
build:
	@echo "Building production binary..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) .

build-profile:
	@echo "Building with Mat profiling and memory tracking..."
	go build -tags matprofile -ldflags="-s -w" -o $(BINARY_NAME) .

run:
	@echo "Running production build..."
	go run .

run-profile:
	@echo "Running with Mat profiling enabled..."
	@echo "Monitor terminal output for Mat creation/cleanup tracking"
	@echo "Memory profiler available at: http://localhost:6060/debug/pprof/"
	@echo "Mat-specific profiling at: http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat"
	@echo "Format detection debug logs will show image loading details"
	go run -tags matprofile .

# Debug target with format detection focus
run-debug-format:
	@echo "Running with focused format detection debugging..."
	@echo "This will show detailed format detection and URI analysis"
	@echo "Watch for FormatDebug and ImageDebug log entries"
	OTSU_DEBUG_FORMAT=true go run -tags matprofile .

# Debug targets for specific components
run-debug-gui:
	@echo "Running with GUI debugging enabled..."
	OTSU_DEBUG_GUI=true go run -tags matprofile .

run-debug-algorithms:
	@echo "Running with algorithm debugging enabled..."
	OTSU_DEBUG_ALGORITHMS=true go run -tags matprofile .

run-debug-performance:
	@echo "Running with performance debugging enabled..."
	OTSU_DEBUG_PERFORMANCE=true go run -tags matprofile .

run-debug-memory:
	@echo "Running with memory debugging enabled..."
	OTSU_DEBUG_MEMORY=true go run -tags matprofile .

run-debug-image:
	@echo "Running with image processing debugging enabled..."
	OTSU_DEBUG_IMAGE=true go run -tags matprofile .

run-debug-triclass:
	@echo "Running with Iterative Triclass algorithm debugging enabled..."
	@echo "This will show detailed triclass processing steps and pixel analysis"
	OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_PIXELS=true go run -tags matprofile .

run-debug-pixels:
	@echo "Running with pixel-level analysis debugging enabled..."
	@echo "This will show detailed pixel sampling and analysis"
	OTSU_DEBUG_PIXELS=true go run -tags matprofile .

run-debug-comprehensive:
	@echo "Running with comprehensive debugging for complex issues..."
	@echo "Enables triclass, pixel analysis, image conversion, and format debugging"
	OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_PIXELS=true OTSU_DEBUG_IMAGE=true OTSU_DEBUG_FORMAT=true go run -tags matprofile .

run-debug-all:
	@echo "Running with ALL debugging enabled..."
	OTSU_DEBUG_FORMAT=true OTSU_DEBUG_IMAGE=true OTSU_DEBUG_MEMORY=true OTSU_DEBUG_PERFORMANCE=true OTSU_DEBUG_GUI=true OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_PIXELS=true go run -tags matprofile .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -f $(BINARY_NAME)-*

# Dependency checking
deps:
	@echo "Installing Go dependencies..."
	go mod tidy
	go mod download

# Dependency verification
check-deps:
	@echo "Checking system dependencies..."
	@echo "Checking Go version..."
	@go version || (echo "ERROR: Go not found. Please install Go 1.24+"; exit 1)
	@echo "Checking OpenCV..."
	@pkg-config --exists opencv4 || pkg-config --exists opencv || (echo "ERROR: OpenCV not found. Please install OpenCV 4.11.0+"; exit 1)
	@echo "Checking OpenCV version..."
	@pkg-config --modversion opencv4 2>/dev/null || pkg-config --modversion opencv 2>/dev/null || echo "Could not determine OpenCV version"
	@echo "Checking Fyne dependencies..."
	@go list -m fyne.io/fyne/v2 > /dev/null || (echo "ERROR: Fyne not found. Run 'make deps'"; exit 1)
	@echo "All dependencies OK!"

# Test with profiling enabled
test:
	@echo "Running tests with Mat profiling..."
	go test -tags matprofile ./...

# Memory leak detection
check-leaks:
	@echo "Running memory leak detection..."
	@echo "Building with profiling..."
	@make build-profile
	@echo ""
	@echo "Starting application with memory leak monitoring..."
	@echo "- Watch terminal output for MatProfile count changes"
	@echo "- Initial count should be 0"
	@echo "- Final count should return to 0"
	@echo "- Any non-zero final count indicates memory leaks"
	@echo ""
	@echo "Memory profiler endpoints:"
	@echo "- Main pprof: http://localhost:6060/debug/pprof/"
	@echo "- Mat profile: http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat"
	@echo ""
	@echo "Press Ctrl+C to stop and see final memory report..."
	@./$(BINARY_NAME)

# Profiling with pprof server
profile: build-profile
	@echo "Starting application with full profiling enabled..."
	@echo "Available profiling endpoints:"
	@echo "- CPU profile: http://localhost:6060/debug/pprof/profile"
	@echo "- Memory profile: http://localhost:6060/debug/pprof/heap"
	@echo "- Goroutine profile: http://localhost:6060/debug/pprof/goroutine"
	@echo "- Mat profile: http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat"
	@echo ""
	@echo "Example usage:"
	@echo "  go tool pprof http://localhost:6060/debug/pprof/heap"
	@echo "  go tool pprof http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat"
	@echo ""
	@./$(BINARY_NAME)

# Get current MatProfile count (requires running application)
profile-count:
	@echo "Fetching current MatProfile count..."
	@curl -s http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat?debug=1 | head -1 || echo "Application not running or profiling not enabled"

# Cross-platform builds with profiling enabled by default
build-windows:
	@echo "Building for Windows with profiling..."
	GOOS=windows GOARCH=amd64 go build -tags matprofile -ldflags="-s -w" -o $(BINARY_NAME).exe .

build-macos:
	@echo "Building for macOS (Intel) with profiling..."
	GOOS=darwin GOARCH=amd64 go build -tags matprofile -ldflags="-s -w" -o $(BINARY_NAME)-macos-amd64 .

build-macos-arm64:
	@echo "Building for macOS (Apple Silicon) with profiling..."
	GOOS=darwin GOARCH=arm64 go build -tags matprofile -ldflags="-s -w" -o $(BINARY_NAME)-macos-arm64 .

build-linux:
	@echo "Building for Linux with profiling..."
	GOOS=linux GOARCH=amd64 go build -tags matprofile -ldflags="-s -w" -o $(BINARY_NAME)-linux-amd64 .

# Universal binary for macOS with profiling
build-macos-universal:
	@echo "Building universal binary for macOS with profiling..."
	@echo "Building ARM64 binary..."
	GOOS=darwin GOARCH=arm64 go build -tags matprofile -ldflags="-s -w" -o $(BINARY_NAME)-arm64 .
	@echo "Building x86_64 binary..."
	GOOS=darwin GOARCH=amd64 go build -tags matprofile -ldflags="-s -w" -o $(BINARY_NAME)-x86_64 .
	@echo "Creating universal binary..."
	lipo -create -output $(BINARY_NAME)-macos-universal $(BINARY_NAME)-arm64 $(BINARY_NAME)-x86_64
	@echo "Cleaning up individual architecture binaries..."
	rm -f $(BINARY_NAME)-arm64 $(BINARY_NAME)-x86_64
	@echo "Universal binary created: $(BINARY_NAME)-macos-universal"

# Create macOS app bundle (requires fyne command)
build-macos-app:
	@echo "Creating macOS app bundle..."
	@command -v fyne >/dev/null 2>&1 || (echo "ERROR: fyne command not found. Install with: go install fyne.io/fyne/v2/cmd/fyne@latest"; exit 1)
	@make build-macos-universal
	@echo "Packaging app bundle..."
	fyne package -os darwin -executable $(BINARY_NAME)-macos-universal
	@echo "macOS app bundle created successfully!"

# Format code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@command -v golangci-lint >/dev/null 2>&1 || (echo "WARNING: golangci-lint not found. Install from: https://golangci-lint.run/usage/install/"; exit 0)
	golangci-lint run

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Development workflow target
dev: check-deps deps build-profile
	@echo "Development environment ready!"
	@echo "Run 'make run-profile' to start with memory profiling"
	@echo "Run 'make run-debug-format' for format detection debugging"
	@echo "Run 'make check-leaks' for memory leak detection"

# Production workflow target  
prod: check-deps deps test build
	@echo "Production build complete!"
	@echo "Binary: $(BINARY_NAME)"

# Help target
help:
	@echo "Otsu Obliterator - Available Targets:"
	@echo ""
	@echo "🔧 MAIN TARGETS:"
	@echo "  dev                  - Set up development environment"
	@echo "  build-profile        - Build with memory profiling enabled"
	@echo "  run-profile          - Run with memory profiling"
	@echo "  run-debug-format     - Run with format detection debugging"
	@echo "  check-leaks          - Run with memory leak detection"
	@echo "  profile              - Start with full profiling server"
	@echo "  prod                 - Full production build workflow"
	@echo ""
	@echo "🚀 PRODUCTION:"
	@echo "  build                - Build production binary"
	@echo "  run                  - Run production binary"
	@echo ""
	@echo "🐛 DEBUG TARGETS:"
	@echo "  run-debug-format     - Format detection and image loading"
	@echo "  run-debug-gui        - GUI events and interactions"
	@echo "  run-debug-algorithms - Algorithm execution and parameters"
	@echo "  run-debug-performance- Performance timing and metrics"
	@echo "  run-debug-memory     - Memory usage and tracking"
	@echo "  run-debug-image      - Image processing and conversion"
	@echo "  run-debug-triclass   - Iterative Triclass algorithm debugging"
	@echo "  run-debug-pixels     - Pixel-level analysis and sampling"
	@echo "  run-debug-comprehensive- Multiple debug categories for complex issues"
	@echo "  run-debug-all        - All debugging enabled"
	@echo ""
	@echo "🌍 CROSS-PLATFORM:"
	@echo "  build-windows        - Build for Windows"
	@echo "  build-macos          - Build for macOS (Intel)"
	@echo "  build-macos-arm64    - Build for macOS (Apple Silicon)"
	@echo "  build-macos-universal- Build universal macOS binary"
	@echo "  build-macos-app      - Create macOS app bundle"
	@echo "  build-linux          - Build for Linux"
	@echo ""
	@echo "🔧 MAINTENANCE:"
	@echo "  deps                 - Install dependencies"
	@echo "  check-deps           - Verify system dependencies"
	@echo "  test                 - Run tests"
	@echo "  clean                - Clean build artifacts"
	@echo "  fmt                  - Format code"
	@echo "  lint                 - Lint code"
	@echo "  vet                  - Vet code"
	@echo ""
	@echo "🚀 QUICK START:"
	@echo "  make dev && make run-debug-comprehensive"
	@echo ""
	@echo "📈 DEBUGGING:"
	@echo "  profile-count        - Get current MatProfile count"
	@echo "  run-debug-format     - Focus on image format detection issues"
	@echo "  run-debug-comprehensive- Focus on complex algorithm/display issues"

# Default target
.DEFAULT_GOAL := help