# Migration Guide: Task-Centric CLI Refactoring

## For Users - Command Migration

### Agent Commands

| Old Command | New Command | Notes |
|-------------|-------------|-------|
| `wt agent create alice task-1 [main]` | `wt create alice task-1 [main]` | Promoted to root |
| `wt agent remove alice` | `wt task-1 agent remove alice` | Task-scoped |
| `wt agent list` | `wt task-1 agent list` | Task-scoped |
| `wt agent check alice` | `wt task-1 agent check alice` | Task-scoped |
| `wt agent sync alice --auto-rebase` | `wt task-1 agent sync alice --auto-rebase` | Task-scoped |
| `wt agent prune --older-than 14d` | `wt task-1 agent prune --older-than 14d` | Task-scoped |
| `wt agent status` | `wt task-1 agent status` | Task-scoped |

### Lock Commands

| Old Command | New Command | Notes |
|-------------|-------------|-------|
| `wt lock claim task-1 alice --pid 1234` | `wt task-1 lock claim alice --pid 1234` | Task in hierarchy |
| `wt lock release task-1 alice` | `wt task-1 lock release alice` | Task in hierarchy |
| `wt lock release task-1 alice --force` | `wt task-1 lock release alice --force` | Task in hierarchy |
| `wt lock list` | `wt task-1 lock list` | Task-scoped, filtered |
| `wt lock clean --timeout 1h` | `wt task-1 lock clean --timeout 1h` | Task-scoped, filtered |
| `wt lock clean --dry-run` | `wt task-1 lock clean --dry-run` | Task-scoped, filtered |

### Queue Commands

| Old Command | New Command | Notes |
|-------------|-------------|-------|
| `wt queue add task-1 --priority high` | `wt task-1 queue add --priority high` | Task in hierarchy |
| `wt queue list --state pending` | `wt task-1 queue list` | Task-scoped |
| `wt queue get task-1` | `wt task-1 queue get` | Task in hierarchy |
| `wt queue remove task-1` | `wt task-1 queue remove` | Task in hierarchy |

### Unchanged Commands

These commands remain **exactly the same**:
- `wt https://github.com/user/repo` - Clone from remote
- `wt https://github.com/user/repo my-proj` - Clone with custom name
- `wt myproject` - Initialize local project
- `wt --help` - Show help
- `wt --version` - Show version

## For Developers - Implementation Guide

### File Changes to Apply

#### 1. Update `cmd/wt/utils.go`

Add after line 15 (const targetPathKey):
```go
const (
    targetPathKey contextKey = "targetPath"
    taskIDKey     contextKey = "taskID"
)
```

Add after `getTargetPath()` function (after line 127):
```go
// Context helpers for task ID
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

#### 2. Replace `cmd/wt/main.go`

The new main.go is much simpler (231 lines vs 910):
- Contains only: root command, create command, task router
- Imports: agent, conflict, parse, repo, cobra (removes queue, locking from main)
- Exports: 3 command builders, 1 setup function, 1 router function, 1 utility

#### 3. Create `cmd/wt/lock_commands.go`

New file with 228 lines containing:
- `newTaskLockCmd()` - Parent command
- `newTaskLockClaimCmd()` - Claim lock (args: agent-name)
- `newTaskLockReleaseCmd()` - Release lock (args: agent-name)
- `newTaskLockListCmd()` - List locks (filters by task-id)
- `newTaskLockCleanCmd()` - Clean stale locks (task-filtered)

All commands extract task-id from context via `getTaskID(ctx)`.

#### 4. Create `cmd/wt/queue_commands.go`

New file with 212 lines containing:
- `newTaskQueueCmd()` - Parent command
- `newTaskQueueAddCmd()` - Add task (no task-id arg)
- `newTaskQueueListCmd()` - Show task details (no task-id arg)
- `newTaskQueueGetCmd()` - Get task details (no task-id arg)
- `newTaskQueueRemoveCmd()` - Remove task (no task-id arg)

All commands extract task-id from context via `getTaskID(ctx)`.

#### 5. Create `cmd/wt/agent_commands.go`

New file with 313 lines containing:
- `newTaskAgentCmd()` - Parent command
- `newTaskAgentRemoveCmd()` - Remove agent
- `newTaskAgentListCmd()` - List agents
- `newTaskAgentCheckCmd()` - Check conflicts
- `newTaskAgentSyncCmd()` - Sync agent
- `newTaskAgentPruneCmd()` - Prune old agents
- `newTaskAgentStatusCmd()` - Show dashboard

All commands are identical to old implementations but organized under task.

#### 6. Delete from `cmd/wt/main.go`

Remove these functions entirely:
- `newAgentCmd()` (lines 109-134)
- `newAgentRemoveCmd()` (lines 173-202)
- `newAgentListCmd()` (lines 204-238)
- `newAgentCheckCmd()` (lines 240-282)
- `newAgentSyncCmd()` (lines 284-331)
- `newAgentPruneCmd()` (lines 333-403)
- `newAgentStatusCmd()` (lines 405-450)
- `newQueueCmd()` (lines 456-478)
- `newQueueAddCmd()` (lines 480-543)
- `newQueueListCmd()` (lines 545-618)
- `newQueueGetCmd()` (lines 620-665)
- `newQueueRemoveCmd()` (lines 667-689)
- `newLockCmd()` (lines 694-716)
- `newLockClaimCmd()` (lines 718-745)
- `newLockReleaseCmd()` (lines 747-787)
- `newLockListCmd()` (lines 789-832)
- `newLockCleanCmd()` (lines 834-897)

Keep:
- `formatDuration()` function (move to main.go)
- `runSetup()` function (integrate into routing)

#### 7. Rewrite `cmd/wt/main.go` Root

New structure:
```go
func newRootCmd() *cobra.Command {
    // No RunE or route to setup based on context
    cmd.AddCommand(newCreateCmd())
    cmd.AddCommand(newTaskCmd())
    return cmd
}

func runRootOrSetup(cmd, args) error {
    // Route based on URL or .bare directory presence
}

func newCreateCmd() *cobra.Command {
    // Promoted from newAgentCreateCmd
}

func newTaskCmd() *cobra.Command {
    // Dynamic task-id router
    cmd.AddCommand(newTaskLockCmd())
    cmd.AddCommand(newTaskQueueCmd())
    cmd.AddCommand(newTaskAgentCmd())
    return cmd
}
```

#### 8. Update `cmd/wt/main_test.go`

Delete tests for:
- `TestAgentCommand` - No longer exists
- `TestQueueCmd` - No longer exists
- `TestLockCmd` - No longer exists
- All `TestAgentXyzCmd_*` tests
- All `TestQueueXyzCmd_*` tests
- All `TestLockXyzCmd_*` tests

Add tests for:
- Root command routing (URL detection)
- Context propagation of task-id
- Each task-scoped operation
- Integration tests for new hierarchy

#### 9. Verify No Changes Needed

These files require **ZERO changes**:
- `internal/agent/agent.go`
- `internal/agent/metadata.go`
- `internal/locking/locking.go`
- `internal/queue/queue.go`
- `internal/conflict/conflict.go`
- `internal/git/git.go`
- `internal/repo/repo.go`
- `internal/parse/parse.go`
- `go.mod`
- `go.sum`
- `Makefile`

## Checklist for Implementation

### Phase 1: Code Changes
- [ ] Update `cmd/wt/utils.go` - Add taskID context support
- [ ] Create `cmd/wt/lock_commands.go` - New task-scoped lock commands
- [ ] Create `cmd/wt/queue_commands.go` - New task-scoped queue commands
- [ ] Create `cmd/wt/agent_commands.go` - New task-scoped agent commands
- [ ] Rewrite `cmd/wt/main.go` - Root command, create command, task router
- [ ] Backup `cmd/wt/main.go` to `cmd/wt/main_old.go` (for reference)

### Phase 2: Testing & Validation
- [ ] Verify code compiles: `go build ./cmd/wt`
- [ ] Verify no syntax errors: `gofmt -l cmd/wt/*.go`
- [ ] Run linter: `go vet ./cmd/wt`
- [ ] Test help: `wt --help`, `wt create --help`, `wt <task> lock --help`

### Phase 3: Tests
- [ ] Rewrite `cmd/wt/main_test.go`
- [ ] Add routing tests
- [ ] Add context propagation tests
- [ ] Add integration tests for new hierarchy
- [ ] Run all tests: `go test ./cmd/wt`

### Phase 4: Documentation
- [ ] Update `CLAUDE.md` command hierarchy
- [ ] Update `CLAUDE.md` usage examples
- [ ] Update `README.md` if applicable
- [ ] Add migration guide for users

### Phase 5: Final Validation
- [ ] Manual test: `wt create alice task-1`
- [ ] Manual test: `wt task-1 lock claim alice`
- [ ] Manual test: `wt task-1 queue add --priority high`
- [ ] Manual test: `wt task-1 agent list`
- [ ] Manual test: Repo setup still works
- [ ] Verify error messages are clear and helpful

## Rollback Plan (If Needed)

1. Keep `main_old.go` as reference
2. If issues arise, can revert by:
   - Copy `main_old.go` back to `main.go`
   - Delete: `lock_commands.go`, `queue_commands.go`, `agent_commands.go`
   - Revert `utils.go` to previous version

## Breaking Changes Summary

This refactoring introduces **COMPLETE breaking changes**:

❌ **REMOVED**: Old hierarchical command structure
- `wt agent ...` (entire subtree)
- `wt queue ...` (entire subtree)
- `wt lock ...` (entire subtree)

✅ **ADDED**: New task-centric structure
- `wt create` (root level)
- `wt <task-id> lock ...`
- `wt <task-id> queue ...`
- `wt <task-id> agent ...`

✅ **UNCHANGED**: Setup commands
- `wt <url|name>`

## Expected Issues & Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| "Lock already exists" | Task-id mismatch | Ensure using correct task-id |
| Context is empty | Task-id not extracted | Verify `withTaskID()` in PersistentPreRunE |
| Commands not found | File organization | Verify all files in cmd/wt/ |
| Tests fail | Old tests expecting old structure | Rewrite tests for new structure |

## Performance Expectations

- No measurable change in performance
- Context propagation adds negligible overhead
- All operations remain O(n) with same or better constants
- Dynamic routing has no impact on execution speed

---

**Implementation Ready: All components designed and structured for replacement of existing code.**
