#!/usr/bin/env bash
# personal-blog tmux window layout:
#   +------------------------------+
#   |        claude code           |
#   +--------------+---------------+
#   |    shell     |    shell      |
#   +--------------+---------------+

WORK_DIR="${1:-$(tmux display-message -p '#{pane_current_path}')}"

# --- Worktree bootstrap ---
# Detect the main repo root (the repo this worktree was created from)
MAIN_REPO="$(git -C "$WORK_DIR" worktree list 2>/dev/null | head -1 | awk '{print $1}')"

IS_WORKTREE=false
if [[ -n "$MAIN_REPO" && "$MAIN_REPO" != "$WORK_DIR" ]]; then
  IS_WORKTREE=true

  # Copy gitignored files that are needed for the project to function

  # .claude/settings.local.json (MCP server permissions)
  if [[ -f "$MAIN_REPO/.claude/settings.local.json" ]]; then
    mkdir -p "$WORK_DIR/.claude"
    cp "$MAIN_REPO/.claude/settings.local.json" "$WORK_DIR/.claude/settings.local.json"
  fi

  # site/data/cc_sessions.json (session data for Hugo templates)
  if [[ -f "$MAIN_REPO/site/data/cc_sessions.json" ]]; then
    mkdir -p "$WORK_DIR/site/data"
    cp "$MAIN_REPO/site/data/cc_sessions.json" "$WORK_DIR/site/data/cc_sessions.json"
  fi
fi

# --- Tmux layout ---

# Create bottom pane (30% height)
tmux split-window -v -c "$WORK_DIR" -p 30

# Split bottom pane into two side-by-side
tmux split-window -h -c "$WORK_DIR"

# Start claude code in the top pane
tmux select-pane -t :.1
tmux send-keys -t :.1 "claude" Enter
