#!/usr/bin/env bash
# End-to-end proof that llama.cpp runs a ULoRA-bound adapter, against the REAL
# llama-server (no mocks): tiny GQA llama (tinyllama.py) -> ULoRA binds a bundle to
# it and writes a GGUF LoRA -> the KNIRVLLAMA wrapper registers it (restarting
# llama-server) -> requests select it by name.
#
#   ./e2e.sh            LLAMA_CPP_DIR defaults to /var/lib/knirvserver/knirvllama/llama.cpp
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
ulora=$(cd "$here/../.." && pwd)
wrapper=$(cd "$ulora/../KNIRVSERVER/pkg/embedded/llama" && pwd)
llama_cpp=${LLAMA_CPP_DIR:-/var/lib/knirvserver/knirvllama/llama.cpp}
server=$llama_cpp/build/bin/llama-server
for need in "$server" "$llama_cpp/gguf-py"; do [ -e "$need" ] || { echo "e2e: missing $need"; exit 2; }; done
tmp=$(mktemp -d); pid=
trap '[ -n "$pid" ] && kill "$pid" 2>/dev/null; wait 2>/dev/null; rm -rf "$tmp"' EXIT
fail() { echo "E2E FAIL: $*"; [ -f "$tmp/wrapper.log" ] && tail -20 "$tmp/wrapper.log"; exit 1; }

echo "== 1. tiny GQA llama (4 heads, 2 kv heads, 2 layers)"
PYTHONPATH=$llama_cpp/gguf-py python3 "$here/tinyllama.py" "$llama_cpp" "$tmp/model" >/dev/null
mkdir -p "$tmp/out"

echo "== 2. ULoRA: bind a bundle to it, derive the connector from its weights, write the GGUF LoRA"
(cd "$ulora" && ULORA_E2E_BASE="$tmp/model/tiny.gguf" ULORA_E2E_OUT="$tmp/out" go test -count=1 -run TestWriteAdaptersForLlamaE2E ./internal/daemon/ >"$tmp/ulora.log" 2>&1) || { cat "$tmp/ulora.log"; fail "ULoRA could not write the adapter"; }
adapter=$(python3 -c "import json;print(json.load(open('$tmp/out/e2e.json'))['adapter'])")
wrong=$(python3 -c "import json;print(json.load(open('$tmp/out/e2e.json'))['wrong_base'])")

echo "== 3. start the KNIRVLLAMA wrapper over the real llama-server"
(cd "$wrapper" && go build -o "$tmp/llama-wrapper" ./cmd/llama)
port=$((20000 + RANDOM % 20000)); lport=$((port + 1))
"$tmp/llama-wrapper" -data-dir "$tmp/data" -listen 127.0.0.1:$port -llama-address 127.0.0.1:$lport \
  -server-path "$server" -model-path "$tmp/model/tiny.gguf" -ctx-size 256 -no-install >"$tmp/wrapper.log" 2>&1 &
pid=$!
for _ in $(seq 1 120); do curl -sf "localhost:$port/health" >/dev/null 2>&1 && break; sleep 0.5; done
curl -sf "localhost:$port/health" >/dev/null || fail "wrapper never became healthy"

api=localhost:$port
gen() { # gen <adapter-or-empty>  -> prints "<header>|<text>"
  local extra=""; [ -n "$1" ] && extra=",\"adapter\":\"$1\""
  curl -s -D "$tmp/h" "$api/v1/completions" -d "{\"prompt\":\"hello\",\"max_tokens\":12,\"temperature\":0,\"seed\":1,\"cache_prompt\":false$extra}" -o "$tmp/b"
  echo "$(grep -i '^x-knirv-adapter' "$tmp/h" | tr -d '\r' | cut -d' ' -f2)|$(python3 -c "import json;d=json.load(open('$tmp/b'));print(d['choices'][0]['text'])" 2>/dev/null || echo ERR)"
}

echo "== 4. baseline (no adapters loaded)"
baseline=$(gen ""); echo "   $baseline"
[ "${baseline#base|}" != "$baseline" ] || fail "no X-KNIRV-Adapter: base header"
[ "${baseline#*|}" != ERR ] || fail "baseline request failed"
base_text=${baseline#*|}

echo "== 5. an adapter bound to ANOTHER base model must be refused (409)"
code=$(curl -s -o "$tmp/r" -w '%{http_code}' "$api/v1/adapters" -d "{\"name\":\"wrong\",\"path\":\"$wrong\"}")
[ "$code" = 409 ] || fail "wrong-base adapter returned $code: $(cat "$tmp/r")"

echo "== 6. register the real adapter (llama-server restarts with it)"
code=$(curl -s -o "$tmp/r" -w '%{http_code}' "$api/v1/adapters" -d "{\"name\":\"skill-e2e\",\"path\":\"$adapter\"}")
[ "$code" = 201 ] || fail "register returned $code: $(cat "$tmp/r")"
curl -s "$api/v1/adapters" | python3 -c "import sys,json;d=json.load(sys.stdin);a=d['adapters'];assert len(a)==1 and a[0]['loaded'] and a[0]['id']==0,a;print('   loaded:',a[0]['name'],'id',a[0]['id'])"

echo "== 7. selection"
with=$(gen skill-e2e); echo "   adapter: $with"
none=$(gen "");        echo "   no adapter: $none"
[ "${with%%|*}" = skill-e2e ] || fail "header did not name the adapter: $with"
[ "${none%%|*}" = base ] || fail "header did not say base: $none"
[ "${with#*|}" != "$base_text" ] || fail "the adapter did not change the output: the LoRA is not applied"
[ "${none#*|}" = "$base_text" ] || fail "a request without an adapter differs from the baseline: the adapter leaked into base requests"
code=$(curl -s -o /dev/null -w '%{http_code}' "$api/v1/completions" -d '{"prompt":"x","adapter":"ghost"}')
[ "$code" = 404 ] || fail "unknown adapter returned $code, want 404"
code=$(curl -s -o /dev/null -w '%{http_code}' "$api/v1/completions" -d '{"prompt":"x","lora":[{"id":0,"scale":1}]}')
[ "$code" = 400 ] || fail "raw lora returned $code, want 400"

echo "== 8. removal restores the plain base model"
code=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE "$api/v1/adapters/skill-e2e"); [ "$code" = 204 ] || fail "delete returned $code"
after=$(gen ""); [ "${after#*|}" = "$base_text" ] || fail "output after removal differs from baseline: $after"
code=$(curl -s -o /dev/null -w '%{http_code}' "$api/v1/completions" -d '{"prompt":"x","adapter":"skill-e2e"}'); [ "$code" = 404 ] || fail "removed adapter still selectable ($code)"

echo "== 9. the registry survives a wrapper restart"
curl -s -o /dev/null "$api/v1/adapters" -d "{\"name\":\"skill-e2e\",\"path\":\"$adapter\"}"
kill "$pid"; wait "$pid" 2>/dev/null || true
"$tmp/llama-wrapper" -data-dir "$tmp/data" -listen 127.0.0.1:$port -llama-address 127.0.0.1:$lport \
  -server-path "$server" -model-path "$tmp/model/tiny.gguf" -ctx-size 256 -no-install >>"$tmp/wrapper.log" 2>&1 &
pid=$!
for _ in $(seq 1 120); do curl -sf "localhost:$port/health" >/dev/null 2>&1 && break; sleep 0.5; done
again=$(gen skill-e2e); [ "$again" = "$with" ] || fail "after restart: '$again' != '$with'"
echo "E2E OK: llama.cpp ran the ULoRA-bound adapter (adapter != base, base unchanged, registry persisted)"
