#!/bin/bash

# Otsu Obliterator Build Script
set -e

BINARY_NAME="otsu-obliterator"
VERSION="1.0.0"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_help() {
    echo "Otsu Obliterator Build Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  build           Build production binary"
    echo "  build-profile   Build with Mat profiling"
    echo "  run             Run production build"
    echo "  run-profile     Run with profiling server"
    echo "  debug [type]    Run with specific debug mode"
    echo "  test            Run tests"
    echo "  clean           Clean build artifacts"
    echo "  deps            Install dependencies"
    echo "  cross [os]      Cross-compile for target OS"
    echo ""
    echo "Debug types:"
    echo "  format          Image format detection"
    echo "  gui             GUI events"
    echo "  algorithms      Algorithm execution"
    echo "  memory          Memory usage"
    echo "  triclass        Iterative triclass algorithm"
    echo "  safe            Safe debugging (no pixel analysis)"
    echo "  all             All debugging enabled"
    echo ""
    echo "Cross-compile targets:"
    echo "  windows         Windows (amd64)"
    echo "  macos           macOS (Intel)"
    echo "  macos-arm64     macOS (Apple Silicon)"
    echo "  linux           Linux (amd64)"
    echo "  universal       macOS universal binary"
    echo ""
    echo "Examples:"
    echo "  $0 build-profile"
    echo "  $0 debug format"
    echo "  $0 cross windows"
}

check_deps() {
    echo -e "${BLUE}Checking dependencies...${NC}"
    
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go not found. Please install Go 1.24+${NC}"
        exit 1
    fi
    
    if ! pkg-config --exists opencv4 && ! pkg-config --exists opencv; then
        echo -e "${RED}Error: OpenCV not found. Please install OpenCV 4.11.0+${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}Dependencies OK${NC}"
}

build_binary() {
    local build_tags="$1"
    local output="$2"
    local ldflags="-s -w"
    
    echo -e "${BLUE}Building ${output}...${NC}"
    
    if [ -n "$build_tags" ]; then
        go build -tags "$build_tags" -ldflags="$ldflags" -o "$output" .
    else
        go build -ldflags="$ldflags" -o "$output" .
    fi
    
    echo -e "${GREEN}Build complete: ${output}${NC}"
}

run_with_env() {
    local env_vars="$1"
    local binary="$2"
    local message="$3"
    
    echo -e "${YELLOW}$message${NC}"
    
    if [ -n "$env_vars" ]; then
        env $env_vars go run -tags matprofile .
    else
        ./"$binary"
    fi
}

setup_debug_env() {
    local debug_type="$1"
    
    case "$debug_type" in
        format)
            echo "OTSU_DEBUG_FORMAT=true"
            ;;
        gui)
            echo "OTSU_DEBUG_GUI=true"
            ;;
        algorithms)
            echo "OTSU_DEBUG_ALGORITHMS=true"
            ;;
        memory)
            echo "OTSU_DEBUG_MEMORY=true"
            ;;
        triclass)
            echo "OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_PIXELS=true"
            ;;
        safe)
            echo "OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_IMAGE=true OTSU_DEBUG_FORMAT=true"
            ;;
        all)
            echo "OTSU_DEBUG_FORMAT=true OTSU_DEBUG_IMAGE=true OTSU_DEBUG_MEMORY=true OTSU_DEBUG_PERFORMANCE=true OTSU_DEBUG_GUI=true OTSU_DEBUG_ALGORITHMS=true OTSU_DEBUG_TRICLASS=true OTSU_DEBUG_PIXELS=true"
            ;;
        *)
            echo -e "${RED}Unknown debug type: $debug_type${NC}"
            echo "Available types: format, gui, algorithms, memory, triclass, safe, all"
            exit 1
            ;;
    esac
}

cross_compile() {
    local target="$1"
    
    case "$target" in
        windows)
            echo -e "${BLUE}Building for Windows...${NC}"
            GOOS=windows GOARCH=amd64 go build -tags matprofile -ldflags="-s -w" -o "${BINARY_NAME}.exe" .
            ;;
        macos)
            echo -e "${BLUE}Building for macOS (Intel)...${NC}"
            GOOS=darwin GOARCH=amd64 go build -tags matprofile -ldflags="-s -w" -o "${BINARY_NAME}-macos-amd64" .
            ;;
        macos-arm64)
            echo -e "${BLUE}Building for macOS (Apple Silicon)...${NC}"
            GOOS=darwin GOARCH=arm64 go build -tags matprofile -ldflags="-s -w" -o "${BINARY_NAME}-macos-arm64" .
            ;;
        linux)
            echo -e "${BLUE}Building for Linux...${NC}"
            GOOS=linux GOARCH=amd64 go build -tags matprofile -ldflags="-s -w" -o "${BINARY_NAME}-linux-amd64" .
            ;;
        universal)
            echo -e "${BLUE}Building universal macOS binary...${NC}"
            GOOS=darwin GOARCH=arm64 go build -tags matprofile -ldflags="-s -w" -o "${BINARY_NAME}-arm64" .
            GOOS=darwin GOARCH=amd64 go build -tags matprofile -ldflags="-s -w" -o "${BINARY_NAME}-x86_64" .
            lipo -create -output "${BINARY_NAME}-macos-universal" "${BINARY_NAME}-arm64" "${BINARY_NAME}-x86_64"
            rm -f "${BINARY_NAME}-arm64" "${BINARY_NAME}-x86_64"
            echo -e "${GREEN}Universal binary created: ${BINARY_NAME}-macos-universal${NC}"
            ;;
        *)
            echo -e "${RED}Unknown target: $target${NC}"
            echo "Available targets: windows, macos, macos-arm64, linux, universal"
            exit 1
            ;;
    esac
}

main() {
    case "${1:-}" in
        build)
            check_deps
            build_binary "" "$BINARY_NAME"
            ;;
        build-profile)
            check_deps
            build_binary "matprofile" "$BINARY_NAME"
            ;;
        run)
            if [ ! -f "$BINARY_NAME" ]; then
                build_binary "" "$BINARY_NAME"
            fi
            ./"$BINARY_NAME"
            ;;
        run-profile)
            echo -e "${YELLOW}Starting with profiling server on :6060${NC}"
            echo -e "${YELLOW}Memory profiler: http://localhost:6060/debug/pprof/${NC}"
            echo -e "${YELLOW}Mat profiling: http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat${NC}"
            go run -tags matprofile .
            ;;
        debug)
            if [ -z "$2" ]; then
                echo -e "${RED}Debug type required${NC}"
                echo "Usage: $0 debug [type]"
                echo "Types: format, gui, algorithms, memory, triclass, safe, all"
                exit 1
            fi
            debug_env=$(setup_debug_env "$2")
            echo -e "${YELLOW}Running with $2 debugging...${NC}"
            env $debug_env go run -tags matprofile .
            ;;
        test)
            echo -e "${BLUE}Running tests...${NC}"
            go test -tags matprofile ./...
            ;;
        clean)
            echo -e "${BLUE}Cleaning build artifacts...${NC}"
            rm -f "$BINARY_NAME" "${BINARY_NAME}.exe" "${BINARY_NAME}"-*
            echo -e "${GREEN}Clean complete${NC}"
            ;;
        deps)
            echo -e "${BLUE}Installing dependencies...${NC}"
            go mod tidy
            go mod download
            echo -e "${GREEN}Dependencies installed${NC}"
            ;;
        cross)
            if [ -z "$2" ]; then
                echo -e "${RED}Target platform required${NC}"
                echo "Usage: $0 cross [target]"
                echo "Targets: windows, macos, macos-arm64, linux, universal"
                exit 1
            fi
            check_deps
            cross_compile "$2"
            ;;
        help|--help|-h)
            print_help
            ;;
        "")
            print_help
            ;;
        *)
            echo -e "${RED}Unknown command: $1${NC}"
            print_help
            exit 1
            ;;
    esac
}

main "$@"