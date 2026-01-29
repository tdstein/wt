#!/usr/bin/env bash
# SessionStart hook: Provide context about wt availability
set -euo pipefail

# Read hook input from stdin
INPUT=$(cat)

# Parse session info
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // ""')
SOURCE=$(echo "$INPUT" | jq -r '.source // "startup"')

# Check if wt is available
if ! command -v wt &> /dev/null; then
    exit 0
fi

# Check if we're in a wt-managed repo
if [ ! -d ".bare" ] || [ ! -d ".wt" ]; then
    exit 0
fi

# Get current branch for context
CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")

# Provide context about wt (no automatic worktree creation)
cat <<EOF
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Working in wt-managed repository (branch: ${CURRENT_BRANCH})

You are in the current worktree. Use 'wt' commands to manage isolated workspaces when needed:
  • wt add <name> - Create isolated worktree for complex/long-horizon tasks
  • wt list - View all active worktrees
  • wt status - Dashboard of agent activity

For most tasks, working in the current directory is recommended. Only create isolated worktrees for complex work that modifies many files or requires long-term isolation."
  }
}
EOF

exit 0
