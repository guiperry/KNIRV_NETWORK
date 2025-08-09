#!/bin/bash

# Build script for KNIRVROUTER

echo "Building KNIRVROUTER..."
go build -o KNIRVROUTER

if [ $? -eq 0 ]; then
    echo "Build successful! Binary created: KNIRVROUTER"
    echo ""
    echo "Usage:"
    echo "  ./KNIRVROUTER             # Start the desktop GUI (default)"
    echo "  ./KNIRVROUTER -webgui     # Start the web GUI"
    echo "  ./KNIRVROUTER -port 5001  # Start the web GUI on a custom port"
    echo "  ./KNIRVROUTER -chain      # Start in blockchain mode"
    echo "  ./KNIRVROUTER -wallet     # Start in wallet mode"
    echo ""
    echo "Making the binary executable..."
    chmod +x KNIRVROUTER
    echo "Done!"
else
    echo "Build failed!"
fi