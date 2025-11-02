# Makefile for WE-EV-CAN-Dashboard
# Author: Erik Wästlin
# Standard Go build/test/lint targets for multi-cmd repo

# Core binaries to build
CORE_BINARIES = reader handler replay
# Tool binaries
TOOL_BINARIES = tools/raw-convert tools/raw-analysis tools/log-analysis
# UI binary (special handling)
UI_BINARY = ui

SUDO ?= sudo
SYSTEMCTL ?= systemctl
INSTALL_DIR ?= /opt/wecan
SYSTEMD_DIR ?= /etc/systemd/system
SERVICE_USER ?= erwa
SERVICE_GROUP ?= erwa
SERVICE_UNITS = canbus-reader.service canbus-handler.service canbus-ui.service
SERVICE_UNITS_STOP = canbus-ui.service canbus-handler.service canbus-reader.service

.PHONY: all build build-core build-tools build-ui test lint clean \
	install install-binaries install-services install-config install-data stop-services \
	start-services check-services

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

install: build
	@echo "Starting installation workflow..."
	@$(MAKE) stop-services
	@$(MAKE) install-binaries
	@$(MAKE) install-config
	@$(MAKE) install-data
	@$(MAKE) install-services
	@$(MAKE) start-services
	@$(MAKE) check-services
	@echo "Installation complete."

stop-services:
	@echo "Stopping systemd services (best effort)..."
	@for svc in $(SERVICE_UNITS_STOP); do \
		echo "Stopping $$svc..."; \
		if $(SUDO) $(SYSTEMCTL) stop $$svc; then \
			echo "$$svc stopped."; \
		else \
			echo "Warning: could not stop $$svc (may not be active)."; \
		fi; \
	done

install-binaries:
	@echo "Installing binaries into $(INSTALL_DIR)/bin..."
	@$(SUDO) install -d $(INSTALL_DIR)/bin
	@for binary in $(CORE_BINARIES); do \
		echo "Installing $$binary..."; \
		$(SUDO) install -m 755 bin/$$binary $(INSTALL_DIR)/bin/$$binary; \
	done
	@for tool in $(TOOL_BINARIES); do \
		tool_name=$$(basename $$tool); \
		echo "Installing $$tool_name..."; \
		$(SUDO) install -m 755 bin/$$tool_name $(INSTALL_DIR)/bin/$$tool_name; \
	done
	@echo "Installing UI binary and assets..."
	@$(SUDO) install -d $(INSTALL_DIR)/bin/ui
	@$(SUDO) install -m 755 bin/ui/ui $(INSTALL_DIR)/bin/ui/ui
	@$(SUDO) rm -rf $(INSTALL_DIR)/bin/ui/static
	@$(SUDO) cp -r bin/ui/static $(INSTALL_DIR)/bin/ui/

install-services:
	@echo "Installing systemd unit files into $(SYSTEMD_DIR)..."
	@for service in $(SERVICE_UNITS); do \
		echo "Processing $$service..."; \
		tmp_file=$$(mktemp); \
		cp $$service $$tmp_file; \
		case $$service in \
			canbus-reader.service) \
				sed -i -e 's|^WorkingDirectory=.*|WorkingDirectory=$(INSTALL_DIR)|' \
				       -e 's|^ExecStart=.*|ExecStart=$(INSTALL_DIR)/bin/reader|' $$tmp_file; \
				;; \
			canbus-handler.service) \
				sed -i -e 's|^WorkingDirectory=.*|WorkingDirectory=$(INSTALL_DIR)|' \
				       -e 's|^ExecStart=.*|ExecStart=$(INSTALL_DIR)/bin/handler|' $$tmp_file; \
				;; \
			canbus-ui.service) \
				sed -i -e 's|^WorkingDirectory=.*|WorkingDirectory=$(INSTALL_DIR)|' \
				       -e 's|^ExecStart=.*|ExecStart=$(INSTALL_DIR)/bin/ui/ui|' $$tmp_file; \
				;; \
		esac; \
		$(SUDO) install -m 644 $$tmp_file $(SYSTEMD_DIR)/$$service; \
		rm -f $$tmp_file; \
	done
	@$(SUDO) $(SYSTEMCTL) daemon-reload

install-config:
	@echo "Ensuring configuration files exist in $(INSTALL_DIR)..."
	@$(SUDO) install -d -m 775 -o $(SERVICE_USER) -g $(SERVICE_GROUP) $(INSTALL_DIR)
	@$(SUDO) install -d -m 775 -o $(SERVICE_USER) -g $(SERVICE_GROUP) $(INSTALL_DIR)/logs
	@$(SUDO) sh -c 'touch $(INSTALL_DIR)/logs/reader_service.log'
	@$(SUDO) chown $(SERVICE_USER):$(SERVICE_GROUP) $(INSTALL_DIR)/logs/reader_service.log
	@$(SUDO) chmod 664 $(INSTALL_DIR)/logs/reader_service.log
	@$(SUDO) sh -c 'touch $(INSTALL_DIR)/logs/canbus.json'
	@$(SUDO) chown $(SERVICE_USER):$(SERVICE_GROUP) $(INSTALL_DIR)/logs/canbus.json
	@$(SUDO) chmod 664 $(INSTALL_DIR)/logs/canbus.json
	@if [ -f config.yaml ]; then \
		if [ -f $(INSTALL_DIR)/config.yaml ]; then \
			echo "config.yaml already exists at $(INSTALL_DIR); leaving it in place."; \
		else \
			echo "Installing config.yaml..."; \
			$(SUDO) install -m 644 config.yaml $(INSTALL_DIR)/config.yaml; \
		fi; \
	else \
		echo "Warning: config.yaml not found in repository; skipping."; \
	fi
	@if [ -f config-dev.yaml ]; then \
		if [ -f $(INSTALL_DIR)/config-dev.yaml ]; then \
			echo "config-dev.yaml already exists at $(INSTALL_DIR); leaving it in place."; \
		else \
			echo "Installing config-dev.yaml..."; \
			$(SUDO) install -m 644 config-dev.yaml $(INSTALL_DIR)/config-dev.yaml; \
		fi; \
		else \
			echo "config-dev.yaml not found in repository; skipping."; \
		fi

install-data:
	@echo "Installing JSON data assets into $(INSTALL_DIR)/data..."
	@if ls data/*.json >/dev/null 2>&1; then \
		$(SUDO) install -d -m 775 -o $(SERVICE_USER) -g $(SERVICE_GROUP) $(INSTALL_DIR)/data; \
		for json in data/*.json; do \
			base=$$(basename $$json); \
			if [ -f $(INSTALL_DIR)/data/$$base ]; then \
				echo "$$base already exists at $(INSTALL_DIR)/data; leaving it in place."; \
			else \
				echo "Installing $$base..."; \
				$(SUDO) install -m 664 -o $(SERVICE_USER) -g $(SERVICE_GROUP) $$json $(INSTALL_DIR)/data/$$base; \
			fi; \
		done; \
	else \
		echo "Warning: no JSON files found in data/; skipping."; \
	fi

start-services:
	@echo "Starting and enabling systemd services..."
	@for svc in $(SERVICE_UNITS); do \
		echo "Enabling $$svc..."; \
		$(SUDO) $(SYSTEMCTL) enable $$svc; \
		echo "Starting $$svc..."; \
		$(SUDO) $(SYSTEMCTL) start $$svc; \
	done

check-services:
	@echo "Verifying service status..."
	@for svc in $(SERVICE_UNITS); do \
		if $(SUDO) $(SYSTEMCTL) is-active --quiet $$svc; then \
			echo "$$svc is active."; \
		else \
			echo "Error: $$svc failed to start. Showing status output:"; \
			$(SUDO) $(SYSTEMCTL) status $$svc --no-pager || true; \
			exit 1; \
		fi; \
	done

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

log-analysis: bin/
	go build -o bin/log-analysis ./cmd/tools/log-analysis

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
