# Plan: Living Claude Code Session Stats Blog Post

## Context

Create an auto-updating blog post on howinator.io that tracks stats from every Claude Code session. A `SessionEnd` hook fires a Python script (managed via `uv`) that parses the session transcript JSONL, extracts metrics, generates an LLM summary via Opus, and writes to a Hugo data file. A shortcode renders the data as a summary dashboard + expandable session cards.

## Architecture

```
CC Session Ends
  → SessionEnd hook (in ~/.claude/settings.json)
  → `uv run` executes Python script
  → Script reads transcript JSONL path from stdin payload
  → Computes stats (prompts, tool calls, tokens, active time)
  → Calls `claude -p --model haiku` to generate 1-sentence summary
  → Appends to data/cc_sessions.json
  → Hugo rebuilds → blog post updates
```

## Files

| File | Purpose |
|------|---------|
| `scripts/cc-stats/pyproject.toml` | Python project config (uv-managed) |
| `scripts/cc-stats/cc_stats/__init__.py` | Package init |
| `scripts/cc-stats/cc_stats/main.py` | Core script: parses transcript, computes stats, updates data file |
| `data/cc_sessions.json` | Session data store (JSON array + aggregated totals) |
| `layouts/shortcodes/cc-sessions.html` | Hugo shortcode rendering dashboard + accordion |
| `assets/css/cc-stats.css` | Styles for stat boxes and session cards |
| `content/blog/claude-code-sessions.md` | The living blog post |

## Data Schema (`data/cc_sessions.json`)

```json
{
  "sessions": [
    {
      "session_id": "uuid",
      "date": "2026-02-12T10:00:00Z",
      "date_display": "Feb 12, 2026",
      "summary": "LLM-generated one-sentence summary of the session",
      "project": "personal-blog",
      "cwd": "/Users/howie/projects/personal-blog",
      "num_user_prompts": 5,
      "num_tool_calls": 23,
      "total_input_tokens": 45000,
      "total_output_tokens": 8200,
      "total_tokens": 53200,
      "total_tokens_display": "53.2k",
      "active_time_seconds": 270,
      "active_time_display": "4m 30s",
      "cc_version": "2.1.37"
    }
  ],
  "totals": {
    "session_count": 1,
    "total_tokens": 53200,
    "total_tokens_display": "53.2k",
    "total_tool_calls": 23,
    "total_active_time_seconds": 270,
    "total_active_time_display": "4m 30s"
  }
}
```

## Stats Computed from Transcript

- **User prompts**: Count entries where `type == "user"` and `message.content` contains text
- **Tool calls**: Count `{type: "tool_use"}` blocks within assistant message content arrays
- **Tokens**: Sum all token fields from assistant message `usage` objects
- **Active time**: Sum of capped (5min max) gaps between consecutive transcript entries
- **Summary**: LLM-generated 1-sentence summary via `claude -p --model haiku`

## Hook Configuration (`~/.claude/settings.json`)

```json
{
  "hooks": {
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "uv run --project /Users/howie/projects/personal-blog/scripts/cc-stats process-cc-session",
            "timeout": 120
          }
        ]
      }
    ]
  }
}
```

## Future TODOs

- Cache LLM summaries to avoid re-generation on reprocessing
- Frustration/difficulty score from transcript analysis
- Transcript links (a la Simon Willison's claude-code-transcripts)
