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
        "profile")
            extra_flags="-tags ${BUILD_TAGS}"
            log "Building with profiling..."
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
            build
            "./${BUILD_DIR}/${BINARY_NAME}"
        fi
    fi
}

show_help() {
    cat << EOF
Usage: $0 [command] [options]

Commands:
  build [target]      Build binary (default, profile, windows, macos, macos-arm64, linux)
  run                 Build and run application
  debug [type]        Run with debug flags (safe, all, format, gui, algorithms)
  test                Run tests
  clean               Clean build artifacts
  deps                Install dependencies
  help                Show this help

Examples:
  $0 build profile    Build with profiling
  $0 debug safe       Run with safe debugging
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
        run_with_env "" "Running ${BINARY_NAME}..."
        ;;
    "debug")
        case "${2:-safe}" in
            "safe")
                run_with_env "OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_IMAGE=true OTSU_DEBUG_FORMAT=true" "Running with safe debugging..."
                ;;
            "all")
                run_with_env "OTSU_DEBUG_FORMAT=true OTSU_DEBUG_IMAGE=true OTSU_DEBUG_MEMORY=true OTSU_DEBUG_PERFORMANCE=true OTSU_DEBUG_GUI=true OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_TRICLASS=true" "Running with all debugging..."
                ;;
            "format")
                run_with_env "OTSU_DEBUG_FORMAT=true" "Running with format debugging..."
                ;;
            "gui")
                run_with_env "OTSU_DEBUG_GUI=true" "Running with GUI debugging..."
                ;;
            "algorithms")
                run_with_env "OTSU_DEBUG_ALGORITHMS=true" "Running with algorithm debugging..."
                ;;
            *)
                error "Unknown debug type: ${2}"
                exit 1
                ;;
        esac
        ;;
    "test")
        log "Running tests..."
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