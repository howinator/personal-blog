# cc-stats — Claude Code Session Stats Processor

## Overview

Python script (uv-managed) that processes Claude Code session transcripts into stats for the blog. Runs as a `SessionEnd` hook — parses JSONL transcripts, computes metrics, generates an LLM summary via the Anthropic API, and writes to `data/cc_sessions.json` for Hugo to render.

## Project Structure

```
scripts/cc-stats/
  pyproject.toml          # uv project config, hatchling build backend
  cc_stats/
    __init__.py
    main.py               # Entry point: process-cc-session
  AGENTS.md               # This file
  PLAN.md                 # Architecture/design doc
```

## How It Works

1. **Input**: JSON payload on stdin from the SessionEnd hook:
   ```json
   {"session_id": "...", "transcript_path": "/path/to.jsonl", "cwd": "...", "hook_event_name": "SessionEnd"}
   ```

2. **Transcript parsing**: Reads the JSONL file, counts user prompts (text entries), tool_use blocks, and sums token usage from assistant messages.

3. **Active time**: Sums capped (5min max) gaps between consecutive transcript entries.

4. **Summary generation**: Calls the Anthropic API (`claude-haiku-4-5-20251001`) via `urllib.request` using `$ANTHROPIC_API_KEY`. Falls back to first user prompt truncated at 120 chars.

5. **Output**: Atomically writes (temp file + rename) to `data/cc_sessions.json` with upsert semantics (idempotent by session_id).

## Key Design Decisions

- **No external dependencies** — stdlib only (`json`, `urllib.request`, `pathlib`, etc.)
- **Always exits 0** — wrapped in try/except so it never blocks Claude Code
- **Skips empty sessions** — sessions with 0 user prompts or 0 total tokens are ignored
- **Filters system text** — strips `[Request interrupted...]`, `<local-command-caveat>`, `<system-reminder>` from user texts before building summaries
- **Atomic writes** — prevents data corruption from concurrent sessions

## Environment Variables

- `ANTHROPIC_API_KEY` — required for LLM summary generation (falls back to truncated first prompt)
- `CC_STATS_BLOG_ROOT` — blog repo root (default: `~/projects/personal-blog`)

## Development

```bash
uv sync --project scripts/cc-stats                    # Install
uv sync --project scripts/cc-stats --reinstall-package cc-stats  # Reinstall after code changes

# Test with a real transcript:
echo '{"session_id":"test","transcript_path":"/path/to/transcript.jsonl","cwd":"/tmp","hook_event_name":"SessionEnd"}' \
  | uv run --project scripts/cc-stats process-cc-session
```

## Hook Configuration

Installed in `~/.claude/settings.json`:
```json
{
  "hooks": {
    "SessionEnd": [{
      "hooks": [{
        "type": "command",
        "command": "uv run --project /Users/howie/projects/personal-blog/scripts/cc-stats process-cc-session",
        "timeout": 120
      }]
    }]
  }
}
```
