# AGENTS.md - cc-live Daemon (scripts/cc-live/)

## Overview

Go binary (`cc-live-daemon`) that manages Claude Code session lifecycle and metrics. Runs on the laptop, triggered by Claude Code hooks. Compiles to `~/.cc-live/cc-live-daemon`.

## Subcommands

- **`register`** — Called by `SessionStart` hook. Reads hook payload from stdin, inserts session into SQLite, starts background daemon if not running.
- **`unregister`** — Called by `SessionEnd` hook. Removes session from SQLite. If no sessions remain, sends stop and kills daemon.
- **`serve`** — Long-running background process. Every 30s parses transcript JSONL files incrementally, computing per-session metrics. Sends rich heartbeat payload to the cc-live server. Auto-exits after 30 minutes with zero registered sessions.
- **`sync`** — Batch reparse all transcripts from `~/.claude/projects/` into `session_stats` SQLite table. Generates LLM summaries via Anthropic API. Exports `site/data/cc_sessions.json` for Hugo.

## Files

- `main.go` — Single-file daemon with all subcommands. Deps: `modernc.org/sqlite` (pure Go SQLite).
- `go.mod` / `go.sum` — Go module.

## SQLite Schema (`~/.cc-live/state.db`)

**`sessions`** — Active sessions (used by serve/register/unregister):
- `session_id`, `cwd`, `transcript_path`, `sensitive`, `registered_at`

**`session_stats`** — Historical metrics for all sessions (used by sync):
- `session_id`, `project`, `model`, `total_tokens`, `input_tokens`, `output_tokens`, `cache_read`, `cache_write`, `user_prompts`, `tool_calls`, `active_time_seconds`, `started_at`, `ended_at`, `summary`, `cwd`
- Display fields: `total_tokens_display`, `input_tokens_display`, `output_tokens_display`, `cache_read_display`, `cache_write_display`, `active_time_display`

## Transcript Parsing

- Transcripts are JSONL files in `~/.claude/projects/`
- Top-level fields: `sessionId`, `cwd`, `version`, `timestamp`, `type`, `message`
- `processEntry()` extracts metadata from top-level fields (not inside `message`)
- `processTranscript()` is shared between `register` and `sync` — takes optional `transcriptOverrides`

## Environment Variables

- `CC_LIVE_ENDPOINT` — Server URL (e.g., `http://<hosting-vm-tailscale-ip>:8080`)
- `CC_LIVE_API_KEY` — Shared secret for heartbeat/stop authentication
- `CC_LIVE_SENSITIVE` — Set to `1` at `register` time to redact prompts for that session
- `CC_STATS_BLOG_ROOT` — Override blog root path (default: `~/projects/personal-blog`)
- Silently exits 0 if endpoint/key not configured, so it never blocks Claude Code.

## Build

```bash
make build-daemon     # Compile to ~/.cc-live/cc-live-daemon
make restart-daemon   # Build + kill old daemon
make reset-daemon     # Build + kill + wipe SQLite DB + rotate logs
make sync             # Build + run sync (reparse all transcripts)
```

## Key Patterns

- Summary generation: `cleanUserTexts()` filters system prefixes, then Anthropic Haiku API call with 10s (live) or 30s (sync) timeout
- SQLite upsert preserves existing non-empty summary on conflict
- `formatTokens()`: >=10M -> "X.XM", else commas (e.g. "1,234,567")
- Sensitive sessions: deterministic random noise seeded from prompt content replaces `last_prompt`
