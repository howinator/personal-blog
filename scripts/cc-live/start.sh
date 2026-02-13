#!/bin/bash
# Start the cc-live heartbeat daemon in the background.
# Called by Claude Code SessionStart hook.

set -euo pipefail

PIDFILE="/tmp/cc-live-daemon.pid"

# Kill any existing daemon
if [ -f "$PIDFILE" ]; then
    kill "$(cat "$PIDFILE")" 2>/dev/null || true
    rm -f "$PIDFILE"
fi

if [ -z "${CC_LIVE_ENDPOINT:-}" ] || [ -z "${CC_LIVE_API_KEY:-}" ]; then
    exit 0  # Silently skip if not configured
fi

# Start heartbeat loop in background
(while true; do
    curl -sf -m 5 -X POST "$CC_LIVE_ENDPOINT/api/live/heartbeat" \
        -H "Authorization: Bearer $CC_LIVE_API_KEY" >/dev/null 2>&1 || true
    sleep 30
done) &

echo $! > "$PIDFILE"
