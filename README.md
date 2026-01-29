# wt - Parallel Agent Coordination Tool

**Vision: Scale AI-assisted development linearly with agent count.**

A CLI tool that manages bare git repositories and enables multiple agents to work in parallel worktrees with automatic conflict detection and workspace isolation.

## Claude Code Integration

`wt` integrates seamlessly with [Claude Code](https://claude.ai/code) through automated hooks, enabling true parallel agent execution without manual workspace management.

### Quick Start

```bash
# 1. Install wt (see Installation below)
sudo make install

# 2. Clone or initialize your workspace
wt clone https://github.com/user/my-repo
# OR
wt init my-project

# 3. Install Claude Code hooks
cd ./my-repo  # or ./my-project
wt hooks install

# 4. Start Claude - hooks handle everything automatically
claude
```

The hooks automatically:

- Create isolated worktrees for complex agents (Plan, Explore, Test, Execute, Bash)
- Use intelligent keyword detection to determine when isolation is needed
- Isolate agents when multiple are running concurrently to prevent conflicts
- Detect and prevent merge conflicts before they occur
- Clean up stale worktrees safely
- Enable multiple agents to work simultaneously without interference

### Hook Management Commands

```bash
# Install hooks to current project
wt hooks install

# Install to specific directory
wt hooks install /path/to/project

# Overwrite existing hooks
wt hooks install --force

# View configuration
wt hooks config

# List available scripts
wt hooks list
```

📖 See [`.claude/hooks/README.md`](.claude/hooks/README.md) for detailed hook documentation and workflow examples.

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
# Clone from remote repository
wt clone https://github.com/user/repo
wt clone git@github.com:user/repo.git

# Clone with custom directory name
wt clone https://github.com/user/repo my-custom-name

# Initialize a new local project
wt init my-project
```

## Directory Structure

```text
~/my-project/
├── .bare/                      # Shared git objects (bare repository)
├── .wt/                        # Application state directory
│   └── metadata/               # Agent metadata JSON files
├── .git                        # Pointer file to .bare
└── main/                       # Primary worktree
```

## Agent Management Commands

wt provides built-in commands for managing agent worktrees with automatic metadata tracking and conflict detection.

### Creating Agent Worktrees

```bash
cd ~/wt/my-project

# Create agent worktrees
wt add alice
wt add bob main  # Optionally specify base branch

# Branch naming convention: <agent-name>
# Creates branches: alice and bob
```

Each agent worktree includes:

- Independent working directory at `~/wt/my-project/<agent-name>/`
- Dedicated branch named `<agent-name>`
- JSON metadata tracked in `.wt/metadata/<agent>.json`
- Automatic timestamp tracking for age-based cleanup

### Listing Agent Worktrees

```bash
# View all active agents
wt list

# Example output:
# AGENT                BRANCH                         AGE        STATUS
# alice                alice                          5m         active
# bob                  bob                            2h         active
```

### Switching Between Agent Worktrees

```bash
# Get path to switch to an agent worktree
wt switch alice

# Use in shell command substitution
cd $(wt switch alice --path)

# Or just get the path
wt switch alice --path
```

The switch command helps you quickly navigate between agent worktrees without typing full paths.

### Checking Agent Status

```bash
# Check merge conflicts and divergence
wt check alice

# Output includes:
# - Uncommitted change detection
# - Commits ahead/behind base branch
# - Conflict detection (non-destructive)
# - List of conflicting files
```

### Syncing with Base Branch

```bash
# Sync with base branch
wt sync alice
```

The sync command:

- Detects if branch is behind base
- Merges base branch into agent branch
- Shows conflict status if any arise

### Removing Agent Worktrees

```bash
# Remove worktree
wt remove alice
```

### Pruning Stale Worktrees

```bash
# Find and remove worktrees older than 7 days
wt prune

# Prune with custom age threshold
wt prune --older-than=14d

# Interactive confirmation for each removal
```

### Dashboard View

```bash
# Show comprehensive status
wt status

# Displays:
# - Total worktree count
# - Agent list with details
# - Age and status information
```

## Manual Worktree Management (Advanced)

You can still use git commands directly if needed:

```bash
cd ~/wt/my-project/main

# Manual worktree creation
git worktree add ../agent-charlie -b feature-api

# Manual cleanup
git worktree remove ../agent-charlie
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

## Documentation

- **[parallel-workflows.md](docs/parallel-workflows.md)** - Guide to using wt with parallel agent workflows
- **[CLAUDE.md](CLAUDE.md)** - Development guide for contributors
