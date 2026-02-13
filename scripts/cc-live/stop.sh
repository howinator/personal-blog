#!/bin/bash
# Stop the cc-live heartbeat daemon and signal session end.
# Called by Claude Code SessionEnd hook.

set -euo pipefail

PIDFILE="/tmp/cc-live-daemon.pid"

if [ -f "$PIDFILE" ]; then
    kill "$(cat "$PIDFILE")" 2>/dev/null || true
    rm -f "$PIDFILE"
fi

if [ -z "${CC_LIVE_ENDPOINT:-}" ] || [ -z "${CC_LIVE_API_KEY:-}" ]; then
    exit 0
fi

curl -sf -m 5 -X POST "$CC_LIVE_ENDPOINT/api/live/stop" \
    -H "Authorization: Bearer $CC_LIVE_API_KEY" >/dev/null 2>&1 || true
