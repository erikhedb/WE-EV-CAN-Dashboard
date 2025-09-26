#!/bin/bash

# Production start script for WE-EV-CAN-Dashboard UI
# This script starts the built UI binary with the correct config

# Get the directory where this script is located (project root)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Set config path relative to project root
CONFIG_PATH="$SCRIPT_DIR/config.yaml"

echo "Starting UI server from project root: $SCRIPT_DIR"
echo "Using config: $CONFIG_PATH"

# Change to project root directory
cd "$SCRIPT_DIR"

# Check if config exists
if [ ! -f "$CONFIG_PATH" ]; then
    echo "Error: Config file not found at $CONFIG_PATH"
    exit 1
fi

# Check if UI binary exists
if [ ! -f "$SCRIPT_DIR/bin/ui/ui" ]; then
    echo "Error: UI binary not found. Please run 'make build-ui' first."
    exit 1
fi

# Run the UI server with the config path
echo "Starting UI server in production mode..."
"$SCRIPT_DIR/bin/ui/ui" -config="$CONFIG_PATH"