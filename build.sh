#!/bin/bash

set -e

# Auto-detect project settings
BINARY_NAME=$(basename "$(pwd)")
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

# Build flags
LDFLAGS="-s -w -X main.version=${VERSION}"
BUILD_TAGS="matprofile"

# Colors
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
    
    if ! pkg-config --exists opencv4 && ! pkg-config --exists opencv; then
        warn "OpenCV not found. Install OpenCV 4.11.0+ for full functionality"
    fi
    
    success "Dependencies checked"
}

build() {
    local target=${1:-"default"}
    local output_name="${BINARY_NAME}"
    local extra_flags=""
    
    case $target in
        "profile"|"debug")
            extra_flags="-tags ${BUILD_TAGS}"
            log "Building with MatProfile memory tracking..."
            ;;
        "windows")
            output_name="${BINARY_NAME}.exe"
            export GOOS=windows GOARCH=amd64
            extra_flags="-tags ${BUILD_TAGS}"
            ;;
        "macos")
            output_name="${BINARY_NAME}-macos-amd64"
            export GOOS=darwin GOARCH=amd64
            extra_flags="-tags ${BUILD_TAGS}"
            ;;
        "macos-arm64")
            output_name="${BINARY_NAME}-macos-arm64"
            export GOOS=darwin GOARCH=arm64
            extra_flags="-tags ${BUILD_TAGS}"
            ;;
        "linux")
            output_name="${BINARY_NAME}-linux-amd64"
            export GOOS=linux GOARCH=amd64
            extra_flags="-tags ${BUILD_TAGS}"
            ;;
        *)
            log "Building ${BINARY_NAME}..."
            ;;
    esac
    
    mkdir -p "${BUILD_DIR}"
    
    if ! go build ${extra_flags} -ldflags="${LDFLAGS}" -o "${BUILD_DIR}/${output_name}" "./${CMD_DIR}"; then
        error "Build failed"
        exit 1
    fi
    
    success "Built: ${BUILD_DIR}/${output_name}"
}

run_with_env() {
    local env_vars="$1"
    local message="$2"
    
    log "$message"
    
    if [ -n "$env_vars" ]; then
        env $env_vars go run -tags ${BUILD_TAGS} "./${CMD_DIR}"
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
  run                 Build and run application with memory tracking
  debug [type]        Run with debug environment variables
  test                Run tests
  clean               Clean build artifacts
  deps                Install dependencies
  help                Show this help

Debug types:
  basic               Basic application debugging (LOG_LEVEL=debug)
  memory              Memory usage monitoring with MatProfile
  all                 All debugging features enabled

Examples:
  $0 build profile    Build with MatProfile memory tracking
  $0 debug memory     Run with memory debugging
  $0 build windows    Cross-compile for Windows
EOF
}

case "${1:-help}" in
    "build")
        check_deps
        build "${2:-default}"
        ;;
    "run")
        check_deps
        # Always build with MatProfile for development
        build "profile"
        export LOG_LEVEL=info
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
        log "Running tests with MatProfile..."
        go test -tags ${BUILD_TAGS} ./...
        success "Tests completed"
        ;;
    "clean")
        log "Cleaning build artifacts..."
        rm -rf "${BUILD_DIR}"
        rm -f "${BINARY_NAME}" "${BINARY_NAME}.exe" "${BINARY_NAME}"-*
        success "Clean completed"
        ;;
    "deps")
        log "Installing dependencies..."
        go mod tidy
        go mod download
        success "Dependencies installed"
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