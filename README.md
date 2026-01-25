# wt - Git Worktree Setup for Parallel Agents

A CLI tool that creates bare git repository structures optimized for multiple Claude Code agents working in parallel worktrees.

## Installation

Add `bin/wt` to your PATH or run directly:

```bash
./bin/wt <name>
```

## Usage

```bash
# Initialize a new local project
wt my-project

# Clone from remote repository
wt https://github.com/user/repo
wt git@github.com:user/repo.git

# Clone with custom directory name
wt https://github.com/user/repo my-custom-name
```

## Directory Structure

```
~/wt/<project>/
├── .bare/    # Shared git objects (bare repository)
├── .git      # Pointer file to .bare
└── main/     # Primary worktree
```

## Agent Management Commands

wt provides built-in commands for managing agent worktrees with automatic metadata tracking and conflict detection.

### Creating Agent Worktrees

```bash
cd ~/wt/my-project

# Create agent worktrees with task IDs
wt agent create alice 1234 main
wt agent create bob 5678 main

# Branch naming convention: task/<task-id>/<agent-name>
# Creates: task/1234/alice and task/5678/bob
```

Each agent worktree includes:
- Independent working directory at `~/wt/my-project/<agent-name>/`
- Dedicated branch following `task/<task-id>/<agent-name>` convention
- JSON metadata tracked in `.bare/worktree-metadata/<agent>.json`
- Automatic timestamp tracking for age-based cleanup

### Listing Agent Worktrees

```bash
# View all active agents
wt agent list

# Example output:
# AGENT                TASK       BRANCH                         AGE        STATUS
# alice                1234       task/1234/alice                5m         active
# bob                  5678       task/5678/bob                  2h         active
```

### Checking Agent Status

```bash
# Check merge conflicts and divergence
wt agent check alice

# Output includes:
# - Uncommitted change detection
# - Commits ahead/behind base branch
# - Conflict detection (non-destructive)
# - List of conflicting files
```

### Syncing with Base Branch

```bash
# Check sync status
wt agent sync alice

# Auto-rebase onto base branch
wt agent sync alice --auto-rebase
```

The sync command:
- Detects if branch is behind base
- Requires clean working directory
- Optionally rebases with `--auto-rebase`
- Aborts rebase on conflicts

### Removing Agent Worktrees

```bash
# Remove worktree (keeps branch)
wt agent remove alice

# Remove worktree and delete merged branch
wt agent remove alice --delete-branch
```

### Pruning Stale Worktrees

```bash
# Find worktrees older than 7 days (dry run)
wt agent prune --dry-run

# Prune with custom age threshold
wt agent prune --older-than=14d

# Interactive confirmation for each removal
```

### Dashboard View

```bash
# Show comprehensive status
wt agent status

# Displays:
# - Total worktree count
# - Active worktree count
# - Agent list with details
# - Git worktree list
```

## Manual Worktree Management (Advanced)

You can still use git commands directly if needed:

```bash
cd ~/wt/my-project

# Manual worktree creation
git worktree add agent-charlie -b feature-api main

# Manual cleanup
git worktree remove agent-charlie
```

## Benefits

- **Parallel work**: Multiple agents work simultaneously without conflicts
- **Shared history**: All worktrees share the same git objects
- **Fast branching**: Creating worktrees is instantaneous
- **Independent state**: Each worktree has its own HEAD, index, and working directory
- **Easy cleanup**: Worktrees can be removed without losing commits

## Running Tests

```bash
./tests/test-runner.sh
```

## Project Structure

```
wt/
├── bin/wt                  # Main CLI executable
├── lib/
│   ├── parse.sh            # Argument parsing
│   ├── repo.sh             # Repository operations
│   ├── agent.sh            # Agent worktree management
│   ├── metadata.sh         # JSON metadata tracking
│   └── conflict.sh         # Conflict detection and sync
└── tests/
    ├── test-runner.sh      # Test framework
    ├── test-integration.sh # End-to-end tests
    ├── test-local.sh       # Local setup tests
    ├── test-parse.sh       # Parsing tests
    ├── test-agent.sh       # Agent management tests
    ├── test-metadata.sh    # Metadata library tests
    └── test-conflict.sh    # Conflict detection tests
```
