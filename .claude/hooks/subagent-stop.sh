#!/usr/bin/env bash
# SubagentStop hook: Handle subagent completion with autonomous merge-back
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

# Generate worktree name
AGENT_ID_SHORT="${AGENT_ID:6:8}"
AGENT_TYPE_LOWER=$(echo "$AGENT_TYPE" | tr '[:upper:]' '[:lower:]')
WORKTREE_NAME="${AGENT_TYPE_LOWER}-${AGENT_ID_SHORT}"

# Check if this worktree exists
if ! wt list | grep -q "^${WORKTREE_NAME}"; then
    # Worktree doesn't exist, nothing to do
    exit 0
fi

# Check for conflicts
CONFLICT_OUTPUT=$(wt check "$WORKTREE_NAME" 2>&1 || true)

# Check if conflicts exist (look for "Merge conflicts: true" or absence of "[OK]")
if echo "$CONFLICT_OUTPUT" | grep -q "Merge conflicts: true" || ! echo "$CONFLICT_OUTPUT" | grep -q "\[OK\] No conflicts"; then
    # Conflicts found - notify Claude via decision block
    CONFLICT_DETAILS=$(echo "$CONFLICT_OUTPUT" | grep -A 20 "Conflicting files:" || echo "See wt check output for details")

    cat <<EOF
{
  "decision": "block",
  "reason": "Subagent '${AGENT_TYPE}' completed but conflicts detected in worktree '${WORKTREE_NAME}'.

${CONFLICT_DETAILS}

Actions needed:
1. Review conflicts: cd ${WORKTREE_NAME}/ && git status
2. Resolve conflicts manually
3. Or discard changes: wt remove ${WORKTREE_NAME}

The subagent's work is preserved in the worktree for review."
}
EOF
    exit 0
fi

# No conflicts detected - proceed with auto-merge
CURRENT_BRANCH=$(git branch --show-current 2>/dev/null)

# Get agent branch name (same as worktree name)
AGENT_BRANCH="$WORKTREE_NAME"

# Attempt merge with --no-ff to preserve agent work as distinct commit
MERGE_OUTPUT=$(git merge --no-ff "$AGENT_BRANCH" -m "Merge subagent work from ${WORKTREE_NAME}

Co-Authored-By: Claude (${AGENT_TYPE} agent)" 2>&1 || true)

# Check if merge succeeded
if echo "$MERGE_OUTPUT" | grep -q "Already up to date"; then
    # No changes to merge - clean up worktree
    wt remove "$WORKTREE_NAME" 2>/dev/null || true
    cat <<EOF
{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "additionalContext": "Subagent '${AGENT_TYPE}' completed. No changes to merge, cleaned up worktree '${WORKTREE_NAME}'."
  }
}
EOF
elif git diff --quiet HEAD@{1} HEAD 2>/dev/null || [ ! -f .git/MERGE_HEAD ]; then
    # Merge completed successfully
    wt remove "$WORKTREE_NAME" 2>/dev/null || true
    cat <<EOF
{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "additionalContext": "✓ Subagent '${AGENT_TYPE}' completed successfully.
Auto-merged ${WORKTREE_NAME} → ${CURRENT_BRANCH} and cleaned up worktree.

Merged changes are now in your current branch."
  }
}
EOF
else
    # Merge failed - preserve worktree for manual resolution
    cat <<EOF
{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "additionalContext": "⚠ Subagent '${AGENT_TYPE}' completed but auto-merge failed.
Preserved worktree '${WORKTREE_NAME}' for manual merge.

To merge manually:
  git merge ${AGENT_BRANCH}
  wt remove ${WORKTREE_NAME}"
  }
}
EOF
fi

exit 0
