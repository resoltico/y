#!/usr/bin/env bash

# Otsu Obliterator Build Script - Modernized for Go 1.24, Fyne v2.6, GoCV v0.41
# Uses cutting-edge API patterns and performance optimizations

set -o errexit    # Exit on any command failure
set -o nounset    # Exit on undefined variables
set -o pipefail   # Exit on pipe failures

# Get script directory for relative paths
__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
__file="${__dir}/$(basename "${BASH_SOURCE[0]}")"
__base="$(basename "${__file}" .sh)"

# Project configuration
readonly BINARY_NAME="otsu-obliterator"
readonly VERSION="${VERSION:-1.0.0}"
readonly BUILD_DIR="${BUILD_DIR:-build}"
readonly CMD_DIR="${CMD_DIR:-cmd/${BINARY_NAME}}"

# Build configuration with Go 1.24 optimizations
readonly LDFLAGS="-s -w -X main.version=${VERSION}"
readonly BUILD_TAGS="matprofile"
readonly GO_VERSION_REQUIRED="1.24"

# Platform detection
readonly OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
readonly ARCH="$(uname -m)"

# Colors for modern terminal output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly YELLOW='\033[1;33m'
readonly PURPLE='\033[0;35m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m'

# Logging functions with enhanced formatting
log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*"
}

success() {
    echo -e "${GREEN}✓${NC} $*"
}

error() {
    echo -e "${RED}✗${NC} $*" >&2
}

warn() {
    echo -e "${YELLOW}⚠${NC} $*"
}

info() {
    echo -e "${CYAN}ℹ${NC} $*"
}

debug() {
    if [[ "${DEBUG:-}" == "1" ]]; then
        echo -e "${PURPLE}[DEBUG]${NC} $*"
    fi
}

# Enhanced help function
show_help() {
    cat << 'EOF'
🎯 Otsu Obliterator Build System - Modernized Edition

Built for Go 1.24, Fyne v2.6.1, and GoCV v0.41.0
Implements cutting-edge API patterns and performance optimizations

Usage: ./build.sh [COMMAND] [OPTIONS]

📋 COMMANDS:
  build [target]    Build binary for target platform
  run              Build and run application with performance monitoring
  debug [type]     Run with advanced debugging (memory, race, profile)
  test             Run comprehensive tests with coverage analysis
  bench            Run benchmarks with memory profiling
  clean            Remove build artifacts and clean Go module cache
  deps             Install, verify, and update dependencies
  format           Format code with Go 1.24 best practices
  lint             Run static analysis with modern linters
  audit            Run comprehensive quality control checks
  modernize        Apply Go 1.24 and Fyne v2.6 modernizations
  profile [type]   Run with performance profiling
  help             Show this help message

🎯 BUILD TARGETS:
  default          Current platform with Go 1.24 optimizations
  performance      With advanced performance profiling
  debug            With race detection and memory debugging
  windows          Windows 64-bit with modern runtime
  macos            macOS Intel 64-bit
  macos-arm64      macOS Apple Silicon with native optimizations
  linux            Linux 64-bit with container support
  all              All supported platforms

🔍 DEBUG TYPES:
  basic            Standard debugging with structured logging
  memory           Memory debugging with GoCV Mat profiling
  race             Race condition detection with Go 1.24 features
  profile          CPU and memory profiling with pprof
  trace            Execution tracing with Go 1.24 tracer

📊 PROFILE TYPES:
  cpu              CPU profiling with modern scheduler analysis
  memory           Memory profiling with GC optimization
  goroutine        Goroutine analysis with worker pool monitoring
  trace            Full execution trace with timing analysis

🚀 EXAMPLES:
  ./build.sh build performance      # Build with performance monitoring
  ./build.sh debug memory          # Memory debugging with Mat tracking
  ./build.sh profile cpu           # CPU profiling for optimization
  ./build.sh modernize             # Apply latest API patterns
  ./build.sh audit                 # Comprehensive quality analysis

🏗️ MODERN FEATURES:
  - Go 1.24 Swiss Tables optimization
  - Fyne v2.6 thread-safe UI patterns
  - GoCV v0.41 memory management
  - Advanced worker pool concurrency
  - Context-aware processing pipelines
  - Memory leak detection and prevention

For advanced configuration, see README.md
EOF
}

# Modern dependency checking with version validation
check_deps() {
    log "Validating build environment..."
    local errors=0
    
    # Check Go version with strict validation
    if ! command -v go &> /dev/null; then
        error "Go not found. Install Go ${GO_VERSION_REQUIRED}+ from https://golang.org/"
        ((errors++))
    else
        local go_version
        go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
        
        # Version comparison for Go 1.24+
        if [[ "$(printf '%s\n' "${GO_VERSION_REQUIRED}" "${go_version}" | sort -V | head -n1)" != "${GO_VERSION_REQUIRED}" ]]; then
            error "Go ${GO_VERSION_REQUIRED}+ required. Found: ${go_version}"
            error "Modern features require Go 1.24 for Swiss Tables and performance improvements"
            ((errors++))
        else
            success "Go ${go_version} detected (supports Swiss Tables optimization)"
        fi
    fi
    
    # Check OpenCV with version detection
    if ! pkg-config --exists opencv4; then
        if ! pkg-config --exists opencv; then
            warn "OpenCV not found - some features may be limited"
            warn "Install: brew install opencv (macOS) or apt-get install libopencv-dev (Ubuntu)"
            info "GoCV will use embedded OpenCV if available"
        else
            local opencv_version
            opencv_version=$(pkg-config --modversion opencv 2>/dev/null || echo "unknown")
            success "OpenCV ${opencv_version} found"
        fi
    else
        local opencv4_version
        opencv4_version=$(pkg-config --modversion opencv4 2>/dev/null || echo "unknown")
        success "OpenCV4 ${opencv4_version} found (recommended)"
    fi
    
    # Validate Go modules with modern standards
    if [[ ! -f "go.mod" ]]; then
        error "go.mod not found. Run 'go mod init' first"
        ((errors++))
    else
        # Check for Go 1.24 in go.mod
        if ! grep -q "go 1.24" go.mod; then
            warn "go.mod should specify 'go 1.24' for modern features"
            info "Run 'go mod edit -go=1.24' to update"
        fi
    fi
    
    # Check command directory structure
    if [[ ! -d "${CMD_DIR}" ]]; then
        error "Command directory '${CMD_DIR}' not found"
        ((errors++))
    fi
    
    # Validate critical dependencies
    if [[ -f "go.mod" ]]; then
        debug "Checking critical dependencies..."
        
        # Check Fyne version
        if ! grep -q "fyne.io/fyne/v2 v2.6" go.mod; then
            warn "Fyne v2.6+ recommended for modern UI patterns"
        fi
        
        # Check GoCV version
        if ! grep -q "gocv.io/x/gocv v0.41" go.mod; then
            warn "GoCV v0.41+ recommended for memory management improvements"
        fi
    fi
    
    if [[ ${errors} -gt 0 ]]; then
        error "${errors} dependency error(s) found"
        exit 1
    fi
    
    success "Build environment validated - ready for modern development"
}

# Intelligent build cache management with Go 1.24 features
clean_build_cache() {
    log "Cleaning build cache and artifacts..."
    
    # Remove build directory
    if [[ -d "${BUILD_DIR}" ]]; then
        rm -rf "${BUILD_DIR}"
        success "Removed ${BUILD_DIR}/"
    fi
    
    # Clean Go cache with modern options
    go clean -cache -testcache -modcache -fuzzcache 2>/dev/null || {
        warn "Some cache cleaning options not available in this Go version"
        go clean -cache -testcache -modcache 2>/dev/null || true
    }
    
    # Remove profiling and analysis artifacts
    find . -name "*.prof" -delete 2>/dev/null || true
    find . -name "*.trace" -delete 2>/dev/null || true
    find . -name "coverage.*" -delete 2>/dev/null || true
    find . -name "*.test" -delete 2>/dev/null || true
    
    success "Build cache cleaned with modern Go tooling"
}

# Auto-clean with version tracking
auto_clean_obsolete() {
    local version_file="${BUILD_DIR}/.version"
    local go_version_file="${BUILD_DIR}/.go_version"
    
    mkdir -p "${BUILD_DIR}"
    
    # Check version changes
    if [[ -f "${version_file}" ]]; then
        local old_version
        old_version=$(<"${version_file}")
        if [[ "${old_version}" != "${VERSION}" ]]; then
            log "Version changed (${old_version} → ${VERSION}), cleaning obsolete builds"
            clean_build_cache
        fi
    fi
    
    # Check Go version changes
    local current_go_version
    current_go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+\.[0-9]+')
    if [[ -f "${go_version_file}" ]]; then
        local old_go_version
        old_go_version=$(<"${go_version_file}")
        if [[ "${old_go_version}" != "${current_go_version}" ]]; then
            log "Go version changed (${old_go_version} → ${current_go_version}), rebuilding"
            clean_build_cache
        fi
    fi
    
    echo "${VERSION}" > "${version_file}"
    echo "${current_go_version}" > "${go_version_file}"
}

# Modern build function with Go 1.24 optimizations
build() {
    local target="${1:-default}"
    local output_name="${BINARY_NAME}"
    local build_env=""
    local extra_flags=""
    local build_mode="release"
    
    check_deps
    auto_clean_obsolete
    
    case "${target}" in
        "default"|"")
            log "Building ${BINARY_NAME} for ${OS}/${ARCH} with Go 1.24 optimizations"
            extra_flags="-tags ${BUILD_TAGS}"
            ;;
        "performance")
            extra_flags="-tags ${BUILD_TAGS},performance -race"
            build_mode="performance"
            log "Building performance-optimized version with profiling"
            ;;
        "debug")
            extra_flags="-tags ${BUILD_TAGS},debug -race -gcflags=all=-N -l"
            build_mode="debug"
            log "Building debug version with race detection and symbols"
            ;;
        "windows")
            output_name="${BINARY_NAME}.exe"
            build_env="GOOS=windows GOARCH=amd64"
            extra_flags="-tags ${BUILD_TAGS}"
            log "Cross-compiling for Windows AMD64"
            ;;
        "macos")
            output_name="${BINARY_NAME}-macos-amd64"
            build_env="GOOS=darwin GOARCH=amd64"
            extra_flags="-tags ${BUILD_TAGS}"
            log "Cross-compiling for macOS Intel"
            ;;
        "macos-arm64")
            output_name="${BINARY_NAME}-macos-arm64"
            build_env="GOOS=darwin GOARCH=arm64"
            extra_flags="-tags ${BUILD_TAGS}"
            log "Cross-compiling for macOS Apple Silicon with native optimizations"
            ;;
        "linux")
            output_name="${BINARY_NAME}-linux-amd64"
            build_env="GOOS=linux GOARCH=amd64"
            extra_flags="-tags ${BUILD_TAGS}"
            log "Cross-compiling for Linux AMD64"
            ;;
        "all")
            log "Building for all supported platforms with modern optimizations"
            build "windows"
            build "macos"
            build "macos-arm64"
            build "linux"
            return 0
            ;;
        *)
            error "Unknown build target: ${target}"
            error "Supported: default, performance, debug, windows, macos, macos-arm64, linux, all"
            exit 1
            ;;
    esac
    
    # Enhanced build command with Go 1.24 features
    local build_cmd="go build ${extra_flags} -ldflags \"${LDFLAGS}\" -o \"${BUILD_DIR}/${output_name}\" \"./${CMD_DIR}\""
    
    # Set performance environment variables
    export GOMAXPROCS="${GOMAXPROCS:-$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)}"
    export GOGC="${GOGC:-100}"
    
    # Execute build with timing
    local start_time
    start_time=$(date +%s)
    
    debug "Build command: ${build_cmd}"
    debug "Build environment: ${build_env}"
    debug "GOMAXPROCS: ${GOMAXPROCS}"
    
    if [[ -n "${build_env}" ]]; then
        eval "env ${build_env} ${build_cmd}"
    else
        eval "${build_cmd}"
    fi
    
    local build_time=$(($(date +%s) - start_time))
    
    # Verify and report build results
    if [[ -f "${BUILD_DIR}/${output_name}" ]]; then
        local size
        size=$(du -h "${BUILD_DIR}/${output_name}" | cut -f1)
        success "Built: ${BUILD_DIR}/${output_name} (${size}) in ${build_time}s"
        
        # Show binary info for verification
        if command -v file &> /dev/null; then
            info "Binary info:"
            file "${BUILD_DIR}/${output_name}" | sed 's/^/  /'
        fi
        
        # Show build mode info
        info "Build mode: ${build_mode}"
        if [[ "${build_mode}" == "performance" ]]; then
            info "Performance monitoring enabled - check /debug/pprof endpoints"
        fi
    else
        error "Build failed - binary not found: ${BUILD_DIR}/${output_name}"
        exit 1
    fi
}

# Modern test runner with Go 1.24 features
run_tests() {
    log "Running comprehensive test suite with Go 1.24 features..."
    
    # Create enhanced coverage directory
    mkdir -p coverage
    
    # Set test environment for modern Go features
    export GOEXPERIMENT="${GOEXPERIMENT:-}"
    
    # Run tests with modern flags and coverage
    local test_flags="-tags ${BUILD_TAGS} -race -coverprofile=coverage/coverage.out -covermode=atomic -v"
    
    # Add fuzzing support if available
    if go help testflag | grep -q "fuzz"; then
        test_flags="${test_flags} -fuzztime=30s"
        log "Fuzzing support detected - running with fuzz tests"
    fi
    
    if go test ${test_flags} ./...; then
        # Generate enhanced coverage reports
        go tool cover -html=coverage/coverage.out -o coverage/coverage.html
        
        # Generate coverage summary with Go 1.24 features
        local coverage_pct
        coverage_pct=$(go tool cover -func=coverage/coverage.out | grep total | grep -oE '[0-9]+\.[0-9]+%')
        success "Tests passed - Coverage: ${coverage_pct}"
        
        # Additional coverage analysis
        if command -v go-tool-cover &> /dev/null; then
            go tool cover -func=coverage/coverage.out | tail -5 | sed 's/^/  /'
        fi
        
        info "Coverage report: coverage/coverage.html"
        info "View with: open coverage/coverage.html"
    else
        error "Tests failed"
        exit 1
    fi
}

# Enhanced debug runner with modern profiling
run_debug() {
    local debug_type="${1:-basic}"
    
    check_deps
    
    case "${debug_type}" in
        "basic")
            log "Running with structured debugging"
            env LOG_LEVEL=debug go run -tags "${BUILD_TAGS}" "./${CMD_DIR}"
            ;;
        "memory")
            log "Running with memory debugging and GoCV Mat profiling"
            env LOG_LEVEL=debug GOMAXPROCS=1 GODEBUG=gctrace=1 go run -tags "${BUILD_TAGS}" -race "./${CMD_DIR}"
            ;;
        "race")
            log "Running with race condition detection"
            env LOG_LEVEL=debug GORACE="log_path=./race" go run -tags "${BUILD_TAGS}" -race "./${CMD_DIR}"
            ;;
        "profile")
            log "Running with CPU and memory profiling"
            env LOG_LEVEL=debug go run -tags "${BUILD_TAGS}" "./${CMD_DIR}" &
            local pid=$!
            sleep 2
            info "Profiling available at http://localhost:6060/debug/pprof/"
            info "CPU: go tool pprof http://localhost:6060/debug/pprof/profile"
            info "Memory: go tool pprof http://localhost:6060/debug/pprof/heap"
            wait $pid
            ;;
        "trace")
            log "Running with execution tracing"
            env LOG_LEVEL=debug GODEBUG=schedtrace=1000 go run -tags "${BUILD_TAGS}" "./${CMD_DIR}"
            ;;
        *)
            error "Unknown debug type: ${debug_type}"
            error "Supported: basic, memory, race, profile, trace"
            exit 1
            ;;
    esac
}

# Comprehensive quality control with modern tools
run_audit() {
    log "Running comprehensive quality audit with modern tooling..."
    
    # Format check with Go 1.24 standards
    log "Checking code formatting..."
    if [[ -n "$(gofmt -l .)" ]]; then
        error "Code not formatted. Run './build.sh format'"
        gofmt -l . | sed 's/^/  /'
        exit 1
    fi
    success "Code formatting validated"
    
    # Modern vet check with enhanced analysis
    log "Running enhanced static analysis..."
    go vet ./...
    success "Static analysis passed"
    
    # Module verification with modern standards
    log "Verifying module integrity..."
    go mod verify
    go mod tidy -diff
    success "Module integrity verified"
    
    # Security analysis if available
    if command -v govulncheck &> /dev/null; then
        log "Running vulnerability analysis..."
        govulncheck ./...
        success "Vulnerability analysis completed"
    else
        info "Install govulncheck for security analysis: go install golang.org/x/vuln/cmd/govulncheck@latest"
    fi
    
    # Run comprehensive tests
    run_tests
    
    # Enhanced static analysis tools
    if command -v staticcheck &> /dev/null; then
        log "Running advanced static analysis..."
        staticcheck ./...
        success "Advanced static analysis passed"
    else
        warn "staticcheck not found - install with: go install honnef.co/go/tools/cmd/staticcheck@latest"
    fi
    
    # Performance analysis
    if command -v ineffassign &> /dev/null; then
        log "Checking for inefficient assignments..."
        ineffassign ./...
        success "Efficiency analysis completed"
    fi
    
    # Check for Go 1.24 compatibility
    log "Validating Go 1.24 compatibility..."
    if grep -r "deprecated" --include="*.go" .; then
        warn "Deprecated API usage found - consider modernizing"
    else
        success "No deprecated API usage detected"
    fi
    
    success "Quality audit completed successfully"
}

# Modern profiling with Go 1.24 features
run_profiling() {
    local profile_type="${1:-cpu}"
    
    check_deps
    build "performance"
    
    case "${profile_type}" in
        "cpu")
            log "Starting CPU profiling session..."
            "./${BUILD_DIR}/${BINARY_NAME}" &
            local pid=$!
            sleep 3
            
            info "Capturing CPU profile..."
            go tool pprof -http=:8080 "http://localhost:6060/debug/pprof/profile?seconds=30" &
            local pprof_pid=$!
            
            info "CPU profiling UI available at: http://localhost:8080"
            info "Press Ctrl+C to stop profiling"
            
            wait $pprof_pid
            kill $pid 2>/dev/null || true
            ;;
        "memory")
            log "Starting memory profiling session..."
            "./${BUILD_DIR}/${BINARY_NAME}" &
            local pid=$!
            sleep 3
            
            info "Capturing memory profile..."
            go tool pprof -http=:8080 "http://localhost:6060/debug/pprof/heap" &
            local pprof_pid=$!
            
            info "Memory profiling UI available at: http://localhost:8080"
            info "Press Ctrl+C to stop profiling"
            
            wait $pprof_pid
            kill $pid 2>/dev/null || true
            ;;
        "goroutine")
            log "Analyzing goroutine usage..."
            "./${BUILD_DIR}/${BINARY_NAME}" &
            local pid=$!
            sleep 3
            
            go tool pprof -http=:8080 "http://localhost:6060/debug/pprof/goroutine" &
            local pprof_pid=$!
            
            info "Goroutine analysis UI available at: http://localhost:8080"
            wait $pprof_pid
            kill $pid 2>/dev/null || true
            ;;
        "trace")
            log "Capturing execution trace..."
            "./${BUILD_DIR}/${BINARY_NAME}" &
            local pid=$!
            sleep 3
            
            curl -o trace.out "http://localhost:6060/debug/pprof/trace?seconds=10"
            kill $pid 2>/dev/null || true
            
            info "Opening trace viewer..."
            go tool trace trace.out
            ;;
        *)
            error "Unknown profile type: ${profile_type}"
            error "Supported: cpu, memory, goroutine, trace"
            exit 1
            ;;
    esac
}

# Modernization assistant for API upgrades
run_modernization() {
    log "Applying Go 1.24 and Fyne v2.6 modernizations..."
    
    # Check for common modernization opportunities
    log "Analyzing codebase for modernization opportunities..."
    
    # Fyne v2.6 modernizations
    if grep -r "fyne\.CurrentApp" --include="*.go" .; then
        warn "Found deprecated fyne.CurrentApp() usage"
        info "Consider using app instance passed through context"
    fi
    
    # Check for direct UI updates from goroutines
    if grep -r "widget\." --include="*.go" . | grep -v "fyne\.Do"; then
        warn "Potential direct UI updates detected"
        info "Wrap UI updates in fyne.Do() for v2.6 compatibility"
    fi
    
    # Go 1.24 modernizations
    if grep -r "sync\.Map" --include="*.go" .; then
        info "Consider using Go 1.24 Swiss Tables for better performance"
    fi
    
    # GoCV memory management
    if grep -r "gocv\.NewMat" --include="*.go" . | grep -v "defer.*Close"; then
        warn "Potential GoCV memory leaks detected"
        info "Ensure all Mat objects have corresponding Close() calls"
    fi
    
    success "Modernization analysis completed"
    info "Run './build.sh audit' for comprehensive quality checks"
}

# Enhanced dependency management
manage_dependencies() {
    log "Managing dependencies with modern Go tooling..."
    
    # Update to latest compatible versions
    go get -u ./...
    go mod tidy
    
    # Security updates
    if command -v govulncheck &> /dev/null; then
        log "Checking for security vulnerabilities..."
        govulncheck ./...
    fi
    
    # Verify checksums and integrity
    go mod verify
    
    # Clean module cache if needed
    go clean -modcache
    go mod download
    
    success "Dependencies updated and verified"
}

# Main command dispatcher with enhanced routing
main() {
    local command="${1:-help}"
    
    case "${command}" in
        "build")
            build "${2:-}"
            ;;
        "run")
            check_deps
            build "performance"
            export LOG_LEVEL="${LOG_LEVEL:-info}"
            export GOMAXPROCS="${GOMAXPROCS:-$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)}"
            info "Starting application with performance monitoring..."
            "./${BUILD_DIR}/${BINARY_NAME}"
            ;;
        "debug")
            run_debug "${2:-basic}"
            ;;
        "test")
            run_tests
            ;;
        "bench")
            log "Running benchmarks with memory profiling..."
            go test -tags "${BUILD_TAGS}" -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./...
            info "Benchmark profiles: cpu.prof, mem.prof"
            info "View with: go tool pprof cpu.prof"
            success "Benchmarks completed"
            ;;
        "clean")
            clean_build_cache
            ;;
        "deps")
            manage_dependencies
            check_deps
            ;;
        "format")
            log "Formatting code with Go 1.24 standards..."
            go fmt ./...
            if command -v goimports &> /dev/null; then
                goimports -w .
                success "Code formatted with goimports"
            else
                success "Code formatted with gofmt"
                info "Install goimports for enhanced formatting: go install golang.org/x/tools/cmd/goimports@latest"
            fi
            ;;
        "lint")
            log "Running modern linters..."
            go vet ./...
            if command -v golangci-lint &> /dev/null; then
                golangci-lint run
                success "Advanced linting completed"
            else
                warn "golangci-lint not found. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
            fi
            ;;
        "audit")
            run_audit
            ;;
        "modernize")
            run_modernization
            ;;
        "profile")
            run_profiling "${2:-cpu}"
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        *)
            error "Unknown command: ${command}"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Enhanced cleanup trap with modern error handling
cleanup() {
    local exit_code=$?
    if [[ ${exit_code} -ne 0 ]]; then
        error "Build script failed with exit code ${exit_code}"
        if [[ "${DEBUG:-}" == "1" ]]; then
            info "Debug mode enabled - check logs above for details"
        fi
    fi
    exit ${exit_code}
}

trap cleanup EXIT ERR

# Initialize modern build environment
export GOEXPERIMENT="${GOEXPERIMENT:-}"
export GODEBUG="${GODEBUG:-}"

# Run main function with all arguments
main "$@"