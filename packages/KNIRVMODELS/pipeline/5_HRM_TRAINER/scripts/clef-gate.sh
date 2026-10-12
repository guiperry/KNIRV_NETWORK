#!/usr/bin/env bash
# M7-0: run the Clef teacher gate (BANKING77 + BFCL v3) with the Cloudflare
# credentials KNIRVSERVER already holds (root.key -> backend_server's environment).
# The token is read from the running server's process environment and handed to
# the harness; it is never printed, written to disk, or put on a command line.
#
#   sudo ./clef-gate.sh [--preflight | --irrelevance [--model clef|clef-flash] | --limit-b77 N --limit-bfcl M]
#   --irrelevance: only the full BFCL irrelevance split (abstention check, ~$0.01)
#
# Needs root only to read /proc/<pid>/environ; the harness itself runs as the
# invoking user. The harness is vendored at ../clef-harness; override with CLEF_HARNESS.
set -euo pipefail

HARNESS="${CLEF_HARNESS:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../clef-harness" && pwd)}"
RUN_AS="${SUDO_USER:-$(id -un)}"
B77=400 BFCL=200 PRE=0 MODEL=clef-flash
while [ $# -gt 0 ]; do
  case "$1" in
    --preflight) PRE=1 ;;
    --limit-b77) B77="$2"; shift ;;
    --irrelevance) IRR=1 ;;
    --model) MODEL="$2"; shift ;;
    --limit-bfcl) BFCL="$2"; shift ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac; shift
done
[ "$(id -u)" -eq 0 ] || { echo "run with sudo (reads the server's environment)" >&2; exit 1; }
[ -d "$HARNESS" ] || { echo "clef-harness not found at $HARNESS (set CLEF_HARNESS)" >&2; exit 1; }

PID="$(pgrep -fo 'bin/backend_server' || true)"
[ -n "$PID" ] || { echo "backend_server is not running" >&2; exit 1; }

get() { tr '\0' '\n' < "/proc/$PID/environ" | grep -m1 "^$1=" | cut -d= -f2- || true; }
CLOUDFLARE_API_TOKEN="$(get CLOUDFLARE_API_TOKEN)"
CLOUDFLARE_ACCOUNT_ID="$(get CLOUDFLARE_ACCOUNT_ID)"
[ -n "$CLOUDFLARE_API_TOKEN" ] && [ -n "$CLOUDFLARE_ACCOUNT_ID" ] || {
  echo "backend_server (pid $PID) has no CLOUDFLARE_API_TOKEN/ACCOUNT_ID: root.key not unlocked?" >&2; exit 1; }
export CLOUDFLARE_API_TOKEN CLOUDFLARE_ACCOUNT_ID
echo "credentials: present (from backend_server pid $PID)"

run() { runuser -u "$RUN_AS" --preserve-environment -- "$@"; }
cd "$HARNESS"
run python3 run_banking77.py --preflight
[ "$PRE" -eq 1 ] && exit 0
if [ "${IRR:-0}" -eq 1 ]; then
  OUT=results/bfcl_irrelevance.json; [ "$MODEL" = clef-flash ] || OUT="results/bfcl_irrelevance_$MODEL.json"
  run python3 run_bfcl.py --model "$MODEL" --splits BFCL_v3_irrelevance --limit 0 --out "$OUT"
  echo "results: $HARNESS/$OUT"
  exit 0
fi
run python3 run_banking77.py --limit "$B77"
run python3 run_bfcl.py --limit "$BFCL"
echo "results: $HARNESS/results"
