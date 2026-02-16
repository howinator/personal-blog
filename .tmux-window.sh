#!/usr/bin/env bash
# .tmux-window.sh — Create tmux panes with SSH connections to a remote Firecracker VM
# Called by tmuxwt when opening a window for the personal-blog project.

WORK_DIR="${1:-$(tmux display-message -p '#{pane_current_path}')}"
BRANCH=$(git -C "$WORK_DIR" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")

if [[ -z "$BRANCH" ]]; then
    echo "Error: not in a git worktree" >&2
    exit 1
fi

# Configuration
MOG_HOST="mogclaude-host"                    # SSH config name (Tailscale IP)
MOG_CLI="sudo ~/mogclaude/cli/mogclaude"     # CLI path on remote host
VM_KEY="~/.ssh/mogclaude-vm"                 # Local copy of VM SSH key
ENV_FILE="${WORK_DIR}/.mogclaude-env"

# --- Determine if we should fork from a parent or create fresh ---

find_parent_env_id() {
    # Walk up the git history to find the parent branch's worktree with a .mogclaude-env
    # Use the merge base to find which branch we forked from
    local parent_branch
    for candidate in main master; do
        if git -C "$WORK_DIR" rev-parse --verify "$candidate" &>/dev/null; then
            parent_branch="$candidate"
            break
        fi
    done

    # Check all worktrees for a .mogclaude-env that isn't ours
    while IFS= read -r wt_line; do
        local wt_path
        wt_path=$(echo "$wt_line" | awk '{print $1}')
        [[ "$wt_path" == "$WORK_DIR" ]] && continue
        if [[ -f "${wt_path}/.mogclaude-env" ]]; then
            cat "${wt_path}/.mogclaude-env"
            return 0
        fi
    done < <(git -C "$WORK_DIR" worktree list 2>/dev/null)

    return 1
}

get_or_create_env() {
    local env_id=""

    # Check if we already have an env for this worktree
    if [[ -f "$ENV_FILE" ]]; then
        env_id=$(cat "$ENV_FILE")
        # Verify it still exists on remote
        if ssh "$MOG_HOST" "$MOG_CLI info $env_id" &>/dev/null; then
            echo "$env_id"
            return 0
        fi
        rm -f "$ENV_FILE"
    fi

    # Check if there's an env for this branch on the remote
    env_id=$(ssh "$MOG_HOST" "$MOG_CLI lookup $BRANCH" 2>/dev/null) || true
    if [[ -n "$env_id" ]]; then
        echo "$env_id" > "$ENV_FILE"
        echo "$env_id"
        return 0
    fi

    # Try to fork from parent
    local parent_env_id
    if parent_env_id=$(find_parent_env_id); then
        env_id=$(ssh "$MOG_HOST" "$MOG_CLI fork $parent_env_id --branch $BRANCH")
    else
        # Create fresh from template
        env_id=$(ssh "$MOG_HOST" "$MOG_CLI create --branch $BRANCH")
    fi

    echo "$env_id" > "$ENV_FILE"
    echo "$env_id"
}

# --- Main ---

echo "Setting up VM for branch: ${BRANCH}..."

ENV_ID=$(get_or_create_env)
ENV_IP=$(ssh "$MOG_HOST" "$MOG_CLI info $ENV_ID --field ip")

echo "Environment ${ENV_ID} at ${ENV_IP}"
echo "Domain: ${BRANCH}.claudmog.howinator.dev"

# SSH command base for connecting to the VM through the GCP host
VM_SSH="ssh -J ${MOG_HOST} -i ${VM_KEY} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@${ENV_IP}"

# Pane 1 (top, 70%): Claude Code inside VM
tmux send-keys "${VM_SSH} -t 'cd ~/personal-blog && claude'" Enter

# Pane 2 (bottom-left, 30%): docker compose
tmux split-window -v -p 30
tmux send-keys "${VM_SSH} -t 'cd ~/personal-blog && make dev'" Enter

# Pane 3 (bottom-right): shell
tmux split-window -h
tmux send-keys "${VM_SSH} -t 'cd ~/personal-blog && bash'" Enter

# Select top pane (Claude Code)
tmux select-pane -t :.1
