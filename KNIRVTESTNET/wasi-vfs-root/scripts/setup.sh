#!/bin/bash
# KNIRV Wizened Environment Setup Script
# This script is run by Wizer ONCE at build time.

echo "🔧 Configuring KNIRV Wizened Environment inside WASM..."

# Set up the environment variables for the pre-packed toolchains.
# These paths are relative to the root of the virtual filesystem.
export GOROOT="/usr/lib/go"
export GOPATH="/go-workspace"
export CARGO_HOME="/usr/lib/cargo"
export RUSTUP_HOME="/usr/lib/rustup"

# The most important part: Add our VFS bin to the PATH.
export PATH="/bin:/usr/lib/go/bin:/usr/lib/cargo/bin:$PATH"

# KNIRV-specific environment variables
export KNIRV_ENV="wizened-wasm"
export KNIRV_TOOLCHAIN_PATH="/bin"
export KNIRV_WORKSPACE="/go-workspace"

# Configure Python environment
export PYTHONPATH="/lib/python"
export PYTHON_EXECUTABLE="/bin/python"

echo "✅ WASM environment configured."
echo "PATH is now: $PATH"
echo "KNIRV environment: $KNIRV_ENV"
echo "Python executable: $PYTHON_EXECUTABLE"
echo "Go workspace: $KNIRV_WORKSPACE"
