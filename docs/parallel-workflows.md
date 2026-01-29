# Parallel Agent Workflows with wt

This guide explains how to use wt for managing multiple Claude Code agents working in parallel on the same codebase.

## Overview

wt enables multiple agents to work simultaneously by providing:

- **Isolated worktrees**: Each agent has its own working directory
- **Automatic conflict detection**: Check for merge issues before they happen
- **Metadata tracking**: Monitor agent activity and lifecycle
- **Coordination tools**: Sync, prune, and manage agent worktrees

## Basic Workflow

### 1. Initialize Project

```bash
# Clone an existing repository
wt clone https://github.com/user/repo my-project

# Or initialize a new local project
wt init my-project

cd ~/wt/my-project/main
```

### 2. Create Agent Worktrees

```bash
# Create agents for different tasks
wt add alice  # Authentication feature
wt add bob    # UI improvements
wt add carol  # API endpoints

# Verify creation
wt list
```

Output:
```
Agent Worktrees:

AGENT                BRANCH                         AGE        STATUS
alice                alice                          1m         active
bob                  bob                            1m         active
carol                carol                          1m         active

Total: 3 agent(s)
```

### 3. Agents Work Independently

Each agent operates in isolation:

```bash
# Alice works on authentication
cd ~/wt/my-project/alice
# ... make changes ...
git add .
git commit -m "Add JWT authentication"

# Bob works on UI concurrently
cd ~/wt/my-project/bob
# ... make changes ...
git add .
git commit -m "Update login component"

# Carol works on API
cd ~/wt/my-project/carol
# ... make changes ...
git add .
git commit -m "Add user endpoints"
```

### 4. Check Status Before Merging

```bash
# Check each agent for conflicts and divergence
wt check alice
wt check bob
wt check carol
```

Example output for Alice:
```
Checking merge: alice -> main

Divergence: 3 commits ahead, 0 commits behind

✓ Clean merge - no conflicts detected
```

Example with conflicts:
```
Checking merge: bob -> main

⚠️  Warning: Uncommitted changes detected

Divergence: 2 commits ahead, 1 commits behind
⚠️  Branch is behind main. Consider syncing.

✗ Conflicts detected

Conflicting files:
  - src/components/Login.tsx
  - src/styles/login.css
```

### 5. Sync with Base Branch

If an agent is behind the base branch:

```bash
# Sync with base branch
wt sync alice
```

The sync command:
- Fetches latest changes from base branch
- Merges base branch into agent branch
- Shows conflict status if any arise

### 6. Merge to Main

From the main worktree:

```bash
cd ~/wt/my-project/main

# Merge completed work
git merge alice --no-ff
git merge bob --no-ff
git merge carol --no-ff

# Push to remote
git push origin main
```

### 7. Clean Up Finished Work

```bash
# Remove agent worktrees
wt remove alice
wt remove bob
wt remove carol

# Verify cleanup
wt list
```

## Advanced Workflows

### Handling Conflicts

When conflicts are detected:

1. **Review the conflicts**:
   ```bash
   wt check alice
   # Shows: Conflicting files: src/auth/jwt.ts
   ```

2. **Sync with base branch**:
   ```bash
   cd ~/wt/my-project/alice
   wt sync alice
   # Merges main into alice branch
   # ... resolve conflicts if any ...
   git add .
   git commit
   ```

3. **Verify resolution**:
   ```bash
   wt check alice
   # Should now show: ✓ Clean merge
   ```

### Long-Running Tasks

For agents working on extended tasks:

```bash
# Regularly sync with base
wt sync alice

# Monitor age
wt list
# AGE column shows: 5m, 2h, 3d, etc.

# Update activity timestamp
wt check alice  # Automatically updates last_activity
```

### Stale Worktree Cleanup

Automatically find and remove abandoned worktrees:

```bash
# Remove stale worktrees (older than 7 days)
wt prune

# Use custom threshold
wt prune --older-than=14d

# Interactive confirmation for each removal
```

### Dashboard Monitoring

Get a comprehensive view:

```bash
wt status
```

Output:
```
=== Worktree Status Dashboard ===

Total worktrees: 5

Agent Worktrees:

AGENT                BRANCH                         AGE        STATUS
alice                alice                          2h         active
bob                  bob                            1d         active
carol                carol                          5m         active
dave                 dave                           8d         active
eve                  eve                            3h         active

Total: 5 agent(s)
```

## Best Practices

### 1. Use Descriptive Agent Names

```bash
# Good: Descriptive names that indicate purpose
wt add auth-feature
wt add ui-improvements
wt add api-endpoints
```

### 2. Regular Conflict Checks

```bash
# Before committing major changes
wt check alice

# Before ending work session
wt check alice
```

### 3. Sync Before Starting New Work

```bash
# Start of work session
cd ~/wt/my-project/alice
wt sync alice
```

### 4. Clean Up Promptly

```bash
# After merging to main
wt remove alice
```

### 5. Monitor Dashboard Regularly

```bash
# Weekly review
wt status
wt prune --older-than=7d
```

## Metadata Structure

Each agent worktree has metadata stored in `.wt/metadata/<agent>.json`:

```json
{
  "agent": "alice",
  "branch": "alice",
  "base_branch": "main",
  "created": "2025-01-25T10:30:00Z",
  "last_activity": "2025-01-25T14:45:00Z",
  "status": "active"
}
```

Fields:
- **agent**: Agent name (worktree directory name)
- **branch**: Branch name (same as agent name)
- **base_branch**: Base branch (typically "main")
- **created**: ISO 8601 timestamp of creation
- **last_activity**: ISO 8601 timestamp of last check/sync
- **status**: Current status ("active")

Metadata is automatically updated by:
- `wt add` - Creates initial metadata
- `wt check` - Updates last_activity
- `wt sync` - Updates last_activity
- `wt remove` - Deletes metadata

## Troubleshooting

### Agent Commands Not Found

```bash
# Ensure you're in a wt-managed directory
cd ~/wt/my-project

# Commands work from any subdirectory
cd ~/wt/my-project/main
wt agent list  # Works!
```

### Uncommitted Changes Blocking Sync

```bash
# Commit or stash changes first
cd ~/wt/my-project/alice
git stash
wt sync alice
git stash pop
```

### Branch Already Exists

```bash
# Use different agent name
wt add alice-v2

# Or delete old branch first
git branch -D alice
wt add alice
```

### Worktree Directory Exists

```bash
# Remove existing directory
rm -rf ~/wt/my-project/alice
wt add alice
```

## Integration with Claude Code Skills

The `/execute` skill can leverage wt for parallel agent workflows:

```bash
# From Claude Code CLI
/execute
# Agent creates: wt add alice
# Agent works in: ~/wt/project/alice/
# Agent checks: wt check alice
# Agent merges from: ~/wt/project/main/
```

## Example: Multi-Agent Feature Development

Complete example of parallel development:

```bash
# Setup
wt clone https://github.com/company/app my-app
cd ~/wt/my-app/main

# Create agent worktrees
wt add alice      # Authentication
wt add bob        # API layer
wt add carol      # UI components

# Agents work in parallel
# ... time passes, commits made ...

# Check progress
wt status

# Alice finishes first
wt check alice  # Clean merge
git merge alice --no-ff
wt remove alice

# Bob needs to sync
wt sync bob
wt check bob    # Now clean
git merge bob --no-ff
wt remove bob

# Carol has conflicts
wt check carol  # Shows conflicts
cd ../carol
wt sync carol
# ... resolve conflicts ...
git add .
git commit
cd ../main
wt check carol  # Now clean
git merge carol --no-ff
wt remove carol

# Push final result
git push origin main
```

## Summary

The wt commands provide a complete workflow for parallel agent development:

- **Setup**: `wt clone <url> [target-dir]`, `wt init <target-dir>`
- **Create**: `wt add <name> [base-branch]`
- **Monitor**: `wt list`, `wt status`
- **Validate**: `wt check <name>`
- **Sync**: `wt sync <name>`
- **Cleanup**: `wt remove <name>`
- **Maintenance**: `wt prune [--older-than=Nd]`

These commands automate the manual git worktree operations while adding safety checks, metadata tracking, and coordination features.
