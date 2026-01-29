#!/usr/bin/env bash
# SubagentStart hook: Create isolated worktree for subagent
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

# Create the worktree for this subagent
if wt add "$WORKTREE_NAME" 2>/dev/null; then
    # Success - log for debugging (not shown to Claude)
    echo "Created subagent worktree: ${WORKTREE_NAME}/" >&2
else
    # Failed - not critical, subagent can work in current directory
    echo "Could not create subagent worktree: ${WORKTREE_NAME}" >&2
fi

exit 0
