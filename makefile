# Otsu Obliterator Makefile - Updated for Modularized Algorithms
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

# Debug targets for modularized algorithms
run-debug-2d-otsu:
	@echo "Running with 2D Otsu algorithm debugging enabled..."
	@echo "This will show detailed 2D Otsu processing steps, histogram analysis, and threshold calculation"
	OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_IMAGE=true go run -tags matprofile .

run-debug-triclass:
	@echo "Running with Iterative Triclass algorithm debugging enabled..."
	@echo "This will show detailed triclass processing steps, convergence analysis, and iteration details"
	OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_IMAGE=true go run -tags matprofile .

run-debug-algorithms:
	@echo "Running with comprehensive algorithm debugging enabled..."
	@echo "Shows both 2D Otsu and Iterative Triclass algorithm details"
	OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_IMAGE=true go run -tags matprofile .

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

run-debug-performance:
	@echo "Running with performance debugging enabled..."
	OTSU_DEBUG_PERFORMANCE=true go run -tags matprofile .

run-debug-memory:
	@echo "Running with memory debugging enabled..."
	OTSU_DEBUG_MEMORY=true go run -tags matprofile .

run-debug-image:
	@echo "Running with image processing debugging enabled..."
	OTSU_DEBUG_IMAGE=true go run -tags matprofile .

run-debug-pixels:
	@echo "Running with pixel-level analysis debugging enabled..."
	@echo "This will show detailed pixel sampling and analysis"
	OTSU_DEBUG_PIXELS=true go run -tags matprofile .

# Comprehensive debug modes
run-debug-comprehensive:
	@echo "Running with comprehensive debugging for algorithm development..."
	@echo "Enables algorithm, image, performance, and memory debugging"
	OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_IMAGE=true OTSU_DEBUG_PERFORMANCE=true OTSU_DEBUG_MEMORY=true go run -tags matprofile .

run-debug-all:
	@echo "Running with ALL debugging enabled..."
	OTSU_DEBUG_FORMAT=true OTSU_DEBUG_IMAGE=true OTSU_DEBUG_MEMORY=true OTSU_DEBUG_PERFORMANCE=true OTSU_DEBUG_GUI=true OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_PIXELS=true go run -tags matprofile .

# Safe debug target - mathematical algorithm focus without pixel analysis
run-debug-math:
	@echo "Running with mathematical algorithm debugging (safe mode)..."
	@echo "Focuses on mathematical correctness without pixel-level analysis"
	OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_PERFORMANCE=true go run -tags matprofile .

# Algorithm validation and testing
run-validate-2d-otsu:
	@echo "Running 2D Otsu algorithm validation..."
	@echo "Tests parameter validation, mathematical correctness, and edge cases"
	OTSU_DEBUG_ALGORITHMS=true OTSU_VALIDATE_2D_OTSU=true go run -tags matprofile .

run-validate-triclass:
	@echo "Running Iterative Triclass algorithm validation..."
	@echo "Tests convergence behavior, parameter validation, and mathematical correctness"
	OTSU_DEBUG_TRICLASS=true OTSU_VALIDATE_TRICLASS=true go run -tags matprofile .

# Performance benchmarking
run-benchmark-algorithms:
	@echo "Running algorithm performance benchmarks..."
	@echo "Compares 2D Otsu vs Iterative Triclass performance on various image types"
	OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_PERFORMANCE=true OTSU_BENCHMARK_MODE=true go run -tags matprofile .

# Thread safety testing for Fyne v2.6+
run-test-thread-safety:
	@echo "Running with thread safety validation for Fyne v2.6+..."
	@echo "Tests fyne.Do usage and concurrent access patterns"
	OTSU_DEBUG_GUI=true OTSU_TEST_THREAD_SAFETY=true go run -tags matprofile .

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

# Dependency verification with version checks
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
	@echo "Verifying Fyne v2.6+ compatibility..."
	@go list -m fyne.io/fyne/v2 | grep -E "v2\.[6-9]\.|v[3-9]\." > /dev/null || echo "WARNING: Fyne v2.6+ recommended for full compatibility"
	@echo "All dependencies OK!"

# Test with profiling enabled
test:
	@echo "Running tests with Mat profiling..."
	go test -tags matprofile ./...

# Algorithm-specific testing
test-2d-otsu:
	@echo "Running 2D Otsu algorithm tests..."
	go test -tags matprofile -v ./otsu -run "*2DOtsu*"

test-triclass:
	@echo "Running Iterative Triclass algorithm tests..."
	go test -tags matprofile -v ./otsu -run "*Triclass*"

test-algorithms:
	@echo "Running comprehensive algorithm tests..."
	go test -tags matprofile -v ./otsu

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
	@echo "Run 'make run-debug-algorithms' to start with algorithm debugging"
	@echo "Run 'make run-debug-math' for mathematical algorithm validation"
	@echo "Run 'make run-validate-2d-otsu' or 'make run-validate-triclass' for algorithm testing"

# Production workflow target  
prod: check-deps deps test build
	@echo "Production build complete!"
	@echo "Binary: $(BINARY_NAME)"

# Algorithm development workflow
dev-algorithms: check-deps deps
	@echo "Algorithm development environment ready!"
	@echo "Available targets:"
	@echo "  make run-debug-2d-otsu       - Debug 2D Otsu algorithm"
	@echo "  make run-debug-triclass      - Debug Iterative Triclass algorithm" 
	@echo "  make run-debug-algorithms    - Debug both algorithms"
	@echo "  make run-validate-2d-otsu    - Validate 2D Otsu implementation"
	@echo "  make run-validate-triclass   - Validate Triclass implementation"
	@echo "  make test-algorithms         - Run algorithm test suite"
	@echo "  make run-benchmark-algorithms- Performance benchmarking"

# Help target
help:
	@echo "Otsu Obliterator - Available Targets:"
	@echo ""
	@echo "🔧 MAIN TARGETS:"
	@echo "  dev                       - Set up development environment"
	@echo "  dev-algorithms            - Set up algorithm development environment"
	@echo "  build-profile             - Build with memory profiling enabled"
	@echo "  run-profile               - Run with memory profiling"
	@echo "  check-leaks               - Run with memory leak detection"
	@echo "  profile                   - Start with full profiling server"
	@echo "  prod                      - Full production build workflow"
	@echo ""
	@echo "🚀 PRODUCTION:"
	@echo "  build                     - Build production binary"
	@echo "  run                       - Run production binary"
	@echo ""
	@echo "🧮 ALGORITHM DEBUGGING:"
	@echo "  run-debug-2d-otsu         - Debug 2D Otsu algorithm implementation"
	@echo "  run-debug-triclass        - Debug Iterative Triclass algorithm"
	@echo "  run-debug-algorithms      - Debug both algorithms comprehensively"
	@echo "  run-debug-math            - Mathematical algorithm debugging (safe mode)"
	@echo "  run-validate-2d-otsu      - Validate 2D Otsu mathematical correctness"
	@echo "  run-validate-triclass     - Validate Triclass convergence behavior"
	@echo "  run-benchmark-algorithms  - Performance comparison benchmarking"
	@echo ""
	@echo "🐛 COMPONENT DEBUGGING:"
	@echo "  run-debug-format          - Format detection and image loading"
	@echo "  run-debug-gui             - GUI events and interactions"
	@echo "  run-debug-performance     - Performance timing and metrics"
	@echo "  run-debug-memory          - Memory usage and tracking"
	@echo "  run-debug-image           - Image processing and conversion"
	@echo "  run-debug-pixels          - Pixel-level analysis and sampling"
	@echo "  run-debug-comprehensive   - Multiple debug categories"
	@echo "  run-debug-all             - All debugging enabled"
	@echo ""
	@echo "🧪 TESTING:"
	@echo "  test                      - Run full test suite"
	@echo "  test-algorithms           - Run algorithm-specific tests"
	@echo "  test-2d-otsu              - Test 2D Otsu implementation"
	@echo "  test-triclass             - Test Iterative Triclass implementation"
	@echo "  run-test-thread-safety    - Test Fyne v2.6+ thread safety"
	@echo ""
	@echo "🌍 CROSS-PLATFORM:"
	@echo "  build-windows             - Build for Windows"
	@echo "  build-macos               - Build for macOS (Intel)"
	@echo "  build-macos-arm64         - Build for macOS (Apple Silicon)"
	@echo "  build-macos-universal     - Build universal macOS binary"
	@echo "  build-macos-app           - Create macOS app bundle"
	@echo "  build-linux               - Build for Linux"
	@echo ""
	@echo "🔧 MAINTENANCE:"
	@echo "  deps                      - Install dependencies"
	@echo "  check-deps                - Verify system dependencies"
	@echo "  clean                     - Clean build artifacts"
	@echo "  fmt                       - Format code"
	@echo "  lint                      - Lint code"
	@echo "  vet                       - Vet code"
	@echo ""
	@echo "🚀 ALGORITHM DEVELOPMENT QUICK START:"
	@echo "  make dev-algorithms && make run-debug-math"
	@echo ""
	@echo "📈 PROFILING:"
	@echo "  profile-count             - Get current MatProfile count"
	@echo ""
	@echo "⚠️  NOTES:"
	@echo "  - Use 'run-debug-math' for safe algorithm development"
	@echo "  - Use 'run-debug-algorithms' for comprehensive algorithm analysis"
	@echo "  - All debug modes include memory profiling automatically"
	@echo "  - Fyne v2.6+ thread safety is automatically tested in debug modes"

# Default target
.DEFAULT_GOAL := help