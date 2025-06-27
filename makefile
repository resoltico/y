# Otsu Obliterator - Memory-Aware Image Processing
.PHONY: all build run clean deps test help bench lint format profile package
.DEFAULT_GOAL := help

# Performance and memory targets
all: deps lint test build

build:
	@./build.sh build

build-profile:
	@./build.sh build profile

build-debug:
	@./build.sh build debug

run:
	@./build.sh run

debug:
	@./build.sh debug basic

debug-memory:
	@./build.sh debug memory

debug-all:
	@./build.sh debug all

test:
	@./build.sh test

bench:
	@./build.sh bench

clean:
	@./build.sh clean

deps:
	@./build.sh deps

format:
	@./build.sh format

lint:
	@./build.sh lint

# Cross-compilation targets
build-windows:
	@./build.sh build windows

build-macos:
	@./build.sh build macos

build-macos-arm64:
	@./build.sh build macos-arm64

build-linux:
	@./build.sh build linux

# Packaging targets
package:
	@./build.sh package

package-windows:
	@./build.sh package windows

package-macos:
	@./build.sh package macos

package-linux:
	@./build.sh package linux

# Development workflow
dev: deps format lint test build-profile

# Performance analysis
profile: build-profile
	@echo "Building with profiling enabled. Use 'make run' to execute with memory tracking."

# Memory leak detection
memcheck: build-debug
	@echo "Running with memory debugging. Monitor output for MatProfile statistics."
	@./build.sh debug memory

# Release preparation
release: deps format lint test
	@./build.sh build windows
	@./build.sh build macos
	@./build.sh build macos-arm64
	@./build.sh build linux
	@echo "Release builds completed in build/ directory"

# Distribution packages
dist: deps format lint test
	@./build.sh package windows
	@./build.sh package macos
	@./build.sh package linux
	@echo "Distribution packages completed in build/ directory"

help:
	@echo "Otsu Obliterator - Memory-Aware Image Processing"
	@echo ""
	@echo "Development targets:"
	@echo "  build           Build application for current platform"
	@echo "  run             Build and run with memory tracking"
	@echo "  test            Run tests with coverage analysis"
	@echo "  bench           Execute performance benchmarks"
	@echo "  dev             Complete development workflow"
	@echo ""
	@echo "Memory and performance:"
	@echo "  build-profile   Build with memory profiling"
	@echo "  debug-memory    Run with memory leak detection"
	@echo "  memcheck        Monitor Mat object lifecycle"
	@echo "  profile         Enable CPU and memory profiling"
	@echo ""
	@echo "Code quality:"
	@echo "  format          Format code and organize imports"
	@echo "  lint            Run static analysis"
	@echo "  clean           Remove build artifacts"
	@echo ""
	@echo "Cross-platform builds:"
	@echo "  build-windows   Windows x64 executable"
	@echo "  build-macos     macOS Intel executable"
	@echo "  build-linux     Linux x64 executable"
	@echo "  release         Build all platform targets"
	@echo ""
	@echo "Distribution packaging:"
	@echo "  package         Package for current platform"
	@echo "  package-windows Create Windows installer"
	@echo "  package-macos   Create macOS application bundle"
	@echo "  package-linux   Create Linux distribution package"
	@echo "  dist            Create all distribution packages"
	@echo ""
	@echo "For detailed options: ./build.sh help"