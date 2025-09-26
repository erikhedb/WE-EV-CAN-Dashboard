#!/bin/bash

# Development start script for WE-EV-CAN-Dashboard UI
# This script sets up the correct paths and starts the UI server

# Get the directory where this script is located (project root)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Set config path relative to project root (use dev config)
CONFIG_PATH="$SCRIPT_DIR/config-dev.yaml"

echo "Starting UI server from project root: $SCRIPT_DIR"
echo "Using config: $CONFIG_PATH"

# Change to project root directory
cd "$SCRIPT_DIR"

# Check if config exists
if [ ! -f "$CONFIG_PATH" ]; then
    echo "Error: Config file not found at $CONFIG_PATH"
    exit 1
fi

# Run the UI server with the config path
echo "Starting UI server in development mode..."
go run ./cmd/ui/main.go -config="$CONFIG_PATH"