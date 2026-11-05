#!/usr/bin/env bash
# Cross-runtime golden check: the browser (wasm-bindgen) build must reproduce the native
# golden output (tests/golden.json). The wazero half runs in pkg/head (go test).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT
export RUSTFLAGS="${RUSTFLAGS:-} --cfg getrandom_backend=\"wasm_js\""
wasm-pack build . --target nodejs --release --out-dir "$OUT/pkg" --out-name knirv_head >/dev/null
node tests/golden_wasm.cjs "$OUT/pkg/knirv_head.js"
