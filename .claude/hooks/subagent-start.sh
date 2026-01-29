#!/usr/bin/env bash
# SubagentStart hook: Create isolated worktree for complex/long-running agents
set -euo pipefail

# Read hook input from stdin
INPUT=$(cat)

# Parse subagent info
AGENT_ID=$(echo "$INPUT" | jq -r '.agent_id // ""')
AGENT_TYPE=$(echo "$INPUT" | jq -r '.agent_type // ""')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // ""')

# Check if wt is available
if ! command -v wt &> /dev/null; then
    exit 0
fi

# Check if we're in a wt-managed repo
if [ ! -d ".bare" ] || [ ! -d ".wt" ]; then
    exit 0
fi

# Filter: Only create worktrees for Plan and Explore agents
case "$AGENT_TYPE" in
  Plan|Explore)
    # These agents benefit from isolation
    ;;
  *)
    # All other agents work in current directory
    exit 0
    ;;
esac

# Generate worktree name based on agent type and ID
# Format: <agent-type>-<agent-id-short>
AGENT_ID_SHORT="${AGENT_ID:6:8}"  # Extract 8 chars from agent ID
AGENT_TYPE_LOWER=$(echo "$AGENT_TYPE" | tr '[:upper:]' '[:lower:]')
WORKTREE_NAME="${AGENT_TYPE_LOWER}-${AGENT_ID_SHORT}"

# Check if worktree already exists
if wt list | grep -q "^${WORKTREE_NAME}"; then
    # Already exists, skip creation
    exit 0
fi

# Detect base branch from current context
# This allows agents to work relative to the user's current branch
BASE_BRANCH=$(git branch --show-current 2>/dev/null || echo "")
if [ -z "$BASE_BRANCH" ]; then
    BASE_BRANCH="main"  # Fallback if detached HEAD
fi

# Create the worktree for this subagent with detected base branch
if wt add "$WORKTREE_NAME" "$BASE_BRANCH" 2>/dev/null; then
    # Success - log for debugging (not shown to Claude)
    echo "Created subagent worktree: ${WORKTREE_NAME}/ (base: ${BASE_BRANCH})" >&2
else
    # Failed - not critical, subagent can work in current directory
    echo "Could not create subagent worktree: ${WORKTREE_NAME}" >&2
fi

exit 0
