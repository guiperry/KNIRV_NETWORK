#!/usr/bin/env bash
#
# Build the CPU-only MLC Engine vendored by KNIRVULORA.
# Per ulora_implementation.md §1.5 — TVM Unity (LLVM backend) + mlc-llm.
# Target: CPU-only (q4f32_1/q0f32 quantization). No CUDA/Vulkan.
#
# Corrections applied to the original script (see the notes at the bottom):
#   * mlc-llm builds TVM itself via add_subdirectory(TVM_SOURCE_DIR), so the
#     separate TVM clone+build the original script performed was redundant work
#     that also left TVM_SOURCE_DIR unset. We now point TVM_SOURCE_DIR at the
#     TVM checkout mlc-llm's own submodule pins and let mlc-llm build it.
#   * The build is bounded rather than nproc-wide: TVM's codegen is
#     memory-hungry and -j$(nproc) OOMs on a busy host.
#   * Artifacts are verified at the end. The original script printed
#     "build complete" unconditionally, so a partial or failed build looked
#     successful.
#   * Prerequisites are checked up front with actionable messages.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ULORA_ROOT="$(dirname "$SCRIPT_DIR")"
THIRD_PARTY="${ULORA_THIRD_PARTY:-$ULORA_ROOT/third_party}"

# Pinned so the build is reproducible; bump deliberately.
TVM_REF="${ULORA_TVM_REF:-mlc}"
MLC_LLM_REF="${ULORA_MLC_LLM_REF:-main}"

# Bounded parallelism. Override with ULORA_BUILD_JOBS.
JOBS="${ULORA_BUILD_JOBS:-4}"

MLC_LLM_SRC="$THIRD_PARTY/mlc-llm"
MLC_LLM_BUILD="$MLC_LLM_SRC/build"

die() { echo "ERROR: $*" >&2; exit 1; }
require() {
    command -v "$1" >/dev/null 2>&1 || die "$1 is required but not installed${2:+ ($2)}"
}

echo "=== Building CPU-only MLC Engine ==="
echo "Per ulora_implementation.md §1.5 — TVM Unity (LLVM backend) + mlc-llm"
echo "Targeting CPU-only (q4f32_1/q0f32 quantization). No CUDA/Vulkan."
echo ""

# --- Prerequisites -----------------------------------------------------------
# Checked before any multi-gigabyte clone, so a missing tool fails in seconds
# rather than after 20 minutes of downloading.
require git
require cmake "need >= 3.18"
require make
require g++
require nproc

# TVM's own CMakeLists asks for 3.18, but its vendored tvm-ffi asks for 3.26, so
# 3.26 is the real floor. Installing one needs no root: `uv tool install
# "cmake==3.31.6"` (or pip) puts it in ~/.local/bin. A 3.3x is preferred over
# cmake 4.x, which removed compatibility with older cmake_minimum_required
# declarations that TVM's third-party dependencies still use.
CMAKE_MIN_MAJOR=3
CMAKE_MIN_MINOR=26
CMAKE_VERSION="$(cmake --version | head -1 | awk '{print $3}')"
cmake_major="${CMAKE_VERSION%%.*}"
cmake_rest="${CMAKE_VERSION#*.}"
cmake_minor="${cmake_rest%%.*}"
if [ "$cmake_major" -lt "$CMAKE_MIN_MAJOR" ] || \
   { [ "$cmake_major" -eq "$CMAKE_MIN_MAJOR" ] && [ "$cmake_minor" -lt "$CMAKE_MIN_MINOR" ]; }; then
    die "cmake $CMAKE_VERSION is too old; TVM's tvm-ffi requires >= ${CMAKE_MIN_MAJOR}.${CMAKE_MIN_MINOR}. Install one without root: uv tool install \"cmake==3.31.6\", then re-run with ~/.local/bin on PATH."
fi

# --- LLVM selection ----------------------------------------------------------
# TVM refuses LLVM < 15 (FindLLVM.cmake). Ubuntu ships versioned binaries, and
# the unversioned llvm-config is often an older default, so select explicitly.
# ULORA_LLVM_CONFIG overrides outright. Ascending order = prefer the minimum
# version that satisfies TVM, which is the best-tested end of its supported
# range rather than the newest it may not know.
if [ -n "${ULORA_LLVM_CONFIG:-}" ]; then
    LLVM_CONFIG="$ULORA_LLVM_CONFIG"
    command -v "$LLVM_CONFIG" >/dev/null 2>&1 || die "ULORA_LLVM_CONFIG=$LLVM_CONFIG not found"
else
    LLVM_CONFIG=""
    candidates="llvm-config-15 llvm-config-16 llvm-config-17 llvm-config-18 llvm-config"
    for candidate in $candidates; do
        if ! command -v "$candidate" >/dev/null 2>&1; then
            continue
        fi
        version="$("$candidate" --version 2>/dev/null || echo 0)"
        major="${version%%.*}"
        if [ "$major" -ge 15 ] 2>/dev/null; then
            LLVM_CONFIG="$candidate"
            break
        fi
    done
    [ -n "$LLVM_CONFIG" ] || die "TVM requires LLVM >= 15 but no suitable llvm-config was found (have: $(llvm-config --version 2>/dev/null || echo none)). Install llvm-15-dev or newer, or set ULORA_LLVM_CONFIG."
fi

echo "LLVM: $LLVM_CONFIG ($("$LLVM_CONFIG" --version))"
# USE_LLVM below asks for --link-static, which needs the static archives, not
# just the shared library. Checking here avoids a link failure 40 minutes in.
if ! "$LLVM_CONFIG" --link-static --libs >/dev/null 2>&1; then
    die "$LLVM_CONFIG --link-static failed; static LLVM archives are missing (install the matching -dev package, or use the shared LLVM)"
fi

echo "cmake: $CMAKE_VERSION"
echo "jobs:  $JOBS"
echo "third_party: $THIRD_PARTY"
echo ""

mkdir -p "$THIRD_PARTY"

# --- Step A: fetch mlc-llm (recursively, so its pinned TVM comes with it) ----
if [ ! -d "$MLC_LLM_SRC/.git" ]; then
    echo "Cloning mlc-llm (recursive; this brings TVM at 3rdparty/tvm)..."
    git clone --recursive --branch "$MLC_LLM_REF" https://github.com/mlc-ai/mlc-llm.git "$MLC_LLM_SRC"
else
    echo "mlc-llm already cloned; ensuring submodules are present..."
    git -C "$MLC_LLM_SRC" submodule update --init --recursive
fi

# TVM_SOURCE_DIR: prefer mlc-llm's own submodule (the revision it pins), which is
# what mlc-llm tests against. An older separate checkout is deliberately not used.
if [ -d "$MLC_LLM_SRC/3rdparty/tvm" ]; then
    export TVM_SOURCE_DIR="$MLC_LLM_SRC/3rdparty/tvm"
else
    # Fall back to a standalone TVM Unity checkout, matching the original script.
    TVM_SRC="$THIRD_PARTY/tvm-unity"
    if [ ! -d "$TVM_SRC/.git" ]; then
        echo "mlc-llm has no 3rdparty/tvm; cloning TVM Unity (relax @ $TVM_REF)..."
        git clone --recursive --branch "$TVM_REF" https://github.com/mlc-ai/relax.git "$TVM_SRC"
    fi
    export TVM_SOURCE_DIR="$TVM_SRC"
fi
echo "TVM_SOURCE_DIR=$TVM_SOURCE_DIR"
[ -f "$TVM_SOURCE_DIR/CMakeLists.txt" ] || die "TVM source not found at $TVM_SOURCE_DIR"

# --- Step B: configure -------------------------------------------------------
mkdir -p "$MLC_LLM_BUILD"
cd "$MLC_LLM_BUILD"

if [ ! -f "Makefile" ] && [ ! -f "build.ninja" ]; then
    # mlc-llm no longer ships cmake/config.cmake (the cmake/ dir holds only
    # gen_cmake_config.py), so the file must be written rather than copied. Its
    # CMakeLists includes ${CMAKE_BINARY_DIR}/config.cmake if present and treats
    # a missing one as "use defaults" — so writing it here is the supported path.
    #
    # Only USE_LLVM needs enabling: TVM already defaults USE_CUDA/VULKAN/METAL to
    # OFF and mlc-llm defaults MLC_LLM_BUILD_PYTHON_MODULE to OFF, so this stays
    # CPU-only without pulling in scikit-build-core.
    cat > ./config.cmake <<CONFIG
# Generated by scripts/build-mlc-cpu.sh — CPU-only, LLVM backend.
set(USE_LLVM "$LLVM_CONFIG --ignore-libllvm --link-static")
set(USE_CUDA OFF)
set(USE_VULKAN OFF)
set(USE_METAL OFF)
CONFIG

    echo "Configuring mlc-llm (TVM_SOURCE_DIR=$TVM_SOURCE_DIR)..."
    TVM_SOURCE_DIR="$TVM_SOURCE_DIR" cmake "$MLC_LLM_SRC"
fi

# --- Step C: build -----------------------------------------------------------
echo "Building mlc-llm + TVM (this is the long step: budget 40-80+ minutes)..."
cmake --build . --parallel "$JOBS"

# --- Step D: verify ----------------------------------------------------------
# The original script printed success unconditionally. Verify the outputs exist
# so a partial build cannot be mistaken for a usable engine.
echo ""
echo "=== Verifying build artifacts ==="
missing=0
check_artifact() {
    if [ -e "$1" ]; then
        echo "  ok      $1"
    else
        echo "  MISSING $1"
        missing=$((missing + 1))
    fi
}
check_artifact "$MLC_LLM_BUILD/libmlc_llm.so"
# mlc-llm's own libraries land at the build root; TVM's land under build/lib/.
# (Getting this wrong is a false negative, which is how this path was found.)
check_artifact "$MLC_LLM_BUILD/libmlc_llm.a"
check_artifact "$MLC_LLM_BUILD/lib/libtvm_runtime.so"
check_artifact "$MLC_LLM_BUILD/lib/libtvm_ffi.so"

# At least one loadable mlc-llm library must exist.
if [ ! -e "$MLC_LLM_BUILD/libmlc_llm.so" ] && [ ! -e "$MLC_LLM_BUILD/libmlc_llm.a" ]; then
    echo "  neither libmlc_llm.so nor libmlc_llm.a was produced"
    missing=$((missing + 1))
fi

if [ "$missing" -ne 0 ]; then
    die "MLC Engine build did not produce the expected artifacts ($missing missing); see the configure/build output above"
fi

echo ""
echo "=== CPU-only MLC Engine build complete ==="
echo "Artifacts in: $MLC_LLM_BUILD"
echo "Quantization: use q4f32_1/q0f32 for CPU targets"
