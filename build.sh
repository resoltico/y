#!/bin/bash

set -e

# Project settings
BINARY_NAME="otsu-obliterator"
VERSION=${VERSION:-"1.0.0"}
BUILD_DIR=${BUILD_DIR:-"build"}

# Auto-detect CMD_DIR by finding the actual cmd subdirectory
if [ -z "$CMD_DIR" ]; then
    if [ -d "cmd/${BINARY_NAME}" ]; then
        CMD_DIR="cmd/${BINARY_NAME}"
    elif [ -d "cmd" ] && [ "$(find cmd -mindepth 1 -maxdepth 1 -type d | wc -l)" -eq 1 ]; then
        CMD_DIR=$(find cmd -mindepth 1 -maxdepth 1 -type d | head -1)
    else
        CMD_DIR="cmd/${BINARY_NAME}"
    fi
fi

# Build flags with memory profiling
LDFLAGS="-s -w -X main.version=${VERSION}"
BUILD_TAGS="matprofile"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

check_deps() {
    log "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        error "Go not found. Install Go 1.24+"
        exit 1
    fi
    
    # Check Go version
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    if [ "$(printf '%s\n' "1.24" "$GO_VERSION" | sort -V | head -n1)" != "1.24" ]; then
        warn "Go version $GO_VERSION detected. Go 1.24+ recommended"
    fi
    
    if ! pkg-config --exists opencv4 && ! pkg-config --exists opencv; then
        warn "OpenCV not found. Install OpenCV 4.11.0+ for full functionality"
        warn "Ubuntu/Debian: sudo apt-get install libopencv-dev"
        warn "macOS: brew install opencv"
        warn "Windows: See https://gocv.io/getting-started/"
    fi
    
    success "Dependencies checked"
}

build() {
    local target=${1:-"default"}
    local output_name="${BINARY_NAME}"
    local extra_flags=""
    local env_vars=""
    
    case $target in
        "profile"|"debug")
            extra_flags="-tags ${BUILD_TAGS}"
            log "Building with memory profiling and race detection..."
            ;;
        "windows")
            output_name="${BINARY_NAME}.exe"
            env_vars="GOOS=windows GOARCH=amd64"
            extra_flags="-tags ${BUILD_TAGS}"
            ;;
        "macos")
            output_name="${BINARY_NAME}-macos-amd64"
            env_vars="GOOS=darwin GOARCH=amd64"
            extra_flags="-tags ${BUILD_TAGS}"
            ;;
        "macos-arm64")
            output_name="${BINARY_NAME}-macos-arm64"
            env_vars="GOOS=darwin GOARCH=arm64"
            extra_flags="-tags ${BUILD_TAGS}"
            ;;
        "linux")
            output_name="${BINARY_NAME}-linux-amd64"
            env_vars="GOOS=linux GOARCH=amd64"
            extra_flags="-tags ${BUILD_TAGS}"
            ;;
        *)
            log "Building ${BINARY_NAME}..."
            ;;
    esac
    
    mkdir -p "${BUILD_DIR}"
    
    # Use go build instead of fyne build to preserve main() entry point
    if [ -n "$env_vars" ]; then
        if ! env $env_vars go build ${extra_flags} -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${output_name}" "./${CMD_DIR}"; then
            error "Build failed"
            exit 1
        fi
    else
        if ! go build ${extra_flags} -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${output_name}" "./${CMD_DIR}"; then
            error "Build failed"
            exit 1
        fi
    fi
    
    success "Built: ${BUILD_DIR}/${output_name}"
}

package_app() {
    local target=${1:-"default"}
    
    check_deps
    build $target
    
    log "Packaging application with Fyne..."
    
    if ! command -v fyne &> /dev/null; then
        log "Installing fyne tool..."
        go install fyne.io/fyne/v2/cmd/fyne@latest
    fi
    
    case $target in
        "windows")
            if [ -f "${BUILD_DIR}/${BINARY_NAME}.exe" ]; then
                fyne package -o "${BUILD_DIR}/${BINARY_NAME}-installer.exe" -os windows "./${CMD_DIR}"
                success "Windows package: ${BUILD_DIR}/${BINARY_NAME}-installer.exe"
            fi
            ;;
        "macos"|"macos-arm64")
            if [ -f "${BUILD_DIR}/${BINARY_NAME}-macos-"* ]; then
                fyne package -o "${BUILD_DIR}/${BINARY_NAME}.app" -os darwin "./${CMD_DIR}"
                success "macOS package: ${BUILD_DIR}/${BINARY_NAME}.app"
            fi
            ;;
        "linux")
            if [ -f "${BUILD_DIR}/${BINARY_NAME}-linux-amd64" ]; then
                fyne package -o "${BUILD_DIR}/${BINARY_NAME}.tar.xz" -os linux "./${CMD_DIR}"
                success "Linux package: ${BUILD_DIR}/${BINARY_NAME}.tar.xz"
            fi
            ;;
        *)
            fyne package -o "${BUILD_DIR}/${BINARY_NAME}-package" "./${CMD_DIR}"
            success "Package created: ${BUILD_DIR}/${BINARY_NAME}-package"
            ;;
    esac
}

run_with_env() {
    local env_vars="$1"
    local message="$2"
    
    log "$message"
    
    if [ -n "$env_vars" ]; then
        env $env_vars go run -tags ${BUILD_TAGS} -race "./${CMD_DIR}"
    else
        if [ -f "${BUILD_DIR}/${BINARY_NAME}" ]; then
            "./${BUILD_DIR}/${BINARY_NAME}"
        else
            build "profile"
            "./${BUILD_DIR}/${BINARY_NAME}"
        fi
    fi
}

show_help() {
    cat << EOF
Usage: $0 [command] [options]

Commands:
  build [target]      Build binary (default, profile, debug, windows, macos, macos-arm64, linux)
  package [target]    Build and package application with Fyne packaging
  run                 Build and run application with memory tracking
  debug [type]        Run with debug environment variables
  test                Run tests with coverage
  bench               Run benchmarks
  clean               Clean build artifacts
  deps                Install and verify dependencies
  format              Format code and organize imports
  lint                Run linters
  help                Show this help

Debug types:
  basic               Basic application debugging (LOG_LEVEL=debug)
  memory              Memory usage monitoring with MatProfile
  all                 All debugging features enabled

Performance targets:
  profile             Build with profiling enabled
  debug               Build with race detection and profiling

Packaging targets:
  package windows     Create Windows installer
  package macos       Create macOS application bundle
  package linux       Create Linux distribution package

Examples:
  $0 build profile    Build with memory profiling
  $0 package windows  Create Windows installer package
  $0 debug memory     Run with memory debugging
  $0 build windows    Cross-compile for Windows
  $0 bench            Run performance benchmarks
EOF
}

case "${1:-help}" in
    "build")
        check_deps
        build "${2:-default}"
        ;;
    "package")
        package_app "${2:-default}"
        ;;
    "run")
        check_deps
        build "profile"
        export LOG_LEVEL=info
        export GOMAXPROCS=${GOMAXPROCS:-$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)}
        "./${BUILD_DIR}/${BINARY_NAME}"
        ;;
    "debug")
        check_deps
        case "${2:-basic}" in
            "basic")
                run_with_env "LOG_LEVEL=debug" "Running with basic debugging..."
                ;;
            "memory")
                run_with_env "LOG_LEVEL=debug GOMAXPROCS=1" "Running with memory debugging and MatProfile..."
                ;;
            "all")
                run_with_env "LOG_LEVEL=debug GOMAXPROCS=1" "Running with all debugging features..."
                ;;
            *)
                error "Unknown debug type: ${2}"
                exit 1
                ;;
        esac
        ;;
    "test")
        log "Running tests with coverage and memory profiling..."
        go test -tags ${BUILD_TAGS} -race -coverprofile=coverage.out ./...
        if [ $? -eq 0 ]; then
            go tool cover -html=coverage.out -o coverage.html
            success "Tests completed. Coverage report: coverage.html"
        else
            error "Tests failed"
            exit 1
        fi
        ;;
    "bench")
        log "Running benchmarks..."
        go test -tags ${BUILD_TAGS} -bench=. -benchmem ./...
        success "Benchmarks completed"
        ;;
    "clean")
        log "Cleaning build artifacts..."
        rm -rf "${BUILD_DIR}"
        rm -f "${BINARY_NAME}" "${BINARY_NAME}.exe" "${BINARY_NAME}"-*
        rm -f coverage.out coverage.html
        rm -f cpu.prof mem.prof
        success "Clean completed"
        ;;
    "deps")
        log "Installing and verifying dependencies..."
        go mod download
        go mod verify
        go mod tidy
        success "Dependencies updated"
        ;;
    "format")
        log "Formatting code..."
        go fmt ./...
        if command -v goimports &> /dev/null; then
            goimports -w .
        fi
        success "Code formatted"
        ;;
    "lint")
        log "Running linters..."
        go vet ./...
        if command -v golangci-lint &> /dev/null; then
            golangci-lint run
        else
            warn "golangci-lint not found. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        fi
        success "Linting completed"
        ;;
    "help"|"--help"|"-h")
        show_help
        ;;
    *)
        error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac