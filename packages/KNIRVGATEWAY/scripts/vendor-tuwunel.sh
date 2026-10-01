#!/usr/bin/env bash
# Builds the matrix-construct/tuwunel (Apache-2.0) Matrix homeserver from
# source and vendors the resulting release binary into
# internal/bridge/bin/tuwunel, where internal/bridge/homeserver.go embeds it
# via `//go:embed bin/tuwunel` — the same vendoring convention
# packages/KNIRVSERVER/bin/backend_server uses for its own embedded
# subprocess (see internal/bridge/bin/README.md).
#
# This is a separate, explicit step from `make build-knirvgateway` — a Rust
# release build of a full Matrix homeserver takes significantly longer than
# the Go build it feeds into, and the vendored binary only needs rebuilding
# when the pinned TUWUNEL_REF below changes, not on every `go build`.
#
# Usage:
#   packages/KNIRVGATEWAY/scripts/vendor-tuwunel.sh
#   TUWUNEL_REF=v1.2.3 packages/KNIRVGATEWAY/scripts/vendor-tuwunel.sh
#
# Requires on PATH: git, cargo/rustc (via rustup, so TUWUNEL_RUST_VERSION
# below can be installed on demand), a C/C++ toolchain (gcc or clang),
# cmake, make, autoconf, automake, libtool (jemalloc-sys and rust-rocksdb's
# vendored builds need these — no system RocksDB/jemalloc/liburing packages
# required). io_uring and systemd integration are intentionally disabled
# (see TUWUNEL_FEATURES below): KNIRVGATEWAY manages this binary directly via
# os/exec, not systemd, and avoiding io_uring means no liburing-dev
# dependency either.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEST="$GATEWAY_DIR/internal/bridge/bin/tuwunel"

TUWUNEL_REPO="${TUWUNEL_REPO:-https://github.com/matrix-construct/tuwunel.git}"
TUWUNEL_REF="${TUWUNEL_REF:-main}"
# Tuwunel's own Cargo.toml declares rust-version — bump this alongside it.
TUWUNEL_RUST_VERSION="${TUWUNEL_RUST_VERSION:-1.96.1}"
TUWUNEL_FEATURES="${TUWUNEL_FEATURES:-brotli_compression,element_hacks,gzip_compression,jemalloc,jemalloc_conf,media_thumbnail,release_max_log_level,url_preview,zstd_compression}"

WORKDIR="$(mktemp -d /tmp/knirv-tuwunel-build.XXXXXX)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

echo "==> Cloning tuwunel ($TUWUNEL_REF) into $WORKDIR"
git clone --depth 1 --branch "$TUWUNEL_REF" "$TUWUNEL_REPO" "$WORKDIR/tuwunel" 2>/dev/null \
  || git clone "$TUWUNEL_REPO" "$WORKDIR/tuwunel"
cd "$WORKDIR/tuwunel"
# --branch only works for actual branch/tag refs; fall back to a plain
# checkout for an arbitrary commit SHA.
git checkout --quiet "$TUWUNEL_REF" || true

if command -v rustup >/dev/null 2>&1; then
  rustup toolchain install "$TUWUNEL_RUST_VERSION" >/dev/null 2>&1 || true
  rustup override set "$TUWUNEL_RUST_VERSION"
else
  echo "WARNING: rustup not found — using whatever 'cargo' resolves to on PATH." >&2
  echo "         Tuwunel requires rust-version >= $TUWUNEL_RUST_VERSION." >&2
fi

echo "==> Building tuwunel --release (features: $TUWUNEL_FEATURES)"
echo "    This compiles RocksDB from source via rust-rocksdb and will take a while."
cargo build --release --locked --no-default-features --features "$TUWUNEL_FEATURES"

BUILT_BIN="target/release/tuwunel"
if [ ! -x "$BUILT_BIN" ]; then
  echo "ERROR: expected binary not found at $BUILT_BIN" >&2
  exit 1
fi

mkdir -p "$(dirname "$DEST")"
cp "$BUILT_BIN" "$DEST"
chmod +x "$DEST"

echo "==> Vendored tuwunel binary -> $DEST ($(du -h "$DEST" | cut -f1))"
echo "    Remember: 'bin/' is gitignored — force-add this binary the same way"
echo "    packages/KNIRVSERVER/bin/backend_server is:"
echo "      git add -f $DEST"
