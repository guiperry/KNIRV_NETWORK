#!/bin/bash

# Script to copy the .env file to the Electron app's resources directory
# This ensures the Electron app can find the environment variables

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Navigate to the project root directory (parent of scripts)
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to the project root directory
cd "$PROJECT_ROOT"

echo "Working in directory: $(pwd)"

# Check if .env exists
if [ ! -f .env ]; then
    echo "❌ Error: .env file not found in the project root directory."
    echo "Please create a .env file first or run scripts/setup-env.sh to create one from the template."
    exit 1
fi

# Define the Electron app paths relative to the project root
ELECTRON_DIR="electron/dist/linux-unpacked"
RESOURCES_DIR="$ELECTRON_DIR/resources"


# Check if the Electron app is built
if [ ! -d "$ELECTRON_DIR" ]; then
    echo "❌ Error: Electron app directory not found at $ELECTRON_DIR"
    echo "Please build the Electron app first."
    exit 1
fi

# Create directories if they don't exist
mkdir -p "$RESOURCES_DIR"


# Copy the .env file to multiple locations to ensure it's found
cp .env "$RESOURCES_DIR/.env"

# Copy the default.env file as well
if [ -f default.env ]; then
    cp default.env "$RESOURCES_DIR/default.env"
    echo "✅ Copied .env and default.env files to Electron app resources directory:"
    echo "  - $RESOURCES_DIR/.env"
    echo "  - $RESOURCES_DIR/utils/default.env"
else
    echo "✅ Copied .env file to Electron app resources directory:"
    echo "  - $RESOURCES_DIR/.env"
    echo "⚠️  Warning: default.env file not found, skipping copy."
fi

echo ""
echo "🚀 The Electron app should now be able to find your environment variables."