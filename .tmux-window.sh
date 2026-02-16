#!/usr/bin/env bash
# personal-blog tmux window layout:
#   +------------------------------+
#   |        claude code           |
#   +--------------+---------------+
#   |    shell     |    shell      |
#   +--------------+---------------+

WORK_DIR="${1:-$(tmux display-message -p '#{pane_current_path}')}"

# Create bottom pane (30% height)
tmux split-window -v -c "$WORK_DIR" -p 30

# Split bottom pane into two side-by-side
tmux split-window -h -c "$WORK_DIR"

# Start claude code in the top pane
tmux select-pane -t :.1
tmux send-keys -t :.1 "claude" Enter
