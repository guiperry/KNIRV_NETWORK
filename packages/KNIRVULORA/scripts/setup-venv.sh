#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ULORA_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== Setting up ULoRA Python venv ==="

VENV_DIR="${ULORA_VENV_DIR:-$ULORA_ROOT/.venv}"
PYTHON_BIN="${ULORA_PYTHON_BIN:-python3}"

echo "Using Python: $PYTHON_BIN"
echo "Venv at: $VENV_DIR"

"$PYTHON_BIN" -m venv "$VENV_DIR"
# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"

pip install --upgrade pip
pip install numpy scipy safetensors pyarrow pandas

# Path C (gradient_sft) training stack. Versions are pinned: the recipe hash and
# the loss-curve digest are only comparable across runs on the same stack.
# Spike 5.2 validated: torch 2.7.1+cu128 on driver 595 (RTX 3060, sm_86), on a
# CPU without AVX. Set ULORA_TORCH_INDEX to another wheel index (e.g. cu126, cpu).
TORCH_INDEX="${ULORA_TORCH_INDEX:-https://download.pytorch.org/whl/cu128}"
pip install "torch==2.7.1" --index-url "$TORCH_INDEX"
pip install "transformers==4.53.3" "peft==0.16.0" "accelerate==1.8.1" "safetensors==0.5.3"

echo "=== ULoRA venv ready ==="
echo "Activate with: source $VENV_DIR/bin/activate"
