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

echo "=== ULoRA venv ready ==="
echo "Activate with: source $VENV_DIR/bin/activate"
