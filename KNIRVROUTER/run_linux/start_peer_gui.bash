#!/bin/bash
set -x  # Print commands for debugging

# go back to the root directory
cd ../

echo "Starting desktop peer GUI..."
echo "Current directory: $(pwd)"

# Create a directory for the peer if it doesn't exist
mkdir -p peer_gui
echo "Created peer_gui directory"

# Define the file path
file_path="constants/constants.go"

# Use sed to search and replace the value
sed -i 's/\(BLOCKCHAIN_DB_PATH\s*=\s*"\)[^\/]*\/knirvdb"/\1peer_gui\/knirvdb"/' "$file_path"
echo "Updated database path in constants.go"

# Set environment variables to force software rendering
# This helps avoid issues with missing GPU drivers or X11 extensions
export FYNE_RENDERER=software
export FYNE_SCALE=1.0
export FYNE_THEME=light  # Use light theme for better visibility
export DISPLAY=:0        # Ensure display is set

echo "Environment variables set:"
echo "FYNE_RENDERER=$FYNE_RENDERER"
echo "FYNE_SCALE=$FYNE_SCALE"
echo "FYNE_THEME=$FYNE_THEME"
echo "DISPLAY=$DISPLAY"

# Try to start the desktop GUI using the gui package directly
echo "Attempting to launch desktop GUI..."
go run cmd/desktop_gui/main.go

