# Deploying the cc-live → claug Migration

Step-by-step deployment instructions. Do these in order.

## Prerequisites

- `claug` CLI installed (`cd ~/projects/claug/cmd/claug && go install .`)
- Authenticated with claug: `claug login` (creates `~/.config/claug/auth.json` with API key)
- `leaderboard_opt_in = true` for your user in claug (so sessions publish to public topics)

## Step 1: Deploy claug server changes

The claug repo has changes on the `HOW-29-big-daemon-upgrade` branch:
- API key auth on `GET /api/sessions` (was JWT-only)
- `tool_counts` JSONB column + tracking throughout the stack
- CORS origins: added `howinator.io`, `howinator.dev`
- User-scoped public WebSocket topic (`?user=<id>` filter)
- Stop events now publish to public topics

```bash
cd ~/projects/claug
git checkout HOW-29-big-daemon-upgrade
make deploy
```

This builds + pushes container images and runs `pulumi up`. The `004_tool_counts.sql` migration runs automatically on server startup (embedded migrations, auto-applied).

## Step 2: Backfill historical sessions

Push all historical session data from local SQLite into claug's PostgreSQL.

```bash
cd ~/projects/personal-blog
git checkout claug-switchover

# Preview what will be sent (no API calls)
cd scripts/backfill-sessions && go run . --dry-run

# If it looks right, run for real
go run .
```

This reads `~/.cc-live/state.db` and POSTs sessions as heartbeats to `POST /api/sessions/heartbeat` using your API key. Each session gets upserted into claug's `session_stats` table.

Verify the backfill worked:
```bash
cd ../build-sessions
CC_STATS_BLOG_ROOT="$(cd ../../site && pwd)" go run .
# Check site/data/cc_sessions.json — compare session count against old file
```

## Step 3: Deploy blog changes

```bash
cd ~/projects/personal-blog
git checkout claug-switchover
make deploy
```

This runs `make sync` (fetches sessions from claug API → `cc_sessions.json`), then builds the Hugo site + container image, pushes it, and runs `pulumi up`.

## Step 4: Verify

1. **Historical data**: Visit https://howinator.io/claude-log/ — all sessions should appear with correct stats, token counts, tool breakdowns
2. **Live dot**: Start a new Claude Code session — the nav dot should turn green within ~30 seconds
3. **Live session card**: The session should appear as a live card with updating metrics and typewriter prompt animation
4. **Stop**: End the session — the live card should disappear (immediately via stop event, or after 90s timeout)

## Step 5: Remove cc-live hooks

Both `cc-live-daemon` and `claug` hooks are currently running in parallel. Once everything is verified, remove the cc-live hooks.

Edit `~/.claude/settings.json` — in both `SessionStart` and `SessionEnd`, remove the `cc-live-daemon` hook entries:

```json
// REMOVE these from SessionStart and SessionEnd:
{
  "type": "command",
  "command": "$HOME/.cc-live/cc-live-daemon register",
  "timeout": 5
}
{
  "type": "command",
  "command": "$HOME/.cc-live/cc-live-daemon unregister",
  "timeout": 5
}
```

Keep the `claug start` and `claug stop` hooks.

## Step 6: Clean up cc-live (when ready)

Only do this after you're confident the migration is fully working.

### Repo cleanup
```bash
cd ~/projects/personal-blog
git checkout claug-switchover
rm -rf services/cc-live/
rm -rf scripts/cc-live/
rm -rf scripts/backfill-sessions/   # one-time script, no longer needed
rm -rf tests/integration/           # cc-live integration tests
rm -f compose.test.yaml             # cc-live test compose
git add -A && git commit -m "Remove cc-live and backfill script"
```

### Local machine cleanup
```bash
# Stop the cc-live daemon if running
kill $(cat ~/.cc-live/daemon.pid 2>/dev/null) 2>/dev/null

# Archive the SQLite DB (optional, just in case)
cp ~/.cc-live/state.db ~/.cc-live/state.db.bak

# Remove everything
rm -rf ~/.cc-live/
```

## Rollback

If something goes wrong after Step 3:

- **Blog shows no sessions**: Check that `claug login` was run and `~/.config/claug/auth.json` has an `api_key`. Run `make sync` manually to see errors.
- **Live dot not working**: Check browser console for WebSocket errors. Verify claug server is running and `leaderboard_opt_in = true` for your user.
- **Quick rollback**: Revert the blog to `main` branch and redeploy — cc-live hooks are still running in parallel until Step 5.
