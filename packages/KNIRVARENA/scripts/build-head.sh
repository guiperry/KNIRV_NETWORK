#!/usr/bin/env bash
# Builds the knirv-head crate (../KNIRVMODELS/head, its source of truth) for the browser into src/wasm-pkg/knirv_head.
# Generated output is not committed. getrandom's JS backend is required by candle's rand dependency.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command -v wasm-pack >/dev/null || { echo "wasm-pack not found: cargo install wasm-pack" >&2; exit 1; }
rustup target list --installed 2>/dev/null | grep -q wasm32-unknown-unknown || rustup target add wasm32-unknown-unknown
export RUSTFLAGS="${RUSTFLAGS:-} --cfg getrandom_backend=\"wasm_js\""
HEAD_CRATE="${HEAD_CRATE:-$ROOT/../KNIRVMODELS/head}"
[ -f "$HEAD_CRATE/Cargo.toml" ] || { echo "HEAD crate not found at $HEAD_CRATE" >&2; exit 1; }
wasm-pack build "$HEAD_CRATE" --target web --release \
  --out-dir "$ROOT/src/wasm-pkg/knirv_head" --out-name knirv_head
