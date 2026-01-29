# Claude Code Hooks for `wt`

This directory contains Claude Code hooks that automatically integrate `wt` into the agent lifecycle, providing isolated worktrees for parallel execution.

## Overview

These hooks implement the workspace isolation layer described in the project's CLAUDE.md, ensuring that multiple Claude Code sessions and subagents can work simultaneously without conflicts.

## Hook Scripts

### `setup-init.sh` (Setup Hook)
**Trigger**: `claude --init` or `claude --init-only`

Initializes the `wt` repository structure when a project is set up.

**Behavior**:
- Checks if `wt` is available
- Verifies the directory is a git repository
- Converts to bare repository structure with `wt init .`
- Adds context to Claude about the workspace setup

### `session-start.sh` (SessionStart Hook)
**Trigger**: Every time a Claude Code session starts or resumes

Creates an isolated worktree for each Claude Code session.

**Behavior**:
- Generates worktree name: `session-<session-id>` or `<agent-type>-<session-id>`
- Creates worktree if it doesn't exist
- Provides Claude with context about the isolated workspace
- Skips if already in a session worktree (prevents nesting)

**Worktree Naming**:
- Regular session: `session-a1b2c3d4/`
- Agent session: `explore-a1b2c3d4/`, `plan-a1b2c3d4/`, etc.

### `subagent-start.sh` (SubagentStart Hook)
**Trigger**: When Claude spawns a subagent (Task tool)

Creates isolated worktrees for parallel subagent execution.

**Behavior**:
- Generates worktree name: `<agent-type>-<agent-id>`
- Creates worktree for the subagent
- Allows multiple subagents to work in parallel
- Fails silently if worktree creation fails (subagent uses current directory)

**Worktree Naming Examples**:
- Explore agent: `explore-abc12345/`
- Plan agent: `plan-def67890/`
- Bash agent: `bash-ghi24680/`

### `subagent-stop.sh` (SubagentStop Hook)
**Trigger**: When a subagent completes its work

Checks for conflicts and provides guidance on next steps.

**Behavior**:
- Runs `wt check <worktree>` to detect conflicts
- If conflicts found: **blocks** with decision and provides conflict details
- If no conflicts: provides context about successful completion
- Preserves worktree for review (doesn't auto-remove)

**Output**:
- **With conflicts**: Blocks Claude and shows conflict details
- **Without conflicts**: Provides commands to review/merge changes

### `session-end.sh` (SessionEnd Hook)
**Trigger**: When a Claude Code session ends

Cleans up session worktrees safely.

**Behavior**:
- Checks for uncommitted changes (preserves if found)
- Checks for unpushed commits (preserves if found)
- Removes worktree only if clean
- Runs `wt prune --older-than 7d` to clean up stale worktrees

**Safety**:
- Never removes worktrees with uncommitted changes
- Never removes worktrees with unpushed commits
- Preserves work for manual review when needed

## Workflow Example

```bash
# 1. User starts Claude Code
$ claude
# → SessionStart hook creates: session-abc12345/

# 2. Claude spawns an Explore subagent
# → SubagentStart hook creates: explore-def67890/

# 3. Explore subagent completes
# → SubagentStop hook checks for conflicts
#   - No conflicts: Provides merge instructions
#   - Conflicts: Blocks and shows details

# 4. Claude spawns a Plan subagent (parallel)
# → SubagentStart hook creates: plan-ghi24680/
# → Both subagents work simultaneously without conflicts

# 5. User exits Claude
# → SessionEnd hook cleans up session-abc12345/ (if clean)
```

## Directory Structure

After hooks run, the repository structure looks like:

```
~/wt/project/
├── .bare/                      # Bare git repository (shared)
├── .wt/                        # Worktree metadata
├── main/                       # Primary worktree
├── session-abc12345/           # Session worktree (auto-created)
├── explore-def67890/           # Explore subagent worktree
└── plan-ghi24680/              # Plan subagent worktree (parallel)
```

## Configuration

The hooks are configured in `.claude/settings.json`:

```json
{
  "hooks": {
    "Setup": [...],          // Repository initialization
    "SessionStart": [...],   // Session isolation
    "SubagentStart": [...],  // Subagent isolation
    "SubagentStop": [...],   // Conflict detection
    "SessionEnd": [...]      // Cleanup
  }
}
```

## Input/Output

### Input (via stdin)
All hooks receive JSON input with session context:
```json
{
  "session_id": "abc123...",
  "agent_id": "agent-def456...",
  "agent_type": "Explore",
  "cwd": "/path/to/project",
  "transcript_path": "...",
  "hook_event_name": "SubagentStart"
}
```

### Output (via stdout)
Hooks use JSON output for structured control:

**SessionStart/Setup** - Add context:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Working in isolated worktree: session-abc12345/"
  }
}
```

**SubagentStop** - Block on conflicts:
```json
{
  "decision": "block",
  "reason": "Conflicts detected in worktree 'explore-abc12345'..."
}
```

## Requirements

- `wt` command must be in PATH
- `jq` for JSON parsing
- Repository must be git-initialized
- Bash shell

## Debugging

Enable verbose mode to see hook execution:
```bash
claude --debug
```

Check hook status:
```bash
claude  # Then type /hooks in the session
```

## Safety Features

1. **Graceful Degradation**: Hooks fail silently if `wt` is unavailable
2. **No Nesting**: Skips worktree creation if already in one
3. **Conflict Detection**: Warns about conflicts before merging
4. **Preserve Work**: Never removes worktrees with uncommitted/unpushed changes
5. **Auto-Cleanup**: Prunes stale worktrees (7+ days old)

## Benefits

- **No Conflicts**: Multiple agents never touch the same files
- **True Parallelism**: Subagents work simultaneously
- **Automatic**: No manual workspace management needed
- **Safe**: Changes isolated until explicitly merged
- **Efficient**: Shared object database, separate working trees

## Troubleshooting

**Hook not running?**
- Check `/hooks` menu in Claude Code session
- Verify scripts are executable: `chmod +x .claude/hooks/*.sh`
- Run `claude --debug` to see hook execution logs

**Worktree creation fails?**
- Ensure `wt` is installed and in PATH
- Check if repository is initialized: `wt init .`
- Verify git repository is clean

**Conflicts detected?**
- Review worktree: `cd <worktree>/ && git status`
- Manually resolve conflicts
- Or discard: `wt remove <worktree>`

## Related Documentation

- [Claude Code Hooks Reference](https://code.claude.com/docs/en/hooks)
- [wt Documentation](../../README.md)
