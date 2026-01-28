# Task-Centric CLI Refactoring - Implementation Summary

## Overview

Successfully refactored `wt` from an agent/queue/lock hierarchy to a task-centric architecture where task-id is the primary organizational principle.

## Files Modified

### New Files Created

1. **cmd/wt/lock_commands.go** (228 lines)
   - Contains `newTaskLockCmd()` and all task-scoped lock operations
   - Commands: `claim <agent>`, `release <agent>`, `list`, `clean`
   - Extracts task-id from context instead of positional arguments

2. **cmd/wt/queue_commands.go** (212 lines)
   - Contains `newTaskQueueCmd()` and all task-scoped queue operations
   - Commands: `add`, `list`, `get`, `remove`
   - Extracts task-id from context instead of positional arguments

3. **cmd/wt/agent_commands.go** (313 lines)
   - Contains `newTaskAgentCmd()` and task-scoped agent operations
   - Commands: `remove <name>`, `list`, `check <name>`, `sync <name>`, `prune`, `status`
   - All operations scoped under task (can be extended to support global operations)

### Modified Files

1. **cmd/wt/utils.go** (+7 lines)
   - Added `taskIDKey` context constant
   - Added `withTaskID()` function to store task-id in context
   - Added `getTaskID()` function to retrieve task-id from context

2. **cmd/wt/main.go** (231 lines, was 910)
   - Complete restructure of root command
   - Removed: `newAgentCmd()`, `newQueueCmd()`, `newLockCmd()` (hierarchical structure)
   - New: `newCreateCmd()` (promoted from `newAgentCreateCmd()`)
   - New: `newTaskCmd()` (dynamic task-id router)
   - New: `runRootOrSetup()` (intelligent routing based on URL vs task operations)
   - Renamed: `runSetup()` unchanged but integrated into new routing logic
   - Kept: `formatDuration()` utility function

3. **cmd/wt/main_test.go** (NO CHANGES YET - will be addressed next)
   - 986 lines - requires complete rewrite for new command hierarchy
   - Should test: new routing logic, context propagation, task-scoped operations

## New Command Hierarchy

### Before (Old Structure)
```
wt agent create <name> <task-id> [base]
wt agent remove <name>
wt agent list
wt agent check <name>
wt agent sync <name>
wt agent prune
wt agent status

wt lock claim <task-id> <agent>
wt lock release <task-id> <agent>
wt lock list
wt lock clean

wt queue add <task-id>
wt queue list
wt queue get <task-id>
wt queue remove <task-id>
```

### After (New Structure)
```
wt create <name> <task-id> [base]              # Promoted to root

wt <task-id> lock claim <agent>
wt <task-id> lock release <agent>
wt <task-id> lock list
wt <task-id> lock clean

wt <task-id> queue add
wt <task-id> queue list
wt <task-id> queue get
wt <task-id> queue remove

wt <task-id> agent remove <name>
wt <task-id> agent list
wt <task-id> agent check <name>
wt <task-id> agent sync <name>
wt <task-id> agent prune
wt <task-id> agent status

wt https://... [name]                         # Setup unchanged
wt myproject                                  # Setup unchanged
```

## Key Implementation Details

### 1. Context Propagation
- Added `taskIDKey` context constant to store task-id
- Functions `withTaskID()` and `getTaskID()` enable passing task-id through command tree
- Both `targetPath` and `taskID` now available in all child command contexts

### 2. Dynamic Task-ID Routing
- `newTaskCmd()` serves as parent for all task-scoped operations
- Extracts task-id from `args[0]` in `PersistentPreRunE`
- Child commands (lock, queue, agent) access via `getTaskID(ctx)`
- Replaces `args[0]` of child commands to be operation-specific (e.g., agent-name instead of task-id)

### 3. Command Promotion
- `newAgentCreateCmd()` promoted to `newCreateCmd()` at root level
- Uses `findWtRoot()` to ensure it's called within a wt directory
- Maintains identical functionality to original

### 4. Root Routing
- `runRootOrSetup()` intelligently routes based on context
- If first arg is URL → execute repo setup
- If not in .bare directory and not URL → execute repo setup
- Otherwise → not recognized (returns error)

### 5. Breaking Changes
- **REMOVED**: `wt agent ...` (entire subcommand tree)
- **REMOVED**: `wt queue ...` (entire subcommand tree)
- **REMOVED**: `wt lock ...` (entire subcommand tree)
- **PROMOTED**: `wt agent create` → `wt create` (root level)
- **RESTRUCTURED**: All lock/queue/agent operations now under task: `wt <task-id> <operation>`

## Code Statistics

| File | Lines | Purpose |
|------|-------|---------|
| main.go | 231 | Root command & task router |
| lock_commands.go | 228 | Task-scoped lock operations |
| queue_commands.go | 212 | Task-scoped queue operations |
| agent_commands.go | 313 | Task-scoped agent operations |
| utils.go | 151 | Context & utility functions (+7 new) |
| main_test.go | 986 | Tests (NEEDS REWRITE) |
| **Total (cmd/wt)** | **3031** | **Main command CLI** |

## Behavioral Changes

### Task-ID Extraction
- **Old**: Task-id explicitly in positional args
  - `wt lock claim task-1 alice`
  - `wt queue add task-1 --priority high`

- **New**: Task-id in command hierarchy
  - `wt task-1 lock claim alice`
  - `wt task-1 queue add --priority high`

### Context Flow
- All operations now receive task-id via context
- Commands no longer need to parse task-id from positional args
- First positional arg becomes operation-specific (agent name, etc.)

### Lock Operations
- `newTaskLockListCmd()` now filters locks by task-id
- `newTaskLockCleanCmd()` cleans only stale locks for that task
- Task-scoped cleanup instead of global cleanup

### Queue Operations
- Queue operations now scoped under task
- `newTaskQueueListCmd()` shows details for that specific task
- `newTaskQueueRemoveCmd()` removes task from queue

### Agent Operations
- Agent operations now scoped under task (can be extended for global operations)
- Maintains same functionality but organized differently

## Unused Code (for cleanup)

The following files/functions can be removed from main_old.go when confident:
- `newAgentCmd()` (109-134)
- `newQueueCmd()` (456-478)
- `newLockCmd()` (694-716)

## Next Steps

1. **Rewrite Tests** (cmd/wt/main_test.go)
   - Delete old tests for agent/queue/lock hierarchies
   - Add tests for:
     - Root command routing (URL vs task detection)
     - Context propagation of task-id
     - Task-scoped operations
     - Integration tests for new hierarchy

2. **Update Documentation** (CLAUDE.md)
   - Update command hierarchy diagram
   - Update usage examples
   - Update workflow documentation

3. **Test the Refactored CLI**
   - Manual testing of: `wt create alice task-1`
   - Manual testing of: `wt task-1 lock claim alice`
   - Manual testing of: `wt task-1 queue add`
   - Verify repo setup still works: `wt myproject`, `wt https://...`

4. **Performance Verification**
   - Ensure no performance regressions
   - Verify context propagation efficiency

5. **Migration Guide** (if needed)
   - Document breaking changes for users
   - Show command mapping from old to new
