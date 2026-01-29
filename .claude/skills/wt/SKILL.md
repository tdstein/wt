---
name: wt
description: wt is optimized for multiple agents working in parallel worktrees.
context: fork
allowed-tools: Bash(wt *)
---

# wt - Worktree Management for Parallel Agents

`wt` is a Git worktree management tool that enables multiple Claude Code agents to work simultaneously on the same codebase without conflicts. Each agent gets an isolated workspace (worktree) with its own branch, allowing true parallel execution.

## Core Capabilities

### Repository Setup
- **Clone & initialize**: Set up bare repository structure with primary worktree
- **Auto-detection**: Recognizes URLs vs local paths automatically
- **Base branch detection**: Identifies main/master branch automatically

### Agent Workspace Management
- **Create isolated worktrees**: Each agent gets a dedicated workspace
- **Automatic branch creation**: Follows `<agent-name>` naming convention
- **Metadata tracking**: Monitors agent activity, state, and progress
- **Health monitoring**: Tracks PIDs and last activity timestamps

### Conflict Prevention
- **Pre-merge detection**: Check for conflicts before they occur
- **Divergence tracking**: Monitor commits ahead/behind base branch
- **Non-destructive checks**: Analyze without modifying working directory

### Synchronization & Cleanup
- **Sync with base**: Keep agent worktrees up-to-date
- **Automatic pruning**: Remove stale worktrees based on age
- **Dashboard view**: Monitor all agents at a glance

## When to Use This Skill

### Automatic Isolation
The SubagentStart hook automatically creates isolated worktrees based on:

**Agent Type** (Plan, Explore, Test, Execute, Bash)
- These agent types always get isolated worktrees
- Isolation ensures complex operations don't interfere with main work

**Isolation Keywords** (in agent_type field)
- Keywords: refactor, migrate, scaffold, restructure, rebuild, rewrite
- Triggers worktree creation for large-scale modifications

**Inline Keywords** (override isolation)
- Keywords: read, analyze, explain, document, describe, query, inspect
- Forces agent to work in current directory for read-only tasks

**Concurrency Threshold**
- When 2+ agents are already active, new agents get isolated
- Prevents conflicts during parallel execution

### Manual Worktree Creation
Use `wt add <name>` explicitly when:
1. You need a dedicated workspace outside automatic rules
2. Working on long-horizon feature development
3. Creating multiple parallel workspaces for coordination
4. Testing changes that require full isolation

### Dynamic Base Branch
- Worktrees automatically use your **current branch** as the base
- If you're on `feature-x`, new worktrees branch from `feature-x` (not `main`)
- This enables natural nested workflows and branch-relative development

## Repository Structure

`wt` creates this layout:
```
~/wt/<project>/
├── .bare/                      # Bare git repository (shared)
│   ├── objects/               # Git objects (shared efficiently)
│   └── worktrees/             # Git worktree metadata
├── .wt/                       # Application state
│   └── metadata/              # Agent metadata JSON files
├── .git                       # Pointer to .bare
└── main/                      # Primary worktree
```

After creating agent worktrees:
```
~/wt/<project>/
├── .bare/
├── .wt/
│   └── metadata/
│       ├── alice.json         # Alice's metadata
│       └── bob.json           # Bob's metadata
├── main/                      # Primary worktree (base branch)
├── alice/                     # Alice's worktree (branch: alice)
└── bob/                       # Bob's worktree (branch: bob)
```

## Common Workflows

### Initial Setup (Clone Repository)
```bash
# Clone and set up bare repo structure
wt clone https://github.com/user/repo

# Creates: ~/wt/repo/.bare/, ~/wt/repo/.wt/, and ~/wt/repo/main/
cd ~/wt/repo/main
```

### Initial Setup (Local Repository)
```bash
# Initialize local project
wt init my-project

# Creates bare repo structure
cd my-project/main
```

### Create Agent Worktree
```bash
# Navigate to repository (any worktree)
cd ~/wt/repo/main

# Create agent workspace (uses current branch as base)
wt add alice

# Agent now has:
# - Directory: ~/wt/repo/alice/
# - Branch: alice
# - Base: main (automatically detected from current branch)
# - Metadata: ~/wt/repo/.wt/metadata/alice.json

# Work in agent's space
cd ~/wt/repo/alice
# ... do work ...
```

### Create Worktree from Feature Branch
```bash
# When working on a feature branch
cd ~/wt/repo/main
git checkout feature-x

# Create worktree - automatically uses feature-x as base
wt add sub-task

# Result:
# - Base branch: feature-x (not main!)
# - New work branches from your current context
```

### Explicit Base Branch
```bash
# Override automatic detection with explicit base
wt add alice develop

# Result:
# - Base branch: develop (explicitly specified)
# - Ignores current branch
```

### Check for Conflicts (Before Merging)
```bash
cd ~/wt/repo/alice

# Check if agent's work conflicts with base branch
wt check alice

# Output shows:
# - Commits ahead/behind base
# - List of conflicting files (if any)
# - Conflict markers and details
```

### Sync with Base Branch
```bash
cd ~/wt/repo/alice

# Pull latest changes from base branch
wt sync alice

# Merges base branch into agent's branch
# Handles conflicts if they arise
```

### Monitor All Agents
```bash
cd ~/wt/repo/main

# Dashboard view of all agents
wt status

# Shows:
# - Agent names and branches
# - Last activity timestamps
# - Divergence from base (ahead/behind)
# - Worktree paths
```

### Clean Up Agent Worktree
```bash
cd ~/wt/repo/main

# Remove specific agent
wt remove alice

# Or prune stale agents (default: >7 days old)
wt prune

# With custom age threshold
wt prune --older-than 3d
```

### List All Agents
```bash
cd ~/wt/repo/main

wt list

# Shows all active agent worktrees with:
# - Agent names
# - Branch names
# - Worktree paths
# - Last activity
```

### Switch Between Agents
```bash
cd ~/wt/repo/main

# Get path to switch to agent worktree
wt switch alice

# Use in shell command substitution
cd $(wt switch alice --path)

# Or just get the path for scripting
wt switch alice --path
```

## Integration Patterns

### Pattern 1: Quick Tasks (Default)
```bash
# Most tasks work in current directory without creating worktrees

# 1. Check if in a wt workspace
if [ ! -d ".bare" ]; then
  wt clone <repo-url> || wt init .
  cd main
fi

# 2. Do work directly in current worktree
# ... make changes ...

# 3. Commit and report completion
# No worktree creation needed for simple tasks
```

### Pattern 2: Complex Tasks (Explicit Isolation)
```bash
# For complex/long-horizon work, create isolated worktrees explicitly

# 1. Create dedicated worktree
AGENT_NAME="task-$(date +%s)"
wt add "$AGENT_NAME"
cd "../$AGENT_NAME"

# 2. Do complex work in isolation
# ... multi-file changes ...

# 3. Check for conflicts
wt check "$AGENT_NAME"

# 4. Merge when ready
cd ../main
git merge "$AGENT_NAME" --no-ff
wt remove "$AGENT_NAME"
```

### Pattern 5: Parallel Execution
```bash
# Coordinator creates multiple agent worktrees:
wt add alice
wt add bob
wt add charlie

# Each agent works independently:
# alice: ~/wt/repo/alice/
# bob: ~/wt/repo/bob/
# charlie: ~/wt/repo/charlie/

# No file conflicts, true parallelism
```

### Pattern 3: Autonomous Coordination
```bash
# Agents get automatic worktrees based on dynamic decision logic
# Example: User spawns Plan agent or uses isolation keywords

# 1. SubagentStart hook automatically:
#    - Evaluates agent_type against decision criteria
#    - Priority 1: Inline keywords → work in current directory
#    - Priority 2: Agent type (Plan/Explore/Test/Execute/Bash) → isolate
#    - Priority 3: Isolation keywords (refactor/migrate/etc.) → isolate
#    - Priority 4: Concurrency (2+ active agents) → isolate
#    - Creates worktree with current branch as base (if isolation triggered)

# 2. Agent does work in isolated worktree (or current directory)

# 3. SubagentStop hook automatically:
#    - Checks for conflicts
#    - If clean: merges back to current branch
#    - Removes worktree
#    - All without user intervention

# User sees: "✓ Auto-merged plan-<id> → main and cleaned up worktree"

# Examples of automatic decisions:
# - "Plan" agent → Isolated (agent type)
# - "RefactorAuth" → Isolated (keyword: refactor)
# - "ReadDocs" → Inline (keyword: read)
# - "ImplementFeature" with 2+ agents active → Isolated (concurrency)
```

### Pattern 4: Manual Merging (When Conflicts Exist)
```bash
# If conflicts detected, auto-merge is skipped:
wt check alice  # Shows conflict details
wt sync alice   # Get latest changes

# Resolve conflicts manually
cd alice/
# ... fix conflicts ...
git add .
git commit -m "Resolve conflicts"

# Merge when ready
cd ../main
git merge alice --no-ff

# Clean up
wt remove alice
```

## Best Practices

### Naming Conventions
- **Agent worktrees**: Use descriptive names (e.g., `research-auth`, `implement-api`)
- **Branches**: Auto-created as `<agent-name>` (enforced by tool)
- **Keep it simple**: Short, lowercase, hyphen-separated names

### Workflow Guidelines
1. **Trust automatic isolation**: SubagentStart hook intelligently decides based on agent type, keywords, and concurrency
2. **Use descriptive agent names**: Keywords in agent_type trigger isolation (e.g., "RefactorAuth", "MigrateDB")
3. **Base branch is dynamic**: New worktrees branch from your current context, not always `main`
4. **Explicit control when needed**: Use `wt add` for manual worktree creation outside automatic rules
5. **Check before manual merge**: Run `wt check` before merging work manually
6. **Clean up**: Remove worktrees when done (`wt remove` or `wt prune`)

### Performance Optimization
- **Shared object storage**: `.bare/objects/` is shared efficiently
- **Minimal disk usage**: Worktrees only store working files
- **Fast operations**: Worktree creation is near-instantaneous
- **Parallel I/O**: Each worktree has independent file locks

### Safety Measures
- **Non-destructive checks**: `wt check` never modifies files
- **Metadata tracking**: Activity timestamps prevent accidental deletion
- **Graceful cleanup**: `wt prune` respects age thresholds
- **Context awareness**: Commands work from any subdirectory

## Command Reference

### Setup Commands
```bash
wt clone <repo-url> [target-dir]    # Clone repository
wt init <target-dir>                # Initialize local project
```

### Agent Management
```bash
wt add <name> [base-branch]         # Create agent worktree
wt remove <name>                    # Remove agent worktree
wt list                             # List all agents
wt status                           # Dashboard view
wt switch <name>                    # Switch to agent worktree
```

### Conflict & Sync
```bash
wt check <name>                     # Check for conflicts
wt sync <name>                      # Sync with base branch
```

### Cleanup
```bash
wt prune [--older-than 7d]          # Remove stale worktrees
```

## Troubleshooting

### "Not in a wt-managed repository"
Solution: Run `wt clone` or `wt init` first to set up the workspace.

### "Agent already exists"
Solution: Choose a different name or remove the existing agent with `wt remove`.

### Conflicts detected by `wt check`
Solution:
1. Review conflicting files
2. Run `wt sync` to get latest changes
3. Resolve conflicts manually in the worktree
4. Test and commit resolution

### Stale worktrees consuming space
Solution: Run `wt prune` to automatically clean up old worktrees.

### Working directory confusion
Solution: Use `pwd` to confirm you're in the correct worktree before operations.

## Examples

### Example 1: Research Task
```bash
# Start research
cd ~/wt/myproject/main
wt add research-api
cd ../research-api

# Investigate code
grep -r "API" .
# ... analysis work ...

# Report findings, clean up
cd ../main
wt remove research-api
```

### Example 2: Parallel Implementation
```bash
# Coordinator sets up parallel work
cd ~/wt/myproject/main
wt add implement-frontend
wt add implement-backend
wt add write-tests

# Three agents work simultaneously:
# - implement-frontend: UI changes
# - implement-backend: API changes
# - write-tests: Test coverage

# Each agent works in isolation, no conflicts
```

### Example 3: Safe Feature Development
```bash
# Create feature worktree
cd ~/wt/myproject/main
wt add feature-auth
cd ../feature-auth

# Implement feature
# ... make changes ...
git add .
git commit -m "Implement authentication"

# Check for conflicts before merge
wt check feature-auth

# Sync with latest main
wt sync feature-auth

# Merge to main
cd ../main
git merge feature-auth

# Clean up
wt remove feature-auth
```

## Additional Resources

For more detailed information:
- **Architecture**: See `/Users/me/wt/wt/main/CLAUDE.md`
- **Engineering guidelines**: See `/Users/me/wt/.claude/CLAUDE.md`
- **Command help**: Run `wt -h` or `wt <command> -h`

## Help Output

!`wt -h`
