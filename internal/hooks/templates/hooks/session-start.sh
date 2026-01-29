#!/usr/bin/env bash
# SessionStart hook: Create isolated worktree for this session
set -euo pipefail

# Read hook input from stdin
INPUT=$(cat)

# Parse session info
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // ""')
SOURCE=$(echo "$INPUT" | jq -r '.source // "startup"')
AGENT_TYPE=$(echo "$INPUT" | jq -r '.agent_type // ""')

# Check if wt is available
if ! command -v wt &> /dev/null; then
    exit 0
fi

# Check if we're in a wt-managed repo
if [ ! -d ".bare" ] || [ ! -d ".wt" ]; then
    exit 0
fi

# Check if we're already in a session worktree
CURRENT_DIR=$(basename "$(pwd)")
if [[ "$CURRENT_DIR" == session-* ]] || [[ "$CURRENT_DIR" == *-agent-* ]]; then
    # Already in a session worktree, skip creation
    exit 0
fi

# Generate worktree name
if [ -n "$AGENT_TYPE" ]; then
    WORKTREE_NAME="${AGENT_TYPE}-${SESSION_ID:0:8}"
else
    WORKTREE_NAME="session-${SESSION_ID:0:8}"
fi

# Check if worktree already exists
if wt list | grep -q "^${WORKTREE_NAME}"; then
    # Worktree already exists, provide context
    cat <<EOF
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Using existing worktree: ${WORKTREE_NAME}/"
  }
}
EOF
    exit 0
fi

# Create the worktree
if wt add "$WORKTREE_NAME" 2>/dev/null; then
    # Success - provide context to Claude
    cat <<EOF
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Created isolated worktree: ${WORKTREE_NAME}/

You are working in an isolated git worktree. This prevents conflicts with other concurrent Claude Code sessions or agents. Your changes are isolated until explicitly merged to the main branch.

To see all active worktrees: wt list
To check for conflicts before merging: wt check ${WORKTREE_NAME}
To sync with main branch: wt sync ${WORKTREE_NAME}"
  }
}
EOF
else
    # Failed to create worktree - not critical, continue without isolated workspace
    exit 0
fi

exit 0
