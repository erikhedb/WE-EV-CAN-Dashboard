# Makefile for WE-EV-CAN-Dashboard
# Author: Erik Wästlin
# Standard Go build/test/lint targets for multi-cmd repo

# Binaries to build (add more as needed)
BINARIES = reader handler replay tools/raw-convert

.PHONY: all build test lint clean

all: build

build:
	@echo "Building all binaries..."
	@for dir in $(BINARIES); do \
		cd cmd/$$dir 2>/dev/null || cd cmd/$$dir/..; \
		if [ -f main.go ]; then \
			go build -o ../../bin/$$dir . ; \
			cd - >/dev/null; \
		fi \
	done

bin/:
	@mkdir -p bin

test:
	@echo "Running all tests..."
	go test ./...

lint:
	@echo "Running golangci-lint..."
	@golangci-lint run ./...

clean:
	@echo "Cleaning binaries..."
	rm -rf bin/*

# Usage:
#   make build   # Build all binaries into ./bin
#   make test    # Run all Go tests
#   make lint    # Run linter (requires golangci-lint)
#   make clean   # Remove built binaries
