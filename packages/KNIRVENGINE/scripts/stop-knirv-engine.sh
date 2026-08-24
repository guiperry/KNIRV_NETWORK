#!/usr/bin/env bash
# Gracefully stop KNIRV_ENGINE only. It deliberately does not match or signal
# any KNIRVSERVER process.
set -euo pipefail

config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/KNIRV-Engine"
pid_files=("$config_dir/knirv-engine-gui.pid" "$config_dir/knirv-engine.pid")
declare -A seen_pids=()

is_running() {
  kill -0 "$1" 2>/dev/null
}

signal_tree() {
  local pid="$1"
  local child

  [[ -n "${seen_pids[$pid]:-}" ]] && return
  seen_pids[$pid]=1
  is_running "$pid" || return

  while IFS= read -r child; do
    [[ -n "$child" ]] && signal_tree "$child"
  done < <(pgrep -P "$pid" 2>/dev/null || true)

  echo "Sending SIGINT to KNIRV_ENGINE process $pid"
  kill -INT "$pid" 2>/dev/null || true
}

for pid_file in "${pid_files[@]}"; do
  [[ -f "$pid_file" ]] || continue
  pid="$(tr -d '[:space:]' < "$pid_file")"
  if [[ "$pid" =~ ^[0-9]+$ ]] && is_running "$pid"; then
    signal_tree "$pid"
  else
    rm -f "$pid_file"
  fi
done

# Fallback for an engine launched before PID tracking was introduced. Exact
# process names ensure KNIRVSERVER and unrelated KNIRV services remain intact.
for process_name in knirv-engine knirv-engine-linux knirv-engine-macos; do
  while IFS= read -r pid; do
    [[ -n "$pid" ]] && signal_tree "$pid"
  done < <(pgrep -x "$process_name" 2>/dev/null || true)
done

if (( ${#seen_pids[@]} == 0 )); then
  echo "No KNIRV_ENGINE processes found. KNIRVSERVER services were not touched."
  exit 0
fi

for _ in {1..50}; do
  remaining=0
  for pid in "${!seen_pids[@]}"; do
    is_running "$pid" && remaining=1
  done
  (( remaining == 0 )) && break
  sleep 0.1
done

for pid in "${!seen_pids[@]}"; do
  if is_running "$pid"; then
    echo "Process $pid is still running after SIGINT; sending SIGTERM"
    kill -TERM "$pid" 2>/dev/null || true
  fi
done

echo "KNIRV_ENGINE shutdown signal complete. KNIRVSERVER services were not touched."
