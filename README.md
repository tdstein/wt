# wt - Git Worktree Setup for Parallel Agents

A CLI tool that creates bare git repository structures optimized for multiple Claude Code agents working in parallel worktrees.

## Installation

### System-wide Installation (Recommended)

Install wt to your system using make:

```bash
# Install to /usr/local (requires sudo)
sudo make install

# Install to custom location
make install PREFIX=/opt/local

# User installation (no sudo required)
make install PREFIX=~/.local
```

To uninstall:

```bash
sudo make uninstall
```

### Development Mode

Run directly from the repository without installation:

```bash
./bin/wt <name>
```

Or add `bin/wt` to your PATH for the current session:

```bash
export PATH="$PATH:$(pwd)/bin"
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
# Run all tests
make test

# Or use go test directly
go test ./...

# Run with coverage
make test-coverage
```

## Project Structure

```
wt/
├── bin/wt                  # Compiled Go binary
├── cmd/wt/                 # CLI entry point
│   ├── main.go             # Command definitions
│   ├── utils.go            # Logging and helpers
│   └── main_test.go        # CLI integration tests
├── internal/               # Internal packages
│   ├── parse/              # URL and argument parsing
│   ├── git/                # Git command wrapper
│   ├── repo/               # Repository operations
│   ├── agent/              # Agent management
│   │   ├── agent.go        # Agent operations
│   │   ├── metadata.go     # Metadata management
│   │   ├── agent_test.go   # Agent tests
│   │   └── metadata_test.go # Metadata tests
│   └── conflict/           # Conflict detection
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
└── Makefile                # Build and install targets
```
