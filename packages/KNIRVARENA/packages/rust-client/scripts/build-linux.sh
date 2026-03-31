#!/bin/bash

# This script packages the KNIRVANA Rust client for Linux.
set -e

echo "Packaging for Linux..."

# Copy the built binary to the packaging directory
cp target/release/knirvana_game packaging/linux/

echo "Linux packaging complete."