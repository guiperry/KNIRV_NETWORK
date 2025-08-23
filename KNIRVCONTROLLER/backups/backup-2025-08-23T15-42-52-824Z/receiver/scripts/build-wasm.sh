#!/bin/bash

# Build script for KNIRV-CORTEX WASM modules
set -e

echo "Building KNIRV-CORTEX WASM modules..."

# Navigate to the rust-wasm directory
cd "$(dirname "$0")/../rust-wasm"

# Build the WASM package
echo "Building WASM package with wasm-pack..."
wasm-pack build --target web --out-dir ../src/wasm-pkg --scope knirv

# Navigate back to project root
cd ..

echo "WASM build complete!"
echo "Generated files in src/wasm-pkg/"

# List generated files
ls -la src/wasm-pkg/
