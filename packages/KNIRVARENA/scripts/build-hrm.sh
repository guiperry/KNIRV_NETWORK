#!/usr/bin/env bash
# Builds the knirv-hrm crate (../KNIRVINFERENCER/hrm, its source of truth) for the browser into src/wasm-pkg/knirv_cortex_wasm.
# Generated output is not committed. getrandom's JS backend is required by candle's rand dependency.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command -v wasm-pack >/dev/null || { echo "wasm-pack not found: cargo install wasm-pack" >&2; exit 1; }
rustup target list --installed 2>/dev/null | grep -q wasm32-unknown-unknown || rustup target add wasm32-unknown-unknown
export RUSTFLAGS="${RUSTFLAGS:-} --cfg getrandom_backend=\"wasm_js\""
HRM_CRATE="${HRM_CRATE:-$ROOT/../KNIRVINFERENCER/hrm}"
[ -f "$HRM_CRATE/Cargo.toml" ] || { echo "HRM crate not found at $HRM_CRATE" >&2; exit 1; }
wasm-pack build "$HRM_CRATE" --target web --release \
  --out-dir "$ROOT/src/wasm-pkg/knirv_cortex_wasm" --out-name knirv_cortex_wasm
