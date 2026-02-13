# cc-live — Live Claude Code Status Service

Go WebSocket service that broadcasts whether a Claude Code session is currently active. Runs as a sidecar container alongside the blog's nginx container on the hosting VM.

## Architecture

```
Laptop (Tailscale)                    Hosting VM (Tailscale + DMZ)
┌──────────────────┐                  ┌──────────────────────────┐
│ SessionStart hook│                  │  ┌────────┐ ┌─────────┐ │
│  → register      │                  │  │ nginx  │ │ cc-live │ │
│                  │                  │  │ (blog) │ │  (Go)   │ │
│ Go daemon ───────┼── Tailscale ───→ │  └────────┘ └────┬────┘ │
│  (heartbeat/stop)│   POST :8080     │       Traefik ───┘      │
│                  │                  └──────────────────────────┘
│ SessionEnd hook  │                          ↕ wss://
│  → unregister    │                      Browser
└──────────────────┘
```

### Local Daemon (scripts/cc-live/)

A Go binary (`~/.cc-live/cc-live-daemon`) manages multiple concurrent Claude Code sessions using SQLite for state tracking.

**Subcommands:**
- `register` — Called by `SessionStart` hook. Reads hook payload from stdin, inserts session into SQLite, starts background daemon if not running.
- `unregister` — Called by `SessionEnd` hook. Removes session from SQLite. If no sessions remain, sends stop and kills daemon.
- `serve` — Long-running background process. Every 30s parses transcript JSONL files incrementally, computing per-session metrics (tokens, tool calls, user prompts, active time, last prompt). Sends rich heartbeat payload. Auto-exits after 30 minutes with zero registered sessions.

**State:** `~/.cc-live/state.db` (SQLite via `modernc.org/sqlite`, pure Go, no CGo)

## How It Works

1. **Heartbeat API** (`POST /api/live/heartbeat`): The local daemon sends this every 30s with JSON body `{"sessions": [{session_id, total_tokens, tool_calls, user_prompts, active_time_seconds, last_prompt, project, model, sensitive}, ...]}`. Authenticated via `Authorization: Bearer <key>`.
2. **Stop API** (`POST /api/live/stop`): Sent when all sessions are unregistered for immediate deactivation.
3. **Expiry**: Background goroutine checks every 10s. If no heartbeat in 60s, marks inactive and broadcasts.
4. **WebSocket** (`GET /ws/live`): Browser clients connect here. Receives `{"active": true/false, "sessions": [...]}` on connect and on every state change.

## Files

### Server (services/cc-live/)
- `main.go` — single-file server, ~170 lines. Only external dep: `golang.org/x/net/websocket`.
- `Containerfile` — multi-stage build (golang:1.24-alpine → alpine).
- `go.mod` / `go.sum` — Go module.

### Daemon (scripts/cc-live/)
- `main.go` — multi-session daemon with register/unregister/serve subcommands. Dep: `modernc.org/sqlite`.
- `go.mod` / `go.sum` — Go module.

## Environment Variables

### Server
- `CC_LIVE_API_KEY` (required) — shared secret for heartbeat/stop authentication.

### Daemon
- `CC_LIVE_ENDPOINT` — production: `http://<hosting-vm-tailscale-ip>:8080`
- `CC_LIVE_API_KEY` — same key as the server
- `CC_LIVE_SENSITIVE` — set to `1` to redact prompts for this session (read at `register` time)
- Silently exits 0 if not configured, so it never blocks Claude Code.

## Local Development

```bash
# Run server standalone
CC_LIVE_API_KEY=dev-secret go run ./services/cc-live/

# Or via docker compose from the repo root (includes Traefik + blog)
make dev

# Build daemon binary
make build-daemon

# Simulate session registration
echo '{"session_id":"s1","cwd":"/tmp"}' | CC_LIVE_ENDPOINT=http://localhost:8080 CC_LIVE_API_KEY=dev ~/.cc-live/cc-live-daemon register

# Test WebSocket (requires websocat or browser devtools)
websocat ws://localhost:8000/ws/live
```

## Build & Deploy

```bash
# From repo root
make build-daemon     # build local daemon binary to ~/.cc-live/
make build-cc-live    # build server image
make push-cc-live     # build + push to registry
make deploy-cc-live   # build + push + pulumi up
make deploy-all       # deploy blog + cc-live together
```

## Sensitive Sessions

Start Claude Code with `CC_LIVE_SENSITIVE=1 claude` to redact prompts for that session. The live status dot, metrics (tokens, tool calls, active time), and project name all work normally — only the `last_prompt` field is replaced with deterministic random noise of the same byte length before the heartbeat is sent.

- **Redaction happens in the daemon** (`scripts/cc-live/main.go`), so the real prompt never leaves the laptop.
- **Per-session flag** stored in the `sensitive` column in SQLite, so concurrent sessions can have different sensitivity levels.
- **Deterministic noise**: seeded from the prompt content, so the same prompt produces the same noise across ticks (avoids re-triggering the typewriter animation).
- **Frontend**: sensitive sessions show "Latest Prompt (redacted)" label and muted/dimmed styling on the noise text.

## Multi-Session Robustness

| Scenario | Behavior |
|----------|----------|
| Session A ends, Session B still running | `unregister` removes A, daemon sees B still active → stays online |
| Both sessions end | Both unregistered → daemon sends stop, exits |
| Laptop crashes (no clean SessionEnd) | Sessions stay in SQLite but transcripts stop updating → mtime ages past 15min → daemon sends stop |
| Claude Code idle >15min | Transcript mtime stale → daemon marks inactive even though session is registered |
| Daemon crashes | Next `register` (on SessionStart) detects stale PID, restarts daemon |
| Sensitive + normal concurrent | Each session's sensitivity is stored independently; one shows real prompts, the other shows noise |

## Infrastructure

- **Pulumi**: Defined in `~/projects/homeserver/src/services/index.ts` as a raw `docker.Container` (not `WebService`) for path-based Traefik routing + host port mapping.
- **Traefik routing**: `Host(howinator.io) && PathPrefix(/ws/live)` and `PathPrefix(/api/live)` → cc-live:8080. Higher specificity than the blog's `Host(howinator.io)` catch-all.
- **Port 8080**: Mapped to host for direct Tailscale access from the daemon.
