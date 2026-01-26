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
wt https://github.com/user/repo my-project

# Or initialize a new local project
wt my-project

cd ~/wt/my-project
```

### 2. Create Agent Worktrees

```bash
# Create agents for different tasks
wt agent create alice 1234 main  # Task 1234: Authentication feature
wt agent create bob 5678 main    # Task 5678: UI improvements
wt agent create carol 9012 main  # Task 9012: API endpoints

# Verify creation
wt agent list
```

Output:
```
Agent Worktrees:

AGENT                TASK       BRANCH                         AGE        STATUS
alice                1234       task/1234/alice                1m         active
bob                  5678       task/5678/bob                  1m         active
carol                9012       task/9012/carol                1m         active

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
wt agent check alice
wt agent check bob
wt agent check carol
```

Example output for Alice:
```
Checking merge: task/1234/alice -> main

Divergence: 3 commits ahead, 0 commits behind

✓ Clean merge - no conflicts detected
```

Example with conflicts:
```
Checking merge: task/5678/bob -> main

⚠️  Warning: Uncommitted changes detected

Divergence: 2 commits ahead, 1 commits behind
⚠️  Branch is behind main. Consider rebasing.

✗ Conflicts detected

Conflicting files:
  - src/components/Login.tsx
  - src/styles/login.css
```

### 5. Sync with Base Branch

If an agent is behind the base branch:

```bash
# Check sync status
wt agent sync alice

# Auto-rebase if there are no conflicts
wt agent sync alice --auto-rebase
```

The sync command:
- Fetches latest changes from base branch
- Shows how many commits behind
- Requires clean working directory
- Optionally rebases with `--auto-rebase`

### 6. Merge to Main

From the main worktree:

```bash
cd ~/wt/my-project/main

# Merge completed work
git merge task/1234/alice --no-ff
git merge task/5678/bob --no-ff
git merge task/9012/carol --no-ff

# Push to remote
git push origin main
```

### 7. Clean Up Finished Work

```bash
# Remove agent worktrees
wt agent remove alice --delete-branch
wt agent remove bob --delete-branch
wt agent remove carol --delete-branch

# Verify cleanup
wt agent list
```

## Advanced Workflows

### Handling Conflicts

When conflicts are detected:

1. **Review the conflicts**:
   ```bash
   wt agent check alice
   # Shows: Conflicting files: src/auth/jwt.ts
   ```

2. **Resolve manually**:
   ```bash
   cd ~/wt/my-project/alice
   git rebase main
   # ... resolve conflicts ...
   git rebase --continue
   ```

3. **Verify resolution**:
   ```bash
   wt agent check alice
   # Should now show: ✓ Clean merge
   ```

### Long-Running Tasks

For agents working on extended tasks:

```bash
# Regularly sync with base
wt agent sync alice --auto-rebase

# Monitor age
wt agent list
# AGE column shows: 5m, 2h, 3d, etc.

# Update activity timestamp
wt agent check alice  # Automatically updates last_activity
```

### Stale Worktree Cleanup

Automatically find and remove abandoned worktrees:

```bash
# Preview stale worktrees (older than 7 days)
wt agent prune --dry-run

# Use custom threshold
wt agent prune --older-than=14d --dry-run

# Remove stale worktrees interactively
wt agent prune --older-than=7d
```

### Dashboard Monitoring

Get a comprehensive view:

```bash
wt agent status
```

Output:
```
=== Worktree Status Dashboard ===

Total worktrees: 5
Active worktrees: 5

Agent Worktrees:

AGENT                TASK       BRANCH                         AGE        STATUS
alice                1234       task/1234/alice                2h         active
bob                  5678       task/5678/bob                  1d         active
carol                9012       task/9012/carol                5m         active
dave                 3456       task/3456/dave                 8d         active
eve                  7890       task/7890/eve                  3h         active

Total: 5 agent(s)

Git worktree list:
/Users/me/wt/my-project/.bare (bare)
/Users/me/wt/my-project/main  abcd1234 [main]
/Users/me/wt/my-project/alice ef567890 [task/1234/alice]
/Users/me/wt/my-project/bob   12345678 [task/5678/bob]
...
```

## Best Practices

### 1. Use Descriptive Task IDs

```bash
# Good: Reference issue/ticket numbers
wt agent create alice 1234 main     # GitHub issue #1234
wt agent create bob GH-5678 main    # JIRA ticket GH-5678

# Descriptive names work too
wt agent create carol auth-feature main
```

### 2. Regular Conflict Checks

```bash
# Before committing major changes
wt agent check alice

# Before ending work session
wt agent check alice
```

### 3. Sync Before Starting New Work

```bash
# Start of work session
cd ~/wt/my-project/alice
wt agent sync alice --auto-rebase
```

### 4. Clean Up Promptly

```bash
# After merging to main
wt agent remove alice --delete-branch
```

### 5. Monitor Dashboard Regularly

```bash
# Weekly review
wt agent status
wt agent prune --older-than=7d --dry-run
```

## Metadata Structure

Each agent worktree has metadata stored in `.wt/metadata/<agent>.json`:

```json
{
  "agent": "alice",
  "task_id": "1234",
  "branch": "task/1234/alice",
  "base_branch": "main",
  "created": "2025-01-25T10:30:00Z",
  "last_activity": "2025-01-25T14:45:00Z",
  "status": "active"
}
```

Fields:
- **agent**: Agent name (worktree directory name)
- **task_id**: Associated task/issue identifier
- **branch**: Full branch name
- **base_branch**: Base branch (typically "main")
- **created**: ISO 8601 timestamp of creation
- **last_activity**: ISO 8601 timestamp of last check/sync
- **status**: Current status ("active")

Metadata is automatically updated by:
- `wt agent create` - Creates initial metadata
- `wt agent check` - Updates last_activity
- `wt agent sync` - Updates last_activity
- `wt agent remove` - Deletes metadata

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
wt agent sync alice --auto-rebase
git stash pop
```

### Branch Already Exists

```bash
# Use different agent name or task ID
wt agent create alice-v2 1234 main

# Or delete old branch first
git branch -D task/1234/alice
wt agent create alice 1234 main
```

### Worktree Directory Exists

```bash
# Remove existing directory
rm -rf ~/wt/my-project/alice
wt agent create alice 1234 main
```

## Integration with Claude Code Skills

The `/execute` skill can leverage wt for parallel agent workflows:

```bash
# From Claude Code CLI
/execute
# Agent creates: wt agent create alice <task-id> main
# Agent works in: ~/wt/project/alice/
# Agent checks: wt agent check alice
# Agent merges from: ~/wt/project/main/
```

## Example: Multi-Agent Feature Development

Complete example of parallel development:

```bash
# Setup
wt https://github.com/company/app my-app
cd ~/wt/my-app

# CEO assigns tasks
wt agent create alice auth main      # Authentication
wt agent create bob api main         # API layer
wt agent create carol ui main        # UI components

# Agents work in parallel
# ... time passes, commits made ...

# Check progress
wt agent status

# Alice finishes first
wt agent check alice  # Clean merge
cd main && git merge task/auth/alice --no-ff
wt agent remove alice --delete-branch

# Bob needs to sync
wt agent sync bob --auto-rebase
wt agent check bob    # Now clean
cd main && git merge task/api/bob --no-ff
wt agent remove bob --delete-branch

# Carol has conflicts
wt agent check carol  # Shows conflicts
cd carol
git rebase main
# ... resolve conflicts ...
git rebase --continue
wt agent check carol  # Now clean
cd ../main && git merge task/ui/carol --no-ff
wt agent remove carol --delete-branch

# Push final result
git push origin main
```

## Summary

The wt agent commands provide a complete workflow for parallel agent development:

- **Create**: `wt agent create <name> <task> <base>`
- **Monitor**: `wt agent list`, `wt agent status`
- **Validate**: `wt agent check <name>`
- **Sync**: `wt agent sync <name> [--auto-rebase]`
- **Cleanup**: `wt agent remove <name> [--delete-branch]`
- **Maintenance**: `wt agent prune [--older-than=Nd]`

These commands automate the manual git worktree operations while adding safety checks, metadata tracking, and coordination features.
