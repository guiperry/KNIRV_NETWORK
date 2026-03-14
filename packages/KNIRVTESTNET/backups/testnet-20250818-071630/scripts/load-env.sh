#!/bin/bash
# KNIRV Environment Loader
# Sources all necessary environment files for toolchains

# Load KNIRV environment if it exists
if [ -f ~/.knirv-env ]; then
    source ~/.knirv-env
fi

# Load Cargo environment if it exists
if [ -f ~/.cargo/env ]; then
    source ~/.cargo/env
fi

# Load bashrc if it exists
if [ -f ~/.bashrc ]; then
    source ~/.bashrc 2>/dev/null || true
fi

# Explicitly set PATH with all toolchain locations
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.local/go/bin:$PATH"
export GOROOT="$HOME/.local/go"
export GOPATH="$HOME/.local/go-workspace"

# Verify toolchains are available
echo "🔧 KNIRV Environment Loaded"
echo "=========================="

if command -v go &> /dev/null; then
    echo "✅ Go: $(go version)"
else
    echo "❌ Go not found in PATH"
fi

if command -v rustc &> /dev/null; then
    echo "✅ Rust: $(rustc --version)"
else
    echo "❌ Rust not found in PATH"
fi

if command -v cargo &> /dev/null; then
    echo "✅ Cargo: $(cargo --version)"
else
    echo "❌ Cargo not found in PATH"
fi

echo "PATH: $PATH"
