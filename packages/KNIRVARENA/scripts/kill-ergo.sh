#!/usr/bin/env bash
# kill-ergo.sh — stop all ERGO development and server processes

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

killed=0

kill_by_pattern() {
  local label="$1"
  local pattern="$2"
  local pids
  pids=$(pgrep -f "$pattern" 2>/dev/null || true)
  if [[ -n "$pids" ]]; then
    echo -e "${YELLOW}Stopping ${label}...${NC}"
    echo "$pids" | xargs kill -TERM 2>/dev/null || true
    # Give processes a moment to exit cleanly, then force-kill stragglers
    sleep 1
    local remaining
    remaining=$(pgrep -f "$pattern" 2>/dev/null || true)
    if [[ -n "$remaining" ]]; then
      echo "$remaining" | xargs kill -KILL 2>/dev/null || true
    fi
    killed=$((killed + 1))
    echo -e "${GREEN}  ✓ ${label} stopped${NC}"
  fi
}

echo ""
echo -e "${RED}━━━ ERGO kill script ━━━${NC}"
echo ""

# Vite dev server (npm run dev / npx vite)
kill_by_pattern "Vite dev server"     "vite.*--port 3000"
kill_by_pattern "Vite (generic)"      "node.*vite"

# Node API / simple server
kill_by_pattern "Simple server"       "simple-server.js"
kill_by_pattern "API server"          "api-server.js"
kill_by_pattern "Static server"       "serve-static.js"

# Concurrently wrapper (dev:full)
kill_by_pattern "Concurrently"        "concurrently.*npm run"

# Lorax / networking server if running locally
kill_by_pattern "Lorax client stub"   "lorax"

# Any leftover node processes launched from this repo directory
kill_by_pattern "ERGO node scripts"   "node.*ERGO/scripts/"

# Port 3000 squatter (catches anything holding the Vite port)
if command -v fuser &>/dev/null; then
  port_pid=$(fuser 3000/tcp 2>/dev/null || true)
  if [[ -n "$port_pid" ]]; then
    echo -e "${YELLOW}Releasing port 3000 (pid ${port_pid})...${NC}"
    fuser -k 3000/tcp 2>/dev/null || true
    echo -e "${GREEN}  ✓ Port 3000 released${NC}"
    killed=$((killed + 1))
  fi
fi

echo ""
if [[ $killed -eq 0 ]]; then
  echo -e "${GREEN}Nothing to kill — no ERGO processes found.${NC}"
else
  echo -e "${GREEN}Done. ${killed} process group(s) stopped.${NC}"
fi
echo ""
