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
│   └── metadata/              # JSON metadata for each agent
├── .git                       # Pointer file to .bare
└── main/                      # Primary worktree (base branch)
```

### Agent Worktrees

Agent worktrees can be created manually or automatically:

**Manual Creation** (`wt add <name> [base-branch]`):
- **Directory**: `<name>/` (e.g., `alice/`, `bob/`)
- **Branch**: `<name>` (e.g., `alice`, `bob`)
- **Base**: Specified or detected from current context
- **Metadata**: `.wt/metadata/<name>.json`

**Automatic Creation** (Claude Code hooks):
- **Directory**: `<agent-type>-<agent-id-short>/` (e.g., `plan-a6c59d07/`)
- **Branch**: `<agent-type>-<agent-id-short>`
- **Base**: Current branch (context-aware)
- **Metadata**: `.wt/metadata/<agent-type>-<agent-id-short>.json`
- **Isolation**: Based on agent type, keywords, and concurrency

### Core System

**Agent Management** (`internal/agent/`): Worktree lifecycle, conflict detection, and syncing
- Metadata stored in: `.wt/metadata/`
- Commands: add, remove, list, status, check, sync, prune

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

The CLI is built with Cobra and organized as flat commands:
```
wt                                   # Root command
├── clone <url> [target-dir]        # Clone repository
├── init <target-dir>               # Initialize local project
├── add <name> [base-branch]        # Create agent worktree
├── remove <name>                   # Remove agent worktree
├── list                            # List all agents
├── status                          # Dashboard view
├── check <name>                    # Conflict detection
├── sync <name>                     # Sync with base branch
├── prune [--older-than 7d]         # Remove stale agents
└── hooks                           # Claude Code integration
    ├── install [dir]               # Install hooks to project
    ├── config                      # Print settings.json
    └── list                        # List hook scripts
```

### Package Organization

**Core packages** (internal/):
- `parse/`: URL and argument parsing (clone vs local detection)
- `git/`: Git command wrapper with error handling and upstream tracking
- `repo/`: Repository setup (bare repo creation, worktree initialization)
- `conflict/`: Conflict detection via git merge-tree
- `agent/`: Agent worktree operations and metadata management
- `worktree/`: Base branch detection and context-aware worktree operations
- `hooks/`: Claude Code hooks distribution (embedded templates)

### Key Design Patterns

**Metadata-Driven Architecture**: Each agent worktree has JSON metadata tracking:
- Agent name and branch associations
- Creation timestamp and last activity
- Base branch reference
- Status information

**Non-Destructive Conflict Detection**: The `check` command:
1. Uses `git merge-tree` for simulation
2. Reports divergence (commits ahead/behind)
3. Lists conflicting files without modifying working directory
4. Updates last_activity timestamp in metadata

**Clone and Upstream Tracking**: When cloning repositories:
1. Performs bare clone to `.bare/` directory
2. Fetches remote branches to populate `refs/remotes/origin/*`
3. Creates primary worktree with upstream tracking configured
4. Enables push/pull operations without additional configuration
5. All worktrees automatically track their corresponding remote branches

### Time-Based Operations

**Age Calculation** (internal/agent/):
- Based on metadata `last_activity` field
- Human-readable format: "5m", "2h", "3d"
- Used by `prune` command for cleanup (default: 7 days)

## Important Conventions

### Branch Naming
Agent branches follow these patterns:
- **Manual**: `<name>` (e.g., `alice`, `feature-work`) - set via `wt add <name>`
- **Automatic**: `<agent-type>-<agent-id-short>` (e.g., `plan-a6c59d07`) - created by Claude Code hooks
- Branches are automatically created and managed by the tool

### Base Branch Detection
The tool uses **context-aware base branch detection**:
- During repo setup: Auto-detects the default branch (usually "main" or "master")
- For agent worktrees: Branches from the current working context (if you're on `feature-x`, agents branch from `feature-x`, not `main`)
- This enables hierarchical workflows where sub-agents inherit your current branch context
- Agent operations use the base branch for conflict checking and syncing

### Claude Code Hooks Integration

The tool provides intelligent hooks for automatic agent isolation:

**SubagentStart Hook Decision Logic** (priority order):
1. **Inline keywords** (highest priority): Agents with "read", "analyze", "explain", "document", "describe", "query", "inspect" work in current directory
2. **Agent types**: Plan, Explore, Test, Execute, Bash agents get automatic isolation
3. **Isolation keywords**: Agents with "refactor", "migrate", "scaffold", "restructure", "rebuild", "rewrite" get isolated
4. **Concurrency**: When 2+ agents are active, new agents get isolated to prevent conflicts
5. **Default**: Work in current directory (no isolation)

**Worktree Naming**: Auto-generated as `<agent-type>-<agent-id-short>` (e.g., `plan-a6c59d07`)

**Base Branch**: Inherits from current context (dynamic detection)

**SessionStart Hook**: Provides context about wt availability but does not auto-create worktrees. Users work in current directory by default.

### Context Propagation
Commands auto-discover the worktree root by searching for `.bare/` directory upwards from current working directory. Commands work from any subdirectory.

### Metadata Updates
The following commands automatically update `last_activity`:
- `wt check <name>`
- `wt sync <name>`
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

### Maintaining Claude Code Hooks

**Important**: Hook files exist in two locations and must be kept in sync:
- **Source of truth**: `.claude/hooks/` and `.claude/settings.json`
- **Distributed templates**: `internal/hooks/templates/` (embedded in binary)

#### Making Changes to Hooks

1. Edit hooks in `.claude/hooks/` or `.claude/settings.json`
2. Test your changes (run Claude Code with the hooks)
3. Sync templates: `make sync-hooks`
4. Verify sync: `make test` (test will fail if out of sync)
5. Commit both `.claude/` and `internal/hooks/templates/`

#### Automatic Sync Verification

The test `internal/hooks/hooks_test.go` verifies templates match source:
```bash
# This will fail if hooks are out of sync
go test ./internal/hooks/

# Sync and test
make sync-hooks
make test
```

**CI Integration**: The test suite automatically catches divergence, preventing out-of-sync hooks from being merged.

#### Why This Matters

- Users install hooks via `wt hooks install`, which uses embedded templates
- If templates drift from `.claude/` source, users get outdated hooks
- The sync test ensures distributed hooks always match development hooks

## Dependencies

Minimal external dependencies (see go.mod):
- `github.com/spf13/cobra`: CLI framework
- Standard library for everything else (git, file operations, JSON)

## Module Path

The Go module path is `github.com/tdstein/wt`. When importing internal packages:
```go
import "github.com/tdstein/wt/internal/agent"
```
