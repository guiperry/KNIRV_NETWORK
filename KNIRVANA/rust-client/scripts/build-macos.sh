#!/bin/bash

# This script packages the KNIRVANA Rust client for macOS.
set -e

echo "Packaging for macOS..."

# Copy the built binary to the packaging directory
cp target/release/knirvana_game packaging/macos/

echo "macOS packaging complete."