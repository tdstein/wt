#!/usr/bin/env bash
# SubagentStop hook: Handle subagent completion and conflict detection
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

# No conflicts - provide success context
cat <<EOF
{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "additionalContext": "Subagent '${AGENT_TYPE}' completed successfully in worktree '${WORKTREE_NAME}'. No conflicts detected.

To review changes: cd ${WORKTREE_NAME}/ && git diff main
To merge changes: cd ${WORKTREE_NAME}/ && git checkout main && git merge ${WORKTREE_NAME}
To clean up: wt remove ${WORKTREE_NAME}"
  }
}
EOF

exit 0
