# wt Project Goals

## Vision

**Scale AI-assisted development linearly with agent count.**

When this project succeeds, a user will be able to spin up N agents on a codebase and complete N independent tasks in parallel—with the coordination infrastructure completely invisible.

## The Problem

AI agents are individually powerful but currently operate serially. Running 10 tasks means waiting for 10 sequential completions. Without coordination infrastructure:

- Multiple agents on the same repository cause file conflicts
- Race conditions occur when agents claim the same work
- There's no visibility into who is doing what
- Merge conflicts surface at the end, not during work
- Crashed agents leave abandoned work with no cleanup

This serialization bottleneck is artificial. With proper infrastructure, agents could work in parallel on different parts of a codebase simultaneously.

## The Ideal Outcome

**Parallel agents, zero coordination overhead.**

In the ideal state:

1. User defines N tasks that need to be done
2. User spawns N agents
3. Each agent claims available work automatically
4. Each agent works in complete isolation
5. Conflicts are detected early and flagged
6. Crashed agents are detected and their work is recoverable
7. Completed work merges cleanly
8. User sees N tasks completed in the time it would take one agent to do one

The infrastructure should be **invisible**—users think about what to accomplish, not how to coordinate agents.

## Success Criteria

### Correctness
- No data loss from concurrent agent operations
- No race conditions when claiming tasks
- Conflicts detected before merge time
- Crashed agent state is recoverable

### Performance
- Near-linear throughput scaling: N agents should complete N independent tasks in approximately the time one agent completes one task
- Sub-second overhead for coordination operations (create, claim, check)
- No blocking between independent agents

### Usability
- Single command to create an agent workspace
- Single command to check for conflicts
- Automatic detection and cleanup of stale work
- Clear visibility into all agent activity

### Reliability
- Self-healing: stale locks auto-expire
- Idempotent operations: safe to retry
- Non-destructive checking: conflict detection doesn't modify state

## Non-Goals

wt is **not**:

- **A task execution engine**: wt manages workspaces and coordination, not what agents do inside them
- **A CI/CD system**: wt handles local parallel work, not deployment pipelines
- **A distributed system**: wt runs on a single machine with a single filesystem
- **A replacement for git**: wt builds on git primitives (worktrees, branches) rather than replacing them

## Design Principles

### 1. Git-Native
Build on git primitives rather than around them. Worktrees, branches, and refs are the foundation. Users can always fall back to raw git commands.

### 2. File-Based State
All state (tasks, locks, metadata) is stored in JSON files in `.wt/`. This makes state inspectable, debuggable, and recoverable without special tooling.

### 3. Non-Destructive Operations
Conflict detection should never modify working directory state. Checking should be safe to run at any time, from any state.

### 4. Convention Over Configuration
Strong naming conventions (`task/<task-id>/<agent-name>`) reduce the need for explicit configuration. The tool should work with sensible defaults.

### 5. Fail-Safe Defaults
- Stale locks expire automatically
- Dry-run is available for destructive operations
- Force flags are explicit, not default

### 6. Minimal Dependencies
Standard library where possible. External dependencies only when they provide significant value (e.g., Cobra for CLI).

## Current Capabilities

### Repository Setup
| Command | Purpose |
|---------|---------|
| `wt clone <url> [target-dir]` | Clone repository with bare repo structure |
| `wt init <target-dir>` | Initialize local project with bare repo structure |

### Agent Management
| Command | Purpose |
|---------|---------|
| `wt add <name> [base-branch]` | Create isolated workspace |
| `wt list` | Show all active agents |
| `wt check <name>` | Detect conflicts non-destructively |
| `wt sync <name>` | Update agent from base branch |
| `wt remove <name>` | Clean up finished work |
| `wt prune [--older-than 7d]` | Remove stale workspaces |
| `wt status` | Dashboard view |

## Future Directions

Areas for potential expansion (not commitments):

1. **Orchestration**: Automated agent spawning based on queue depth
2. **Cross-Agent Communication**: Signal mechanisms for blocking/unblocking
3. **Automated Merge Flow**: Queue-based merge ordering
4. **Observability**: Persistent logging, metrics, dashboards
5. **Resource Management**: Caps on concurrent agents per resource
6. **Remote Coordination**: Multi-machine support

## Measuring Progress

The project moves toward its vision when:

- **More tasks run in parallel** without conflicts
- **Less user intervention** is required for coordination
- **Fewer edge cases** cause data loss or blocked work
- **Setup time decreases** for new parallel workflows
- **Recovery time decreases** when things go wrong

The ultimate measure: does adding another agent meaningfully increase throughput?
