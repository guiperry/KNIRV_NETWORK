#!/usr/bin/env bash
# Stop only the KNIRVCLIENT application process. Other KNIRV services are not
# matched or signalled by this script.

set -u -o pipefail

is_knirvclient_process() {
    local pid="$1"
    local command_name

    command_name="$(ps -p "$pid" -o comm= 2>/dev/null | tr -d '[:space:]')"
    command_name="${command_name##*/}"
    case "$command_name" in
        knirvclient|KNIRVCLIENT) return 0 ;;
        *) return 1 ;;
    esac
}

pids=()
for process_name in knirvclient KNIRVCLIENT; do
    while IFS= read -r pid; do
        [[ -n "$pid" ]] && pids+=("$pid")
    done < <(pgrep -x "$process_name" 2>/dev/null || true)
done

if ((${#pids[@]} == 0)); then
    echo "KNIRVCLIENT is not running. No processes were changed."
    exit 0
fi

echo "Stopping KNIRVCLIENT process(es): ${pids[*]}"
for pid in "${pids[@]}"; do
    if is_knirvclient_process "$pid"; then
        kill -TERM "$pid"
    else
        echo "Skipping PID $pid: it is no longer KNIRVCLIENT."
    fi
done

# Give KNIRVCLIENT a chance to release its listener and persist state before
# escalating. Recheck the executable name to avoid signalling a reused PID.
sleep 5

remaining=()
for pid in "${pids[@]}"; do
    if is_knirvclient_process "$pid"; then
        remaining+=("$pid")
    fi
done

if ((${#remaining[@]} > 0)); then
    echo "KNIRVCLIENT did not exit after SIGTERM; forcing stop: ${remaining[*]}"
    for pid in "${remaining[@]}"; do
        if is_knirvclient_process "$pid"; then
            kill -KILL "$pid"
        fi
    done
fi

echo "KNIRVCLIENT stopped. Other KNIRV services were left untouched."
