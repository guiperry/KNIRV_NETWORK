#!/usr/bin/env bash
# Sync the KNIRVBAO plugin's embedded knirv_core.wasm and pin constants with the
# current KNIRVSDK core stamp, so `make release` cannot ship a stale pin.
#
# The plugin embeds pkg/embedded/knirvbao/plugin/assets/knirv_core.wasm and pins
# it via embeddedCoreVersion / embeddedCoreDigest in core.go. These must match
# packages/KNIRVSDK/go-package/internal/core/stamp.go. This script copies the
# matching wasm from the SDK dist dir, verifies its SHA-256 against the stamp
# digest, and rewrites the two constants. Run as a pre-step of `knirvbao-build`.
set -euo pipefail

server_dir=$(cd "$(dirname "$0")/.." && pwd)
plugin_dir="$server_dir/pkg/embedded/knirvbao/plugin"
sdk_dir=${KNIRVSDK_DIR:-"$server_dir/../KNIRVSDK"}
stamp="$sdk_dir/go-package/internal/core/stamp.go"
core_go="$plugin_dir/core.go"
assets_dir="$plugin_dir/assets"

[ -f "$stamp" ] || { echo "sync-knirvbao-core: cannot find SDK stamp at $stamp" >&2; exit 1; }
[ -f "$core_go" ] || { echo "sync-knirvbao-core: cannot find $core_go" >&2; exit 1; }

version=$(grep -oP 'Version\s*=\s*"\K[^"]+' "$stamp")
digest=$(grep -oP 'Digest\s*=\s*"\K[^"]+' "$stamp")
[ -n "$version" ] && [ -n "$digest" ] || { echo "sync-knirvbao-core: failed to parse Version/Digest from stamp" >&2; exit 1; }

# Prefer the released dist artifact for this version; fall back to the WASI build.
wasm_src="$sdk_dir/dist/$version/knirv_core.wasm"
if [ ! -f "$wasm_src" ]; then
  wasm_src="$sdk_dir/bindings/wasi/target/wasm32-wasip1/release/knirv_core.wasm"
fi
[ -f "$wasm_src" ] || { echo "sync-knirvbao-core: no knirv_core.wasm for version $version (looked in dist/$version and bindings/wasi build)" >&2; exit 1; }

actual=$(sha256sum "$wasm_src" | awk '{print $1}')
if [ "$actual" != "$digest" ]; then
  echo "sync-knirvbao-core: wasm digest $actual does not match SDK stamp $digest" >&2
  exit 1
fi

mkdir -p "$assets_dir"
cp "$wasm_src" "$assets_dir/knirv_core.wasm"

# Rewrite the two pin constants without touching anything else in core.go.
tmp=$(mktemp)
awk -v v="$version" -v d="$digest" '
  /^const embeddedCoreVersion = / { print "const embeddedCoreVersion = \"" v "\""; next }
  /^const embeddedCoreDigest = /  { print "const embeddedCoreDigest = \"" d "\"";  next }
  { print }
' "$core_go" > "$tmp"
mv "$tmp" "$core_go"

echo "sync-knirvbao-core: pinned embedded core to v$version ($digest)"
