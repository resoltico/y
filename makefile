# Otsu Obliterator Makefile
.PHONY: all build run clean deps test help
.DEFAULT_GOAL := help

# Delegate all commands to build script
all: build

build:
	@./build.sh build

build-profile:
	@./build.sh build-profile

run:
	@./build.sh run

run-profile:
	@./build.sh run-profile

test:
	@./build.sh test

clean:
	@./build.sh clean

deps:
	@./build.sh deps

# Debug shortcuts
debug-format:
	@./build.sh debug format

debug-safe:
	@./build.sh debug safe

debug-all:
	@./build.sh debug all

# Cross-compilation shortcuts
build-windows:
	@./build.sh cross windows

build-macos:
	@./build.sh cross macos

build-linux:
	@./build.sh cross linux

help:
	@./build.sh help

# Development workflow
dev: deps build-profile
	@echo "Development environment ready. Run 'make run-profile' to start."