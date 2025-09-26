# Makefile for WE-EV-CAN-Dashboard
# Author: Erik Wästlin
# Standard Go build/test/lint targets for multi-cmd repo

# Core binaries to build
CORE_BINARIES = reader handler replay
# Tool binaries
TOOL_BINARIES = tools/raw-convert tools/raw-analysis
# UI binary (special handling)
UI_BINARY = ui

.PHONY: all build build-core build-tools build-ui test lint clean

all: build

build: build-core build-tools build-ui

build-core: bin/
	@echo "Building core binaries..."
	@for binary in $(CORE_BINARIES); do \
		echo "Building $$binary..."; \
		go build -o bin/$$binary ./cmd/$$binary; \
	done

build-tools: bin/
	@echo "Building tool binaries..."
	@for tool in $(TOOL_BINARIES); do \
		tool_name=$$(basename $$tool); \
		echo "Building $$tool_name..."; \
		go build -o bin/$$tool_name ./cmd/$$tool; \
	done

build-ui: bin/ui/
	@echo "Building UI binary..."
	go build -o bin/ui/ui ./cmd/ui
	@echo "Copying static assets..."
	cp -r cmd/ui/static bin/ui/
	@echo "UI build complete: bin/ui/ui with static assets in bin/ui/static/"

bin/:
	@mkdir -p bin

bin/ui/:
	@mkdir -p bin/ui

test:
	@echo "Running all tests..."
	go test ./...

lint:
	@echo "Running golangci-lint..."
	@golangci-lint run ./...

clean:
	@echo "Cleaning binaries and build artifacts..."
	@if [ -d bin ]; then rm -rf bin/*; echo "Removed bin/*"; else echo "bin/ directory not found"; fi
	@echo "Cleaning Go module cache..."
	@go clean -cache
	@echo "Clean complete"

# Individual targets for convenience
reader: bin/
	go build -o bin/reader ./cmd/reader

handler: bin/
	go build -o bin/handler ./cmd/handler

replay: bin/
	go build -o bin/replay ./cmd/replay

raw-convert: bin/
	go build -o bin/raw-convert ./cmd/tools/raw-convert

raw-analysis: bin/
	go build -o bin/raw-analysis ./cmd/tools/raw-analysis

ui: build-ui

# Usage:
#   make build        # Build all binaries and UI with assets
#   make build-core   # Build only core binaries (reader, handler, replay)
#   make build-tools  # Build only tool binaries
#   make build-ui     # Build UI binary and copy static assets
#   make ui           # Alias for build-ui
#   make test         # Run all Go tests
#   make lint         # Run linter (requires golangci-lint)
#   make clean        # Remove all built binaries
#   make <binary>     # Build individual binary (e.g., make reader)