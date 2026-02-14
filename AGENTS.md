# AGENTS.md - Personal Blog Monorepo (howinator.io)

## Overview

Monorepo for https://howinator.io/ — a Hugo static site blog with a live Claude Code status system. Three main components:

| Directory | Description |
|-----------|-------------|
| `site/` | Hugo blog (content, theme, templates, static assets) |
| `services/cc-live/` | Go WebSocket relay server (runs on hosting VM) |
| `scripts/cc-live/` | Heartbeat daemon (runs on laptop, triggered by Claude Code hooks) |

Each component has its own `AGENTS.md` with detailed instructions.

## Key Files (root)

- `Makefile` — Build, push, deploy targets for all components
- `compose.yaml` — Local dev stack (Traefik + blog + cc-live)
- `site/config.toml` — Hugo site configuration
- `site/content/blog/` — All blog posts

## Build & Deploy

Uses Podman for container builds and a Zot registry. Managed via Makefile:

```bash
make build            # Build blog image (runs sync first)
make push             # Build + push blog
make deploy           # Build + push blog + pulumi up

make build-cc-live    # Build cc-live server image
make deploy-cc-live   # Build + push cc-live + pulumi up

make deploy-all       # Deploy blog + cc-live together

make dev              # Local dev stack (docker compose)
make dev-heartbeat    # Test heartbeat locally
make dev-down         # Tear down local stack
```

- **Registry**: `zot.ui.sparky.best/{personal-blog,cc-live}`
- **Platform**: `linux/amd64`
- **Infra**: Pulumi IaC in `~/projects/homeserver`

## Daemon Makefile targets

```bash
make build-daemon     # Compile daemon binary to ~/.cc-live/
make restart-daemon   # Build + kill old daemon (auto-restarts on next hook)
make reset-daemon     # Build + kill + wipe SQLite DB + rotate logs
make sync             # Build daemon + run sync (reparse all transcripts -> SQLite -> JSON)
```

## Live Status System (cc-live)

Go WebSocket sidecar that shows a live status dot in the nav when a Claude Code session is active. See `services/cc-live/AGENTS.md` for full details.

**Components:**
- `services/cc-live/` — Go WebSocket service (server-side)
- `scripts/cc-live/` — Heartbeat daemon (client-side, triggered by Claude Code hooks)
- `site/assets/js/live-status.js` — Browser WebSocket client
- `site/layouts/shortcodes/cc-status-dot.html` — Inline status dot shortcode

**Claude Code hooks** (`~/.claude/settings.json`):
- `SessionStart` -> `cc-live-daemon register`
- `SessionEnd` -> `cc-live-daemon unregister`

**Sensitive sessions:** Start with `CC_LIVE_SENSITIVE=1 claude` to redact prompts.

**Debugging:** See `services/cc-live/docs/debugging.md`.

## Claude Code Session Stats

Session stats are managed by the cc-live daemon's `sync` subcommand. `cc-live-daemon sync` reparses all transcripts from `~/.claude/projects/` into the `session_stats` SQLite table, generates LLM summaries via the Anthropic API, and exports `site/data/cc_sessions.json` for Hugo. The `make build` target runs `make sync` automatically before the container build.
