#!/usr/bin/env bash
# SessionEnd hook: Clean up session worktree
set -euo pipefail

# Read hook input from stdin
INPUT=$(cat)

# Parse session info
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // ""')
REASON=$(echo "$INPUT" | jq -r '.reason // "other"')

# Check if wt is available
if ! command -v wt &> /dev/null; then
    exit 0
fi

# Check if we're in a wt-managed repo
if [ ! -d ".bare" ] || [ ! -d ".wt" ]; then
    exit 0
fi

# Generate session worktree name
WORKTREE_NAME="session-${SESSION_ID:0:8}"

# Check if this session worktree exists
if ! wt list | grep -q "^${WORKTREE_NAME}"; then
    # Try to prune any stale worktrees older than 24 hours
    wt prune --older-than 24h &>/dev/null || true
    exit 0
fi

# Check if there are uncommitted changes
cd "$WORKTREE_NAME" 2>/dev/null || exit 0

if ! git diff-index --quiet HEAD -- 2>/dev/null; then
    # Uncommitted changes exist - don't remove, just log
    echo "Session worktree '${WORKTREE_NAME}' has uncommitted changes - preserving for review" >&2
    exit 0
fi

# Check if there are unpushed commits
UNPUSHED=$(git rev-list @{u}.. 2>/dev/null | wc -l || echo "0")
if [ "$UNPUSHED" -gt 0 ]; then
    # Unpushed commits exist - don't remove
    echo "Session worktree '${WORKTREE_NAME}' has ${UNPUSHED} unpushed commit(s) - preserving" >&2
    exit 0
fi

# Safe to remove - no uncommitted changes or unpushed commits
cd .. 2>/dev/null || exit 0
if wt remove "$WORKTREE_NAME" 2>/dev/null; then
    echo "Cleaned up session worktree: ${WORKTREE_NAME}" >&2
else
    echo "Could not remove session worktree: ${WORKTREE_NAME}" >&2
fi

# Also prune any stale worktrees older than 7 days
wt prune --older-than 7d &>/dev/null || true

exit 0
