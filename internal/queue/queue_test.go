package queue

import (
	"os"
	"testing"
	"time"
)

func setupTestManager(t *testing.T) (*Manager, string) {
	tmpDir, err := os.MkdirTemp("", "wt-queue-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	mgr := NewManager(tmpDir)
	return mgr, tmpDir
}

func cleanup(tmpDir string) {
	os.RemoveAll(tmpDir)
}

func TestManager_Init(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	if err := mgr.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify directories exist
	states := []State{StatePending, StateClaimed, StateCompleted, StateFailed}
	for _, state := range states {
		dir := mgr.stateDir(state)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("state directory not created: %s", dir)
		}
	}
}

func TestManager_Add(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	tests := []struct {
		name    string
		opts    AddOptions
		wantErr bool
	}{
		{
			name: "basic task",
			opts: AddOptions{
				TaskID:      "task-001",
				Description: "Test task",
				Priority:    PriorityNormal,
			},
			wantErr: false,
		},
		{
			name: "high priority task",
			opts: AddOptions{
				TaskID:      "task-002",
				Description: "Urgent task",
				Priority:    PriorityHigh,
			},
			wantErr: false,
		},
		{
			name: "task with dependencies",
			opts: AddOptions{
				TaskID:       "task-003",
				Description:  "Dependent task",
				Priority:     PriorityNormal,
				Dependencies: []string{"task-001"},
			},
			wantErr: false,
		},
		{
			name: "missing task ID",
			opts: AddOptions{
				Description: "No ID",
			},
			wantErr: true,
		},
		{
			name: "invalid priority",
			opts: AddOptions{
				TaskID:   "task-004",
				Priority: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.Add(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && err == nil {
				// Verify task file exists
				taskFile := mgr.taskFile(tt.opts.TaskID, StatePending)
				if _, err := os.Stat(taskFile); os.IsNotExist(err) {
					t.Errorf("task file not created: %s", taskFile)
				}
			}
		})
	}
}

func TestManager_AddDuplicate(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	opts := AddOptions{
		TaskID:      "task-001",
		Description: "Test task",
		Priority:    PriorityNormal,
	}

	// Add first time - should succeed
	if err := mgr.Add(opts); err != nil {
		t.Fatalf("first Add failed: %v", err)
	}

	// Add again - should fail
	if err := mgr.Add(opts); err == nil {
		t.Error("duplicate Add should fail")
	}
}

func TestManager_Get(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add a task
	opts := AddOptions{
		TaskID:      "task-001",
		Description: "Test task",
		Priority:    PriorityHigh,
		Tags:        []string{"test", "demo"},
	}

	if err := mgr.Add(opts); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Get the task
	task, err := mgr.Get("task-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Verify fields
	if task.TaskID != "task-001" {
		t.Errorf("TaskID = %s, want task-001", task.TaskID)
	}
	if task.Description != "Test task" {
		t.Errorf("Description = %s, want Test task", task.Description)
	}
	if task.Priority != PriorityHigh {
		t.Errorf("Priority = %s, want %s", task.Priority, PriorityHigh)
	}
	if task.State != StatePending {
		t.Errorf("State = %s, want %s", task.State, StatePending)
	}
	if len(task.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(task.Tags))
	}
}

func TestManager_GetNotFound(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	_, err := mgr.Get("nonexistent")
	if err == nil {
		t.Error("Get should fail for nonexistent task")
	}
}

func TestManager_List(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add multiple tasks
	tasks := []AddOptions{
		{TaskID: "task-001", Description: "Task 1", Priority: PriorityLow},
		{TaskID: "task-002", Description: "Task 2", Priority: PriorityHigh},
		{TaskID: "task-003", Description: "Task 3", Priority: PriorityNormal},
	}

	for _, opts := range tasks {
		if err := mgr.Add(opts); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	// List all tasks
	result, err := mgr.List("")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("List returned %d tasks, want 3", len(result))
	}

	// Verify priority ordering (high, normal, low)
	if result[0].Priority != PriorityHigh {
		t.Errorf("First task priority = %s, want %s", result[0].Priority, PriorityHigh)
	}
	if result[1].Priority != PriorityNormal {
		t.Errorf("Second task priority = %s, want %s", result[1].Priority, PriorityNormal)
	}
	if result[2].Priority != PriorityLow {
		t.Errorf("Third task priority = %s, want %s", result[2].Priority, PriorityLow)
	}
}

func TestManager_ListByState(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add tasks
	if err := mgr.Add(AddOptions{TaskID: "task-001", Description: "Task 1"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := mgr.Add(AddOptions{TaskID: "task-002", Description: "Task 2"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Claim one task
	if err := mgr.Claim("task-001", "agent-1"); err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// List pending tasks
	pending, err := mgr.List(StatePending)
	if err != nil {
		t.Fatalf("List pending failed: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("List pending returned %d tasks, want 1", len(pending))
	}

	// List claimed tasks
	claimed, err := mgr.List(StateClaimed)
	if err != nil {
		t.Fatalf("List claimed failed: %v", err)
	}
	if len(claimed) != 1 {
		t.Errorf("List claimed returned %d tasks, want 1", len(claimed))
	}
}

func TestManager_Remove(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add a task
	if err := mgr.Add(AddOptions{TaskID: "task-001", Description: "Test"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Remove it
	if err := mgr.Remove("task-001"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify it's gone
	if mgr.Exists("task-001") {
		t.Error("task still exists after Remove")
	}

	// Remove again should fail
	if err := mgr.Remove("task-001"); err == nil {
		t.Error("Remove should fail for nonexistent task")
	}
}

func TestManager_UpdateState(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add a task
	if err := mgr.Add(AddOptions{TaskID: "task-001", Description: "Test"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Update to claimed
	if err := mgr.UpdateState("task-001", StateClaimed, "agent-1"); err != nil {
		t.Fatalf("UpdateState to claimed failed: %v", err)
	}

	// Verify state changed
	task, err := mgr.Get("task-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if task.State != StateClaimed {
		t.Errorf("State = %s, want %s", task.State, StateClaimed)
	}
	if task.ClaimedBy != "agent-1" {
		t.Errorf("ClaimedBy = %s, want agent-1", task.ClaimedBy)
	}

	// Update to completed
	if err := mgr.UpdateState("task-001", StateCompleted, ""); err != nil {
		t.Fatalf("UpdateState to completed failed: %v", err)
	}

	// Verify state changed
	task, err = mgr.Get("task-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if task.State != StateCompleted {
		t.Errorf("State = %s, want %s", task.State, StateCompleted)
	}
}

func TestManager_Claim(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add a task
	if err := mgr.Add(AddOptions{TaskID: "task-001", Description: "Test"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Claim it
	if err := mgr.Claim("task-001", "agent-1"); err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Verify claimed
	task, err := mgr.Get("task-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if task.State != StateClaimed {
		t.Errorf("State = %s, want %s", task.State, StateClaimed)
	}
	if task.ClaimedBy != "agent-1" {
		t.Errorf("ClaimedBy = %s, want agent-1", task.ClaimedBy)
	}

	// Claim again should fail
	if err := mgr.Claim("task-001", "agent-2"); err == nil {
		t.Error("Claim should fail for already claimed task")
	}
}

func TestManager_ClaimWithDependencies(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add tasks with dependency
	if err := mgr.Add(AddOptions{TaskID: "task-001", Description: "First"}); err != nil {
		t.Fatalf("Add task-001 failed: %v", err)
	}
	if err := mgr.Add(AddOptions{
		TaskID:       "task-002",
		Description:  "Second",
		Dependencies: []string{"task-001"},
	}); err != nil {
		t.Fatalf("Add task-002 failed: %v", err)
	}

	// Try to claim task-002 - should fail (dependency not completed)
	if err := mgr.Claim("task-002", "agent-1"); err == nil {
		t.Error("Claim should fail when dependency not completed")
	}

	// Complete task-001
	if err := mgr.Claim("task-001", "agent-1"); err != nil {
		t.Fatalf("Claim task-001 failed: %v", err)
	}
	if err := mgr.Complete("task-001"); err != nil {
		t.Fatalf("Complete task-001 failed: %v", err)
	}

	// Now claim task-002 should succeed
	if err := mgr.Claim("task-002", "agent-2"); err != nil {
		t.Errorf("Claim task-002 should succeed after dependency completed: %v", err)
	}
}

func TestManager_GetNextAvailable(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add tasks with different priorities
	if err := mgr.Add(AddOptions{TaskID: "task-001", Description: "Low", Priority: PriorityLow}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := mgr.Add(AddOptions{TaskID: "task-002", Description: "High", Priority: PriorityHigh}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := mgr.Add(AddOptions{TaskID: "task-003", Description: "Normal", Priority: PriorityNormal}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Get next available - should be high priority
	task, err := mgr.GetNextAvailable()
	if err != nil {
		t.Fatalf("GetNextAvailable failed: %v", err)
	}
	if task.TaskID != "task-002" {
		t.Errorf("GetNextAvailable returned %s, want task-002", task.TaskID)
	}
}

func TestManager_GetNextAvailableWithDependencies(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add tasks with dependencies
	if err := mgr.Add(AddOptions{TaskID: "task-001", Description: "First"}); err != nil {
		t.Fatalf("Add task-001 failed: %v", err)
	}
	if err := mgr.Add(AddOptions{
		TaskID:       "task-002",
		Description:  "Second",
		Priority:     PriorityHigh,
		Dependencies: []string{"task-001"},
	}); err != nil {
		t.Fatalf("Add task-002 failed: %v", err)
	}

	// Get next available - should be task-001 (task-002 has unmet dependency)
	task, err := mgr.GetNextAvailable()
	if err != nil {
		t.Fatalf("GetNextAvailable failed: %v", err)
	}
	if task.TaskID != "task-001" {
		t.Errorf("GetNextAvailable returned %s, want task-001", task.TaskID)
	}
}

func TestManager_Exists(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Should not exist initially
	if mgr.Exists("task-001") {
		t.Error("task should not exist")
	}

	// Add task
	if err := mgr.Add(AddOptions{TaskID: "task-001", Description: "Test"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Should exist now
	if !mgr.Exists("task-001") {
		t.Error("task should exist")
	}

	// Should exist in different state
	if err := mgr.Claim("task-001", "agent-1"); err != nil {
		t.Fatalf("Claim failed: %v", err)
	}
	if !mgr.Exists("task-001") {
		t.Error("task should still exist after state change")
	}
}

func TestPriorityOrdering(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add tasks in random priority order
	tasks := []AddOptions{
		{TaskID: "task-001", Description: "Normal", Priority: PriorityNormal},
		{TaskID: "task-002", Description: "Low", Priority: PriorityLow},
		{TaskID: "task-003", Description: "High", Priority: PriorityHigh},
		{TaskID: "task-004", Description: "High2", Priority: PriorityHigh},
		{TaskID: "task-005", Description: "Normal2", Priority: PriorityNormal},
	}

	// Add with delays to ensure different timestamps
	for _, opts := range tasks {
		if err := mgr.Add(opts); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// List and verify ordering
	result, err := mgr.List("")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should be ordered: high (earliest first), normal (earliest first), low
	expectedOrder := []string{"task-003", "task-004", "task-001", "task-005", "task-002"}
	for i, taskID := range expectedOrder {
		if result[i].TaskID != taskID {
			t.Errorf("Task at position %d = %s, want %s", i, result[i].TaskID, taskID)
		}
	}
}

func TestManager_CompleteAndFail(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Add and claim a task
	if err := mgr.Add(AddOptions{TaskID: "task-001", Description: "Test"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := mgr.Claim("task-001", "agent-1"); err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Complete it
	if err := mgr.Complete("task-001"); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	task, _ := mgr.Get("task-001")
	if task.State != StateCompleted {
		t.Errorf("State = %s, want %s", task.State, StateCompleted)
	}

	// Add another task to test Fail
	if err := mgr.Add(AddOptions{TaskID: "task-002", Description: "Test 2"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := mgr.Claim("task-002", "agent-2"); err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Fail it
	if err := mgr.Fail("task-002"); err != nil {
		t.Fatalf("Fail failed: %v", err)
	}

	task, _ = mgr.Get("task-002")
	if task.State != StateFailed {
		t.Errorf("State = %s, want %s", task.State, StateFailed)
	}
}
