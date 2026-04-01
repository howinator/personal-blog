#!/usr/bin/env bash
# personal-blog tmux window layout:
#   +------------------------------+
#   |        claude code           |  pane 1
#   +--------------+---------------+
#   |    shell     |    shell      |  pane 2, 3
#   +--------------+---------------+

oncreate() {
  local WORK_DIR="$1"

  # --- Worktree bootstrap ---
  local MAIN_REPO
  MAIN_REPO="$(git -C "$WORK_DIR" worktree list 2>/dev/null | head -1 | awk '{print $1}')"

  if [[ -n "$MAIN_REPO" && "$MAIN_REPO" != "$WORK_DIR" ]]; then
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

  # Pane 2: install node dependencies in worktrees (they don't have node_modules)
  if [[ -n "$MAIN_REPO" && "$MAIN_REPO" != "$WORK_DIR" ]]; then
    tmux send-keys -t :.2 "pnpm install --frozen-lockfile" Enter
  fi

  # Pane 1: claude code (prefill prompt, don't send)
  tmux select-pane -t :.1
  tmux send-keys -t :.1 "claude 'The ticket number is in the branch name. Read the ticket using the linear MCP and then write a plan for the fix'"
}

ondestroy() {
  local WORK_DIR="$1"
  # No services to clean up
}

"$@"
