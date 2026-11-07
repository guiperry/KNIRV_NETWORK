#!/usr/bin/env bash
#
# ua-dashboard.sh — launch the Understand-Anything knowledge-graph dashboard.
#
# Starts the Vite dev server bundled with the understand-anything plugin, pointed
# at a project's .ua/ (or legacy .understand-anything/) knowledge graph, behind a
# simple token gate. No agent/chat required — run this and open the printed URL.
#
# Usage:
#   scripts/ua-dashboard.sh [PROJECT_DIR] [TOKEN] [PORT]
#
#   PROJECT_DIR  directory that contains .ua/knowledge-graph.json
#                (default: current directory)
#   TOKEN        value for the ?token= access gate (default: knirv)
#   PORT         local port to bind (default: 5173; auto-increments if busy)
#
# Override the plugin location with UNDERSTAND_ANYTHING_PLUGIN=/path if needed.

set -euo pipefail

PROJECT_DIR="${1:-$(pwd)}"
TOKEN="${2:-knirv}"
PORT="${3:-5173}"

# Resolve to an absolute path up front: the script later cd's into the plugin's
# dashboard directory, so a relative PROJECT_DIR would resolve against the wrong cwd.
PROJECT_DIR="$(cd "$PROJECT_DIR" && pwd -P)"

PLUGIN_ROOT="${UNDERSTAND_ANYTHING_PLUGIN:-$HOME/.understand-anything-plugin}"
DASHBOARD_DIR="$PLUGIN_ROOT/packages/dashboard"

if [ ! -d "$DASHBOARD_DIR" ]; then
  echo "error: dashboard not found at $DASHBOARD_DIR" >&2
  echo "       set UNDERSTAND_ANYTHING_PLUGIN or install the understand-anything plugin." >&2
  exit 1
fi

UA_DIR="$PROJECT_DIR/$([ -d "$PROJECT_DIR/.understand-anything" ] && echo .understand-anything || echo .ua)"
if [ ! -f "$UA_DIR/knowledge-graph.json" ]; then
  echo "error: no knowledge-graph.json in $UA_DIR" >&2
  echo "       run /understand on this project first to generate the graph." >&2
  exit 1
fi

# Pick a free port at or after the requested one so we never clobber a running server.
while :; do
  if ! (exec 3<>"/dev/tcp/127.0.0.1/$PORT") 2>/dev/null; then
    break
  fi
  PORT=$((PORT + 1))
done

LOG="/tmp/ua-dashboard.log"
PIDFILE="/tmp/ua-dashboard.pid"
cd "$DASHBOARD_DIR"
GRAPH_DIR="$PROJECT_DIR" \
  UNDERSTAND_ACCESS_TOKEN="$TOKEN" \
  nohup npx vite --host 127.0.0.1 --port "$PORT" > "$LOG" 2>&1 &
PID=$!
echo "$PID" > "$PIDFILE"

echo "Understand-Anything dashboard launching (pid $PID)..."
echo "  project : $PROJECT_DIR"
echo "  graph   : $UA_DIR/knowledge-graph.json"

# Give Vite a moment, then surface the real URL (and fail loudly if it didn't start).
sleep 5
if ! kill -0 "$PID" 2>/dev/null; then
  echo "error: dashboard exited early — see $LOG:" >&2
  tail -n 15 "$LOG" >&2
  exit 1
fi

# Record the actual process holding the port (npx spawns a child node that
# serves it), so `make ua-dashboard-stop` terminates the right process.
LISTENER_PID="$( (ss -ltnp 2>/dev/null || netstat -ltnp 2>/dev/null) \
  | grep "127.0.0.1:$PORT" | grep -oE 'pid=[0-9]+' | head -1 | cut -d= -f2 )"
[ -n "$LISTENER_PID" ] && echo "$LISTENER_PID" > "$PIDFILE"

echo ""
echo "  🔑  Dashboard URL: http://127.0.0.1:$PORT/?token=$TOKEN"
echo "  (log: $LOG  |  stop: kill $PID  |  or: make ua-dashboard-stop)"
