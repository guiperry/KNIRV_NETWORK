#!/usr/bin/env bash
# Starts the built daemon on a temp unix socket and checks the /models/* surface a
# release depends on: all three models registered, HEART idle after startup (the health
# and registry checks never build it; only the /models/heart/decide requests below may),
# the HEAD models route answering 200 or 503, the HEART decision routes mounted under
# /models/heart/decide (POST 200 or 503; GET on the POST-only reason route 405 or 503; never a fabricated answer), HASHER validate/index/verify
# answering. Usage: smoke-models.sh <binary> [--gz]
set -euo pipefail
bin=${1:?usage: smoke-models.sh <binary> [--gz]}
tmp=$(mktemp -d)
pid=
cleanup() { [ -n "$pid" ] && kill "$pid" 2>/dev/null || true; rm -rf "$tmp"; }
trap cleanup EXIT
if [ "${2:-}" = "--gz" ]; then gunzip -c "$bin" > "$tmp/knirvmodels"; chmod +x "$tmp/knirvmodels"; bin="$tmp/knirvmodels"; fi

KNIRV_APP_DATA_DIR="$tmp/data" "$bin" --headless --socket-path "$tmp/m.sock" >"$tmp/log" 2>&1 &
pid=$!
for _ in $(seq 1 600); do [ -S "$tmp/m.sock" ] && break; sleep 0.1; done
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
# Evidence index: version and embedder ids are reported (an empty index is fine).
idx=$(get /models/hasher/index/status)
echo "$idx" | grep -q '"index_version":"' || fail "index status has no version: $idx"
echo "$idx" | grep -q '"embedder_ids":\[' || fail "index status has no embedder ids: $idx"
# Verify with no llama up must degrade, never answer: gate uncommitted, not charged, no SUPPORTED/REFUTED.
ver=$(curl -sS --unix-socket "$tmp/m.sock" -X POST -d '{"text":"Paris is the capital of France.","delivery":"inline","budget_ms":3000}' http://models/models/hasher/verify)
echo "$ver" | grep -q '"degraded":true' || fail "verify did not degrade without llama: $ver"
echo "$ver" | grep -q '"gate":"uncommitted"' || fail "verify gate is not uncommitted without llama: $ver"
echo "$ver" | grep -q '"chargeable":false' || fail "a degraded verify is chargeable: $ver"
echo "$ver" | grep -Eq '"verdict":"(SUPPORTED|REFUTED)"' && fail "verify guessed a verdict without llama: $ver"
# HANDS is mounted only when KNIRV_HANDS_ENABLED=1 (pkg/models/flags.go). Untrained HANDS answers 503 on
# weights/generate/plan/act; status, health and models always answer 200. train is not checked: it starts a job.
# On a trained HANDS box (N2) /plan and /act run the policy-gated tool loop, so they are real requests here;
# a 502 from them counts as a FAIL (N1, intended), since only 200 or 503 is accepted.
post_code() { curl -sS -o /dev/null -w '%{http_code}' --unix-socket "$tmp/m.sock" -X POST -d "$2" "http://models$1"; }
if [ "${KNIRV_HANDS_ENABLED:-}" = 1 ]; then
  for p in /models/hands/status /models/hands/health /models/hands/models; do
    c=$(code "$p"); [ "$c" = 200 ] || fail "$p returned $c (expected 200)"
  done
  c=$(code /models/hands/weights); [ "$c" = 200 ] || [ "$c" = 503 ] || fail "/models/hands/weights returned $c (expected 200 or 503)"
  c=$(code /models/hands/generate); [ "$c" = 405 ] || fail "GET /models/hands/generate returned $c (expected 405)"
  for p in /models/hands/generate /models/hands/plan /models/hands/act; do
    c=$(post_code "$p" '{"prompt":"smoke","n":1}'); [ "$c" = 200 ] || [ "$c" = 503 ] || fail "POST $p returned $c (expected 200 or 503)"
  done
fi
# HEART decision core is served under /models/heart/decide once HEART builds. These requests may build HEART,
# so they run after the idle assertions above. Build failure or no checkpoint is a 503 (never a fabricated answer).
c=$(code /models/heart/decide/models); [ "$c" = 200 ] || [ "$c" = 503 ] || fail "/models/heart/decide/models returned $c (expected 200 or 503)"
c=$(post_code /models/heart/decide/reason '{"tokens":[1]}'); [ "$c" = 200 ] || [ "$c" = 503 ] || fail "POST /models/heart/decide/reason returned $c (expected 200 or 503)"
c=$(code /models/heart/decide/reason); [ "$c" = 405 ] || [ "$c" = 503 ] || fail "GET /models/heart/decide/reason returned $c (expected 405 or 503)"
echo "models smoke: ok ($(echo "$reg" | grep -o '"name":"[a-z]*"' | tr '\n' ' '))"
