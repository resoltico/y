# Otsu Obliterator
.PHONY: all build run clean deps test help
.DEFAULT_GOAL := help

# Delegate to build.sh for actual work
all: build

build:
	@./build.sh build

build-profile:
	@./build.sh build profile

run:
	@./build.sh run

debug-safe:
	@./build.sh debug safe

debug-all:
	@./build.sh debug all

test:
	@./build.sh test

clean:
	@./build.sh clean

deps:
	@./build.sh deps

# Cross-compilation targets
build-windows:
	@./build.sh build windows

build-macos:
	@./build.sh build macos

build-macos-arm64:
	@./build.sh build macos-arm64

build-linux:
	@./build.sh build linux

# Development
dev: deps build-profile

format:
	@go fmt ./...

vet:
	@go vet ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "golangci-lint not found, skipping"; fi

help:
	@echo "Otsu Obliterator"
	@echo ""
	@echo "Primary targets:"
	@echo "  build           Build application"
	@echo "  run             Build and run"
	@echo "  test            Run tests"
	@echo "  clean           Clean artifacts"
	@echo "  deps            Install dependencies"
	@echo ""
	@echo "Debug:"
	@echo "  debug-safe      Safe debugging"
	@echo "  debug-all       Full debugging"
	@echo ""
	@echo "Cross-compile:"
	@echo "  build-windows   Windows build"
	@echo "  build-macos     macOS Intel build"
	@echo "  build-linux     Linux build"
	@echo ""
	@echo "For more options: ./build.sh help"