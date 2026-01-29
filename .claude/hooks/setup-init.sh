#!/usr/bin/env bash
# Setup hook: Initialize wt repository structure
set -euo pipefail

# Read hook input from stdin
INPUT=$(cat)

# Parse trigger type
TRIGGER=$(echo "$INPUT" | jq -r '.trigger // "init"')

# Check if wt is available
if ! command -v wt &> /dev/null; then
    echo "wt command not found - skipping wt initialization" >&2
    exit 0
fi

# Check if we're already in a wt-managed repo
if [ -d ".bare" ] && [ -d ".wt" ]; then
    echo "Repository already initialized with wt structure" >&2
    exit 0
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir &> /dev/null; then
    echo "Not a git repository - skipping wt initialization" >&2
    exit 0
fi

# Initialize wt structure for this repository
echo "Initializing wt structure for parallel agent workflows..." >&2

# Run wt init in the current directory
if wt init .; then
    # Output context for Claude using JSON format
    cat <<EOF
{
  "hookSpecificOutput": {
    "hookEventName": "Setup",
    "additionalContext": "Repository initialized with wt for parallel agent workflows. Agent worktrees will be created in separate directories to prevent conflicts."
  }
}
EOF
else
    echo "Failed to initialize wt structure" >&2
    exit 1
fi

exit 0
