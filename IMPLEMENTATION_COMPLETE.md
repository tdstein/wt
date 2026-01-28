# Task-Centric CLI Refactoring - Implementation Complete ✓

## Executive Summary

Successfully refactored the `wt` CLI from a hierarchical agent/queue/lock command structure to a task-centric architecture where **task-id is the primary organizational principle**. This is a **complete breaking change** with zero backwards compatibility.

## Implementation Status

| Component | Status | Lines | Notes |
|-----------|--------|-------|-------|
| Context Utilities | ✅ DONE | +7 | Added taskID context support in utils.go |
| Root Command | ✅ DONE | 231 | Restructured with intelligent routing |
| Create Command | ✅ DONE | 49 | Promoted from agent subcommand |
| Task Router | ✅ DONE | 24 | Dynamic task-id extraction |
| Lock Commands | ✅ DONE | 228 | 4 commands, task-scoped |
| Queue Commands | ✅ DONE | 212 | 4 commands, task-scoped |
| Agent Commands | ✅ DONE | 313 | 6 commands, task-scoped |
| Tests | ⏳ TODO | 986 | Requires rewrite |
| Documentation | ⏳ TODO | - | CLAUDE.md needs updating |

**Total CLI Code: 3,031 lines (including tests)**

## Architecture Transformation

### The New Model

```
wt [repo-url|name]                          # Setup (unchanged)
├─ wt create <name> <task-id> [base]        # CREATE: Root-level agent creation
└─ wt <task-id> <operation>                 # TASK: Task-scoped operations
   ├─ lock claim <agent> [--pid N]          # Claim lock
   ├─ lock release <agent> [--force]        # Release lock
   ├─ lock list                             # List task locks
   ├─ lock clean [--timeout] [--dry-run]    # Clean stale locks
   ├─ queue add [flags]                     # Add to queue
   ├─ queue list                            # Show task details
   ├─ queue get                             # Get task details
   ├─ queue remove                          # Remove from queue
   ├─ agent remove <name> [--delete-branch] # Remove agent
   ├─ agent list                            # List agents
   ├─ agent check <name>                    # Check conflicts
   ├─ agent sync <name> [--auto-rebase]     # Sync agent
   ├─ agent prune [--older-than] [--dry-run]# Prune old agents
   └─ agent status                          # Show dashboard
```

## File Organization

### Main Command Files
```
cmd/wt/
├── main.go                 # Root command, Create cmd, Task router
├── lock_commands.go        # Task-scoped lock operations
├── queue_commands.go       # Task-scoped queue operations
├── agent_commands.go       # Task-scoped agent operations
├── utils.go                # Context helpers (with new taskID support)
├── main_test.go            # Tests (NEEDS REWRITE)
└── main_old.go             # Backup of original (for reference)
```

### No Changes Needed
All internal packages remain **100% unchanged**:
- `internal/agent/` - Agent worktree operations
- `internal/queue/` - Task queue management
- `internal/locking/` - Task locking
- `internal/conflict/` - Conflict detection
- `internal/git/` - Git operations
- `internal/repo/` - Repository setup
- `internal/parse/` - Argument parsing

## Breaking Changes

### Deleted Commands (No Replacement)
- ❌ `wt agent create ...` → ✅ `wt create ...` (promoted)
- ❌ `wt agent remove ...` → ✅ `wt <task-id> agent remove ...` (scoped)
- ❌ `wt agent list` → ✅ `wt <task-id> agent list` (scoped)
- ❌ `wt agent check ...` → ✅ `wt <task-id> agent check ...` (scoped)
- ❌ `wt agent sync ...` → ✅ `wt <task-id> agent sync ...` (scoped)
- ❌ `wt agent prune` → ✅ `wt <task-id> agent prune` (scoped)
- ❌ `wt agent status` → ✅ `wt <task-id> agent status` (scoped)
- ❌ `wt queue add <task-id>` → ✅ `wt <task-id> queue add`
- ❌ `wt queue list` → ✅ `wt <task-id> queue list`
- ❌ `wt queue get <task-id>` → ✅ `wt <task-id> queue get`
- ❌ `wt queue remove <task-id>` → ✅ `wt <task-id> queue remove`
- ❌ `wt lock claim <task-id> ...` → ✅ `wt <task-id> lock claim ...`
- ❌ `wt lock release <task-id> ...` → ✅ `wt <task-id> lock release ...`
- ❌ `wt lock list` → ✅ `wt <task-id> lock list`
- ❌ `wt lock clean` → ✅ `wt <task-id> lock clean`

### Preserved Commands
- ✅ `wt <repo-url> [name]` - Clone from remote (unchanged)
- ✅ `wt <name>` - Initialize local project (unchanged)
- ✅ Help & version flags work as before

## Implementation Details

### 1. Context Propagation

**New in utils.go:**
```go
const taskIDKey contextKey = "taskID"

func withTaskID(ctx context.Context, taskID string) context.Context {
    return context.WithValue(ctx, taskIDKey, taskID)
}

func getTaskID(ctx context.Context) string {
    if taskID, ok := ctx.Value(taskIDKey).(string); ok {
        return taskID
    }
    return ""
}
```

### 2. Task Router (main.go)

The `newTaskCmd()` function acts as a dynamic router:
- Extracts task-id from `args[0]`
- Stores it in context via `withTaskID()`
- All child commands (lock, queue, agent) receive it via context
- Child commands no longer need task-id as positional arg

### 3. Command Promotion

`newCreateCmd()` is essentially `newAgentCreateCmd()` moved to root level:
- Same logic, same flags, same behavior
- Wrapped with `findWtRoot()` to ensure it's in a wt directory
- Users now run: `wt create alice task-1` instead of `wt agent create alice task-1`

### 4. Lock Commands - Task Scoping

Example: `newTaskLockListCmd()`
```go
taskID := getTaskID(cmd.Context())     // Get task-id from context
mgr := locking.NewManager(targetPath)
locks, err := mgr.ListAll()            // Get all locks
// Filter to this task
var taskLocks []*locking.Lock
for _, lock := range locks {
    if lock.TaskID == taskID {
        taskLocks = append(taskLocks, lock)
    }
}
```

All operations extract task-id from context instead of positional args.

## Command Examples

### Old vs New

#### Create Agent
```bash
# Old
wt agent create alice task-1

# New
wt create alice task-1
```

#### Claim Lock
```bash
# Old
wt lock claim task-1 alice --pid 1234

# New
wt task-1 lock claim alice --pid 1234
```

#### Add to Queue
```bash
# Old
wt queue add task-1 --priority high --depends-on task-2

# New
wt task-1 queue add --priority high --depends-on task-2
```

#### Check Agent
```bash
# Old
wt agent check alice

# New
wt task-1 agent check alice
```

#### List Locks for Task
```bash
# Old (showed all locks, had to parse)
wt lock list

# New (shows only this task's locks)
wt task-1 lock list
```

## Testing Required

### Unit Tests
- Root command routing (URL vs task detection)
- Context propagation for task-id
- Each task-scoped operation

### Integration Tests
- `wt create alice task-1` → creates worktree
- `wt task-1 lock claim alice` → claims lock
- `wt task-1 queue add` → adds to queue
- All lock operations filtered by task
- All queue operations filtered by task

### End-to-End Tests
- Repo setup with `wt myproject`
- Repo clone with `wt https://...`
- Full workflow: create → lock → queue → operations

## What's Next

1. **Rewrite main_test.go** (986 lines)
   - Delete tests for old hierarchical structure
   - Add tests for new command routing and context
   - Test task-scoped operation filtering

2. **Update CLAUDE.md**
   - Update command hierarchy diagram
   - Update usage examples and workflows
   - Document breaking changes

3. **Manual Testing**
   - Test each new command combination
   - Verify repo setup still works
   - Check error messages are clear

4. **Performance Validation**
   - Ensure no regressions
   - Verify context propagation is efficient

## Key Decisions Made

✅ **Breaking Changes Embraced**
- No aliases or deprecation warnings
- Clean break from old structure
- Simpler, more intuitive hierarchy

✅ **No Internal Package Changes**
- Managers already parameterized
- All logic moved to CLI layer
- Internal packages fully reusable

✅ **Task-Scoped by Default**
- All operations tied to task-id
- Cleaner UX than `wt <operation> --task-id`
- Natural hierarchy reflects user workflow

✅ **Context-Based Propagation**
- Eliminates positional arg confusion
- Task-id available throughout command tree
- Cleaner argument parsing

## Files Modified Summary

| File | Change | Impact |
|------|--------|--------|
| utils.go | +7 lines | Minor - Added context helpers |
| main.go | -679 lines, +231 lines | Major - Complete restructure |
| lock_commands.go | +228 lines (new) | New file - Task-scoped locks |
| queue_commands.go | +212 lines (new) | New file - Task-scoped queue |
| agent_commands.go | +313 lines (new) | New file - Task-scoped agents |
| main_test.go | 0 lines (TBD) | 986 lines to rewrite |
| main_old.go | Original backup | Reference only |

## Success Criteria - All Met ✅

- [x] Promote `wt agent create` to `wt create`
- [x] Move lock commands under `wt <task-id> lock`
- [x] Move queue commands under `wt <task-id> queue`
- [x] Move agent commands under `wt <task-id> agent`
- [x] Implement context-based task-id propagation
- [x] Eliminate agent/queue/lock hierarchical subcommands
- [x] Make no changes to internal packages
- [x] Implement with no backwards compatibility
- [x] Create file organization that's maintainable

## Remaining Work

- [ ] Rewrite all tests (main_test.go)
- [ ] Update documentation (CLAUDE.md)
- [ ] Manual end-to-end testing
- [ ] Performance validation
- [ ] Consider: Global agent operations (`wt agent list` globally vs `wt <task> agent list`)

---

**Status: REFACTORING IMPLEMENTATION COMPLETE**
**Ready for: Testing and Documentation Updates**
