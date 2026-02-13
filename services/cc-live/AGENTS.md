# cc-live — Live Claude Code Status Service

Go WebSocket service that broadcasts whether a Claude Code session is currently active. Runs as a sidecar container alongside the blog's nginx container on the hosting VM.

## Architecture

```
Laptop (Tailscale)                    Hosting VM (Tailscale + DMZ)
┌──────────────────┐                  ┌──────────────────────────┐
│ SessionStart hook│                  │  ┌────────┐ ┌─────────┐ │
│  → start.sh      │                  │  │ nginx  │ │ cc-live │ │
│  → heartbeat ────┼── Tailscale ───→ │  │ (blog) │ │  (Go)   │ │
│                  │   POST :8080     │  └────────┘ └────┬────┘ │
│ SessionEnd hook  │                  │       Traefik ───┘      │
│  → stop.sh       │                  └──────────────────────────┘
└──────────────────┘                          ↕ wss://
                                          Browser
```

## How It Works

1. **Heartbeat API** (`POST /api/live/heartbeat`): The local daemon sends this every 30s over Tailscale. Authenticated via `Authorization: Bearer <key>`.
2. **Stop API** (`POST /api/live/stop`): Sent by `SessionEnd` hook for immediate deactivation.
3. **Expiry**: Background goroutine checks every 10s. If no heartbeat in 60s, marks inactive and broadcasts.
4. **WebSocket** (`GET /ws/live`): Browser clients connect here. Receives `{"active": true/false}` on connect and on every state change.

## Files

- `main.go` — single-file server, ~150 lines. Only external dep: `golang.org/x/net/websocket`.
- `Containerfile` — multi-stage build (golang:1.24-alpine → alpine).
- `go.mod` / `go.sum` — Go module.

## Environment Variables

- `CC_LIVE_API_KEY` (required) — shared secret for heartbeat/stop authentication.

## Local Development

```bash
# Run standalone
CC_LIVE_API_KEY=dev-secret go run .

# Or via docker compose from the repo root (includes Traefik + blog)
make dev

# Test heartbeat
make dev-heartbeat

# Test WebSocket (requires websocat or browser devtools)
websocat ws://localhost:8000/ws/live
```

## Build & Deploy

```bash
# From repo root
make build-cc-live    # build image
make push-cc-live     # build + push to registry
make deploy-cc-live   # build + push + pulumi up
make deploy-all       # deploy blog + cc-live together
```

## Local Daemon Scripts

The heartbeat daemon lives in `scripts/cc-live/`:

- `start.sh` — started by `SessionStart` hook. Launches a background curl loop (every 30s). Writes PID to `/tmp/cc-live-daemon.pid`.
- `stop.sh` — started by `SessionEnd` hook. Kills the daemon and sends a final stop signal.

Requires env vars in shell profile:
- `CC_LIVE_ENDPOINT` — production: `http://<hosting-vm-tailscale-ip>:8080`
- `CC_LIVE_API_KEY` — same key as the server

Scripts silently exit 0 if env vars are not set, so they never block Claude Code.

## Infrastructure

- **Pulumi**: Defined in `~/projects/homeserver/src/services/index.ts` as a raw `docker.Container` (not `WebService`) for path-based Traefik routing + host port mapping.
- **Traefik routing**: `Host(howinator.io) && PathPrefix(/ws/live)` and `PathPrefix(/api/live)` → cc-live:8080. Higher specificity than the blog's `Host(howinator.io)` catch-all.
- **Port 8080**: Mapped to host for direct Tailscale access from the daemon.
