# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`wt` is a Git worktree management tool written in Go, designed to enable parallel agent workflows. It creates bare git repository structures with multiple worktrees, allowing multiple Claude Code agents to work simultaneously on the same codebase without conflicts.

## Core Concepts

### Repository Structure

The tool creates a specific directory layout:
```
~/wt/<project>/
├── .bare/                      # Bare git repository (shared objects)
│   ├── objects/               # Git object database
│   ├── refs/                  # Git references
│   └── worktrees/             # Git worktree tracking
├── .wt/                       # Application state directory
│   ├── metadata/              # JSON metadata for each agent
│   ├── queue/                 # Task management (pending/claimed/completed/failed)
│   └── locks/                 # Task locks for atomic claim/release
├── .git                       # Pointer file to .bare
└── main/                      # Primary worktree (base branch)
```

### Agent Worktrees

Agent worktrees follow a strict naming convention:
- **Directory**: `<agent-name>/` (e.g., `alice/`, `bob/`)
- **Branch**: `task/<task-id>/<agent-name>` (e.g., `task/1234/alice`)
- **Metadata**: `.wt/metadata/<agent>.json`

### Three Core Systems

1. **Agent Management** (`internal/agent/`): Worktree lifecycle, conflict detection, syncing
   - Metadata stored in: `.wt/metadata/`
2. **Task Queue** (`internal/queue/`): Task coordination with priority, dependencies, and state tracking
   - Queue state stored in: `.wt/queue/`
3. **Task Locking** (`internal/locking/`): Atomic task claiming with stale lock detection
   - Locks stored in: `.wt/locks/`

## Development Commands

### Building
```bash
# Build binary to bin/wt
make build

# Build with version info
make build VERSION=1.0.0

# Run tests
make test

# Run tests with coverage report
make test-coverage

# Format code
make fmt

# Run go vet
make vet
```

### Installation
```bash
# System-wide (requires sudo)
sudo make install

# User installation (no sudo)
make install PREFIX=~/.local

# Custom location
make install PREFIX=/opt/local

# Uninstall
sudo make uninstall
```

### Running Tests

Tests are organized by package:
```bash
# All tests
go test ./...

# Specific package
go test ./internal/agent
go test ./internal/queue
go test ./internal/locking

# Verbose output
go test -v ./...

# Single test
go test -v -run TestAgentCreate ./internal/agent
```

## Architecture

### Command Structure (cmd/wt/main.go)

The CLI is built with Cobra and organized hierarchically:
```
wt [repo-url|name] [name]           # Root: Setup bare repo
├── agent                            # Agent worktree management
│   ├── create <name> <task> [base] # Create agent worktree
│   ├── remove <name>                # Remove agent worktree
│   ├── list                         # List all agents
│   ├── check <name>                 # Conflict detection
│   ├── sync <name>                  # Sync with base branch
│   ├── prune                        # Remove stale agents
│   └── status                       # Dashboard view
├── queue                            # Task queue management
│   ├── add <task-id>                # Add task to queue
│   ├── list                         # List tasks by state
│   ├── get <task-id>                # Get task details
│   └── remove <task-id>             # Remove task
└── lock                             # Task locking
    ├── claim <task-id> <agent>      # Claim lock
    ├── release <task-id> <agent>    # Release lock
    ├── list                         # List active locks
    └── clean                        # Remove stale locks
```

### Package Organization

**Core packages** (internal/):
- `parse/`: URL and argument parsing (clone vs local detection)
- `git/`: Git command wrapper with error handling
- `repo/`: Repository setup (bare repo creation, worktree initialization)
- `conflict/`: Conflict detection via git merge --no-commit --no-ff
- `agent/`: Agent worktree operations and metadata management
- `queue/`: Task queue with state machine (pending → claimed → completed/failed)
- `locking/`: Atomic task locks with PID tracking and staleness detection

### Key Design Patterns

**Metadata-Driven Architecture**: Each agent worktree has JSON metadata tracking:
- Creation timestamp and last activity
- Task ID and branch associations
- Progress percentage and state (claimed, working, testing, blocked, failed, completed)
- PID for health monitoring
- Checkpoints for rollback capability

**State-Based Queue System**: Tasks move through directories:
```
.wt/queue/
├── pending/        # Available for claiming
├── claimed/        # Actively being worked on
├── completed/      # Successfully finished
└── failed/         # Failed or abandoned
```

**Lock-Based Coordination**: Prevents race conditions when multiple agents claim tasks:
- Atomic claim/release operations
- Stale lock detection (timeout-based)
- PID tracking for health checks
- Force release for cleanup

**Non-Destructive Conflict Detection**: The `agent check` command:
1. Uses `git merge-tree` or `git merge --no-commit --no-ff` (aborted after check)
2. Reports divergence (commits ahead/behind)
3. Lists conflicting files without modifying working directory
4. Updates last_activity timestamp in metadata

### Time-Based Operations

**Age Calculation** (internal/agent/):
- Based on metadata `last_activity` field
- Human-readable format: "5m", "2h", "3d"
- Used by `prune` command for cleanup (default: 7 days)

**Lock Staleness** (internal/locking/):
- Based on `last_active` timestamp
- Configurable timeout (default: 1 hour)
- Automatic cleanup via `lock clean` command

## Important Conventions

### Branch Naming
Always follow `task/<task-id>/<agent-name>` convention. This is enforced by `agent create` command.

### Base Branch Detection
The tool auto-detects the base branch (usually "main" or "master") during repo setup. Agent operations use this for conflict checking and syncing.

### Context Propagation
Agent commands auto-discover the worktree root by searching for `.bare/` directory upwards from current working directory. Commands work from any subdirectory.

### Metadata Updates
The following commands automatically update `last_activity`:
- `wt agent check <name>`
- `wt agent sync <name>`
- Manual: Touch metadata via `MetadataManager.UpdateActivity()`

## Testing Approach

Tests use:
- Temporary directories for isolation
- Real git operations (not mocked)
- Table-driven tests for multiple scenarios
- Cleanup via `t.Cleanup()` or `defer os.RemoveAll()`

Example pattern from `internal/agent/agent_test.go`:
```go
func TestAgentCreate(t *testing.T) {
    tmpDir := t.TempDir()
    // ... setup bare repo ...
    mgr := agent.NewAgentManager(tmpDir)
    err := mgr.Create(agent.CreateOptions{...})
    // ... assertions ...
}
```

## Common Workflows

### Development Cycle
1. Work in `main/` directory (primary worktree)
2. Build: `make build`
3. Test: `make test`
4. Format: `make fmt`
5. Install locally: `make install PREFIX=~/.local`

### Adding New Commands
1. Add command function in `cmd/wt/main.go` (e.g., `newFooCmd()`)
2. Register with parent command via `.AddCommand()`
3. Implement logic in appropriate `internal/` package
4. Add tests in corresponding `*_test.go` file
5. Update documentation if needed

### Extending Metadata
The `Metadata` struct in `internal/agent/metadata.go` uses `omitempty` for backwards compatibility. New fields can be added without breaking existing metadata files.

## Dependencies

Minimal external dependencies (see go.mod):
- `github.com/spf13/cobra`: CLI framework
- Standard library for everything else (git, file operations, JSON)

## Module Path

The Go module path is `github.com/posit-dev/wt`. When importing internal packages:
```go
import "github.com/posit-dev/wt/internal/agent"
```
