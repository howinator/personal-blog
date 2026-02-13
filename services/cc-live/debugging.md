# cc-live Debugging Guide

Known issues, root causes, and fixes discovered during development.

## SQLITE_BUSY on SessionStart hooks

**Symptom:** Claude Code shows `SessionStart:resume hook error` (or `SessionStart:startup`). The debug log (`~/.claude/debug/<session-id>.txt`) shows:

```
Hook SessionStart:resume (SessionStart) error:
main.go:97: creating table: database is locked (5) (SQLITE_BUSY)
```

**Root cause:** The daemon's `serve` process holds `~/.cc-live/state.db` open. When `register` or `unregister` runs in a separate process (from a hook), `openDB()` calls `CREATE TABLE IF NOT EXISTS` which contends for a write lock. The `modernc.org/sqlite` driver does NOT support `_busy_timeout` or `_journal_mode` as connection string parameters — they must be set via `PRAGMA` statements. Without a working busy timeout, SQLite returns `SQLITE_BUSY` immediately and `log.Fatalf` kills the process.

**Fix:** Set pragmas explicitly and retry the `CREATE TABLE`:

```go
db.Exec(`PRAGMA journal_mode=WAL`)
db.Exec(`PRAGMA busy_timeout=10000`)
// + retry loop for CREATE TABLE (5 attempts, 200ms apart)
```

**Debugging:** Check the Claude Code debug log for `SQLITE_BUSY` errors:
```bash
grep "SQLITE_BUSY\|hook error" ~/.claude/debug/<session-id>.txt
```

## Zombie live session cards (0 tokens, showing project name)

**Symptom:** A live session card appears on the `/claude-log/` page with 0 tokens, 0 prompts, and no useful data. The card persists because the transcript file keeps getting updated.

**Root cause:** Claude Code maintains internal transcript files (containing only `file-history-snapshot` entries) that are NOT real conversation sessions. These get registered via the `SessionStart` hook and their transcript files keep getting modified (staying within the 15-minute activity timeout), but they contain no user/assistant conversation entries.

**Fix:** In `checkSessions()`, skip sessions where the parsed transcript has 0 user prompts AND 0 total tokens:

```go
if tracker.metrics.UserPrompts == 0 && tracker.metrics.TotalTokens == 0 {
    continue
}
```

**Debugging:** Check what's in the SQLite sessions table and inspect the transcript:
```bash
sqlite3 ~/.cc-live/state.db "SELECT * FROM sessions;"
# For each transcript_path, check entry types:
python3 -c "
import json
for line in open('<transcript_path>'):
    d = json.loads(line.strip())
    print(d.get('type', '?'))
"
```

If all entries are `file-history-snapshot`, it's a non-conversation transcript.

## Scientific notation in Hugo templates

**Symptom:** Token counts display as `7` instead of `70.3M` in stat boxes, or show as `6.21e+07` in session detail tables.

**Root cause:** Go's JSON parser decodes numbers as `float64`. Hugo's template engine renders large float64 values using `%g` format, which switches to scientific notation for large numbers (e.g., `7.03e+07`). JavaScript's `parseInt("7.03e+07")` stops at the decimal point and returns `7`.

**Fix (Hugo templates):** Use `{{ printf "%.0f" .total_tokens }}` to force integer formatting. For display values, use the pre-formatted `_display` fields from `cc_sessions.json`.

**Fix (JavaScript):** Use `Math.round(Number(value))` instead of `parseInt(value, 10)`. `Number("7.03e+07")` correctly returns `70300000`.

## Typewriter animation replays on every heartbeat tick

**Symptom:** The "Latest Prompt" typewriter animation re-plays the same text every 30 seconds instead of only once when a new prompt arrives.

**Root cause:** `createLiveCard()` rebuilds `detailsDiv.innerHTML` on every WS message, destroying the prompt element and its `data-prompt` attribute. The next comparison sees `prevPrompt = ''` (fresh element) vs the current `last_prompt`, triggering the animation again.

**Fix:** Track the last-seen prompt per session in a JavaScript object (`lastPromptSeen` map) instead of reading from DOM attributes that get destroyed by `innerHTML` rebuilds.

## Daemon binary not picked up after code changes

**Symptom:** Code changes to `scripts/cc-live/main.go` have no effect. The daemon keeps running old behavior.

**Root cause:** The daemon runs from `~/.cc-live/cc-live-daemon` (a compiled binary). Editing the Go source doesn't affect the running binary. The daemon must be rebuilt and the old process killed.

**Fix:**
```bash
make restart-daemon   # rebuild + kill old process
# or for a full reset (wipe DB + rotate logs):
make reset-daemon
```

The next `SessionStart` hook will auto-start the new daemon.

## Useful commands

```bash
# Check daemon status
ps -p $(cat ~/.cc-live/daemon.pid) -o pid,comm 2>/dev/null || echo "not running"

# View daemon logs (only errors are logged; successful heartbeats are silent)
tail -20 ~/.cc-live/daemon.log

# Check registered sessions
sqlite3 ~/.cc-live/state.db "SELECT * FROM sessions;"

# Check Claude Code debug log for hook errors
grep "hook error\|SQLITE_BUSY" ~/.claude/debug/*.txt

# Manual heartbeat test (local dev)
make dev-heartbeat

# Full daemon reset
make reset-daemon
```
