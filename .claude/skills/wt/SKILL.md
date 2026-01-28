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

### MANDATORY Usage
You MUST use `wt` when:
1. Working with any git repository (setup workspace first)
2. Multiple agents need to work simultaneously (parallel workflows)
3. Each distinct task requires isolation (prevent conflicts)

### Skill Integration
Other skills MUST use `wt` as their foundation when working with repositories. The workflow is:
1. Skill invoked → First action: set up `wt` workspace
2. Do work in isolated worktree
3. Report back to caller

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

# Creates: ~/wt/repo/.bare/ and ~/wt/repo/main/
cd ~/wt/repo/main
```

### Initial Setup (Local Repository)
```bash
# Initialize existing local project
wt init /path/to/my-project

# Creates bare repo structure in place
cd /path/to/my-project/main
```

### Create Agent Worktree
```bash
# Navigate to repository (any worktree)
cd ~/wt/repo/main

# Create agent workspace
wt add alice

# Agent now has:
# - Directory: ~/wt/repo/alice/
# - Branch: alice
# - Metadata: ~/wt/repo/.wt/metadata/alice.json

# Work in agent's space
cd ~/wt/repo/alice
# ... do work ...
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

## Integration Patterns

### Pattern 1: Skill Initialization
```bash
# Every repository-based skill should start with:

# 1. Check if already in a wt workspace
if [ ! -d ".bare" ]; then
  # 2. Set up workspace (clone or init)
  wt clone <repo-url> || wt init .
  cd main
fi

# 3. Create agent worktree
AGENT_NAME="agent-$(date +%s)"
wt add "$AGENT_NAME"
cd "../$AGENT_NAME"

# 4. Do work in isolated space
# ...

# 5. Report completion
```

### Pattern 2: Parallel Execution
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

### Pattern 3: Safe Merging
```bash
# Before merging agent's work:
wt check alice  # Detect conflicts
wt sync alice   # Get latest changes

# If clean, merge agent's branch to main
cd main
git merge alice

# Clean up
wt remove alice
```

## Best Practices

### Naming Conventions
- **Agent worktrees**: Use descriptive names (e.g., `research-auth`, `implement-api`)
- **Branches**: Auto-created as `<agent-name>` (enforced by tool)
- **Keep it simple**: Short, lowercase, hyphen-separated names

### Workflow Guidelines
1. **Always start with setup**: Run `wt clone` or `wt init` before any work
2. **One agent, one worktree**: Don't share worktrees between agents
3. **Check before merge**: Always run `wt check` before merging work
4. **Sync regularly**: Use `wt sync` to stay current with base branch
5. **Clean up**: Remove worktrees when done (`wt remove` or `wt prune`)

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
