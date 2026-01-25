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

## Working with Agents

Each agent gets its own worktree with an independent working directory:

```bash
cd ~/wt/my-project

# Create worktrees for agents
git worktree add agent-alice -b feature-auth main
git worktree add agent-bob -b feature-ui main

# Agents work independently
echo "auth code" > agent-alice/auth.py
git -C agent-alice add . && git -C agent-alice commit -m "Add auth"

echo "ui code" > agent-bob/ui.py
git -C agent-bob add . && git -C agent-bob commit -m "Add UI"

# Merge from main worktree
git -C main merge feature-auth
git -C main merge feature-ui

# Clean up finished agent worktrees
git worktree remove agent-alice
git worktree remove agent-bob
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
├── bin/wt              # Main CLI executable
├── lib/
│   ├── parse.sh        # Argument parsing
│   └── repo.sh         # Repository operations
└── tests/
    ├── test-runner.sh      # Test framework
    ├── test-integration.sh # End-to-end tests
    ├── test-local.sh       # Local setup tests
    └── test-parse.sh       # Parsing tests
```
