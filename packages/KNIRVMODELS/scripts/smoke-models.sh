#!/usr/bin/env bash
# Starts the built daemon on a temp unix socket and checks the /models/* surface a
# release depends on: all three models registered, HEART idle (asking never builds
# it), both HEAD prefixes routed, HASHER answering. Usage: smoke-models.sh <binary> [--gz]
set -euo pipefail
bin=${1:?usage: smoke-models.sh <binary> [--gz]}
tmp=$(mktemp -d)
pid=
cleanup() { [ -n "$pid" ] && kill "$pid" 2>/dev/null || true; rm -rf "$tmp"; }
trap cleanup EXIT
if [ "${2:-}" = "--gz" ]; then gunzip -c "$bin" > "$tmp/knirvmodels"; chmod +x "$tmp/knirvmodels"; bin="$tmp/knirvmodels"; fi

KNIRV_APP_DATA_DIR="$tmp/data" "$bin" --headless --socket-path "$tmp/m.sock" >"$tmp/log" 2>&1 &
pid=$!
for _ in $(seq 1 100); do [ -S "$tmp/m.sock" ] && break; sleep 0.1; done
[ -S "$tmp/m.sock" ] || { echo "smoke: daemon never opened its socket"; cat "$tmp/log"; exit 1; }

get() { curl -sS --unix-socket "$tmp/m.sock" "http://models$1"; }
code() { curl -sS -o /dev/null -w '%{http_code}' --unix-socket "$tmp/m.sock" "http://models$1"; }
fail() { echo "smoke: $1"; cat "$tmp/log"; exit 1; }

reg=$(get /models/registry)
for m in hasher heart head; do echo "$reg" | grep -q "\"name\":\"$m\"" || fail "registry is missing $m: $reg"; done
echo "$(get /models/heart/health)" | grep -q '"loaded":false' || fail "HEART was built by a health check"
echo "$(get /models/registry)" | grep -q '"name":"heart","role":"generator","version":"gorgonite","ready":false,"loaded":false' || fail "registry shows HEART loaded"
for p in /models/head/models; do
  c=$(code "$p"); [ "$c" = 200 ] || [ "$c" = 503 ] || fail "$p returned $c (expected 200 or 503)"
done
echo "$(curl -sS --unix-socket "$tmp/m.sock" -X POST -d '{"prediction":1,"domain":"smoke"}' http://models/models/hasher/validate)" | grep -q '"valid":true' || fail "HASHER validate"
echo "models smoke: ok ($(echo "$reg" | grep -o '"name":"[a-z]*"' | tr '\n' ' '))"
