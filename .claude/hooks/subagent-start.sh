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

# Helper: Check if agent_type contains any keywords
contains_keyword() {
    local text="$1"
    shift
    local keywords=("$@")
    local text_lower=$(echo "$text" | tr '[:upper:]' '[:lower:]')

    for keyword in "${keywords[@]}"; do
        if echo "$text_lower" | grep -q "$keyword"; then
            echo "$keyword"
            return 0
        fi
    done
    return 1
}

# Helper: Count active agents
count_active_agents() {
    if [ ! -d ".wt/metadata" ]; then
        echo 0
        return
    fi
    ls -1 .wt/metadata/*.json 2>/dev/null | wc -l | tr -d ' '
}

# Decision logic: Determine if this agent needs isolation
CREATE_WORKTREE=false

# Priority 1: Check for inline keywords (override isolation)
INLINE_KEYWORDS=("read" "analyze" "explain" "document" "describe" "query" "inspect")
if MATCHED=$(contains_keyword "$AGENT_TYPE" "${INLINE_KEYWORDS[@]}"); then
    echo "Inline execution: matched keyword '$MATCHED'" >&2
    exit 0
fi

# Priority 2: Check agent type
case "$AGENT_TYPE" in
  Plan|Explore|Test|Execute|Bash)
    CREATE_WORKTREE=true
    echo "Isolation: agent type '$AGENT_TYPE'" >&2
    ;;
esac

# Priority 3: Check for isolation keywords
if [ "$CREATE_WORKTREE" != "true" ]; then
    ISOLATION_KEYWORDS=("refactor" "migrate" "scaffold" "restructure" "rebuild" "rewrite")
    if MATCHED=$(contains_keyword "$AGENT_TYPE" "${ISOLATION_KEYWORDS[@]}"); then
        CREATE_WORKTREE=true
        echo "Isolation: matched keyword '$MATCHED'" >&2
    fi
fi

# Priority 4: Check concurrency
if [ "$CREATE_WORKTREE" != "true" ]; then
    ACTIVE_AGENTS=$(count_active_agents)
    if [ "$ACTIVE_AGENTS" -ge 2 ]; then
        CREATE_WORKTREE=true
        echo "Isolation: concurrency (${ACTIVE_AGENTS} active agents)" >&2
    fi
fi

# Exit if no isolation needed
if [ "$CREATE_WORKTREE" != "true" ]; then
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
