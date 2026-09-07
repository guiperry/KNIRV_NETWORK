#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ULORA_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== Building CPU-only MLC Engine ==="
echo "Per ulora_implementation.md §1.5 — TVM Unity (LLVM backend) + mlc-llm"
echo "Targeting CPU-only (q4f32_1/q0f32 quantization). No CUDA/Vulkan."
echo ""

THIRD_PARTY="$ULORA_ROOT/third_party"
mkdir -p "$THIRD_PARTY"

# Step A: TVM Unity, LLVM backend only, no CUDA/Vulkan.
if [ ! -d "$THIRD_PARTY/tvm-unity" ]; then
    echo "Cloning TVM Unity (relax)..."
    git clone --recursive https://github.com/mlc-ai/relax.git "$THIRD_PARTY/tvm-unity"
fi

TVM_BUILD="$THIRD_PARTY/tvm-unity/build"
mkdir -p "$TVM_BUILD"
cd "$TVM_BUILD"

if [ ! -f "Makefile" ]; then
    cp ../cmake/config.cmake .
    echo 'set(USE_LLVM "llvm-config --ignore-libllvm --link-static")' >> config.cmake
    echo 'set(USE_CUDA OFF)' >> config.cmake
    echo 'set(USE_VULKAN OFF)' >> config.cmake
    echo "Configuring TVM Unity..."
    cmake ..
fi

echo "Building TVM Unity (this may take 20-40 minutes)..."
cmake --build . --parallel "$(nproc)"

# Step B: mlc-llm itself, same CPU-only constraint.
if [ ! -d "$THIRD_PARTY/mlc-llm" ]; then
    echo "Cloning mlc-llm..."
    git clone --recursive https://github.com/mlc-ai/mlc-llm.git "$THIRD_PARTY/mlc-llm"
fi

MLC_BUILD="$THIRD_PARTY/mlc-llm/build"
mkdir -p "$MLC_BUILD"
cd "$MLC_BUILD"

if [ ! -f "Makefile" ]; then
    cp ../cmake/config.cmake .
    echo 'set(USE_CUDA OFF)' >> config.cmake
    echo "Configuring mlc-llm..."
    cmake ..
fi

echo "Building mlc-llm (this may take 20-40 minutes)..."
cmake --build . --parallel "$(nproc)"

echo ""
echo "=== CPU-only MLC Engine build complete ==="
echo "Artifacts in: $THIRD_PARTY/mlc-llm/build/"
echo "Quantization: use q4f32_1/q0f32 for CPU targets"
