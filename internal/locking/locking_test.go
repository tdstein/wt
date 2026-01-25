package locking

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func setupTestManager(t *testing.T) (*Manager, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "wt-locking-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	return NewManager(tmpDir), tmpDir
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

	// Verify lock directory exists
	lockDir := mgr.lockDir()
	if _, err := os.Stat(lockDir); os.IsNotExist(err) {
		t.Errorf("lock directory not created: %s", lockDir)
	}
}

func TestManager_Claim(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Claim a task
	err := mgr.Claim("task-001", "alice", 12345)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Verify lock file exists
	lockFile := mgr.lockFile("task-001")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Errorf("lock file not created: %s", lockFile)
	}

	// Verify lock content
	lock, err := mgr.Get("task-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if lock.TaskID != "task-001" {
		t.Errorf("TaskID = %s, want task-001", lock.TaskID)
	}
	if lock.AgentName != "alice" {
		t.Errorf("AgentName = %s, want alice", lock.AgentName)
	}
	if lock.PID != 12345 {
		t.Errorf("PID = %d, want 12345", lock.PID)
	}
	if lock.ClaimedAt.IsZero() {
		t.Error("ClaimedAt timestamp is zero")
	}
	if lock.LastActive.IsZero() {
		t.Error("LastActive timestamp is zero")
	}
}

func TestManager_ClaimValidation(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	tests := []struct {
		name      string
		taskID    string
		agentName string
		wantErr   bool
	}{
		{"valid claim", "task-001", "alice", false},
		{"missing task ID", "", "alice", true},
		{"missing agent name", "task-002", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.Claim(tt.taskID, tt.agentName, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("Claim() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_ClaimAlreadyClaimed(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Claim a task
	err := mgr.Claim("task-001", "alice", 12345)
	if err != nil {
		t.Fatalf("First Claim failed: %v", err)
	}

	// Try to claim same task with different agent - should fail
	err = mgr.Claim("task-001", "bob", 67890)
	if err == nil {
		t.Error("Claim by different agent should fail when task is already claimed")
	}

	// Try to claim same task with same agent - should succeed (updates lock)
	err = mgr.Claim("task-001", "alice", 12345)
	if err != nil {
		t.Errorf("Reclaim by same agent should succeed: %v", err)
	}
}

func TestManager_Release(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Claim a task
	err := mgr.Claim("task-001", "alice", 12345)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Release it
	err = mgr.Release("task-001", "alice")
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Verify lock is gone
	if mgr.IsLocked("task-001") {
		t.Error("task should not be locked after release")
	}

	// Release again should fail
	err = mgr.Release("task-001", "alice")
	if err == nil {
		t.Error("Release should fail for non-existent lock")
	}
}

func TestManager_ReleaseWrongAgent(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Claim a task
	err := mgr.Claim("task-001", "alice", 12345)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Try to release with different agent - should fail
	err = mgr.Release("task-001", "bob")
	if err == nil {
		t.Error("Release by different agent should fail")
	}

	// Task should still be locked by alice
	lock, err := mgr.Get("task-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if lock.AgentName != "alice" {
		t.Errorf("Lock should still be held by alice, got %s", lock.AgentName)
	}
}

func TestManager_ForceRelease(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Claim a task
	err := mgr.Claim("task-001", "alice", 12345)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Force release (no agent check)
	err = mgr.ForceRelease("task-001")
	if err != nil {
		t.Fatalf("ForceRelease failed: %v", err)
	}

	// Verify lock is gone
	if mgr.IsLocked("task-001") {
		t.Error("task should not be locked after force release")
	}

	// Force release non-existent should not error
	err = mgr.ForceRelease("nonexistent")
	if err != nil {
		t.Errorf("ForceRelease of nonexistent lock should not error: %v", err)
	}
}

func TestManager_Get(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Claim a task
	err := mgr.Claim("task-001", "alice", 12345)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Get the lock
	lock, err := mgr.Get("task-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if lock.TaskID != "task-001" {
		t.Errorf("TaskID = %s, want task-001", lock.TaskID)
	}
	if lock.AgentName != "alice" {
		t.Errorf("AgentName = %s, want alice", lock.AgentName)
	}
}

func TestManager_GetNotFound(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	_, err := mgr.Get("nonexistent")
	if err == nil {
		t.Error("Get should fail for nonexistent lock")
	}
}

func TestManager_IsLocked(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Should not be locked initially
	if mgr.IsLocked("task-001") {
		t.Error("task should not be locked initially")
	}

	// Claim it
	err := mgr.Claim("task-001", "alice", 12345)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Should be locked now
	if !mgr.IsLocked("task-001") {
		t.Error("task should be locked after claim")
	}

	// Release it
	err = mgr.Release("task-001", "alice")
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Should not be locked anymore
	if mgr.IsLocked("task-001") {
		t.Error("task should not be locked after release")
	}
}

func TestManager_Touch(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Claim a task
	err := mgr.Claim("task-001", "alice", 12345)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Get initial lock
	initialLock, err := mgr.Get("task-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Touch the lock
	err = mgr.Touch("task-001")
	if err != nil {
		t.Fatalf("Touch failed: %v", err)
	}

	// Get updated lock
	updatedLock, err := mgr.Get("task-001")
	if err != nil {
		t.Fatalf("Get failed after Touch: %v", err)
	}

	// LastActive should be updated
	if !updatedLock.LastActive.After(initialLock.LastActive) {
		t.Errorf("LastActive was not updated: initial=%v, updated=%v",
			initialLock.LastActive, updatedLock.LastActive)
	}

	// ClaimedAt should remain unchanged
	if !updatedLock.ClaimedAt.Equal(initialLock.ClaimedAt) {
		t.Errorf("ClaimedAt changed: initial=%v, updated=%v",
			initialLock.ClaimedAt, updatedLock.ClaimedAt)
	}
}

func TestManager_ListAll(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// List should return empty initially
	locks, err := mgr.ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(locks) != 0 {
		t.Errorf("ListAll returned %d locks, want 0", len(locks))
	}

	// Claim multiple tasks
	tasks := []struct {
		taskID string
		agent  string
	}{
		{"task-001", "alice"},
		{"task-002", "bob"},
		{"task-003", "charlie"},
	}

	for _, task := range tasks {
		err := mgr.Claim(task.taskID, task.agent, 0)
		if err != nil {
			t.Fatalf("Claim(%s) failed: %v", task.taskID, err)
		}
	}

	// List should return all locks
	locks, err = mgr.ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(locks) != len(tasks) {
		t.Errorf("ListAll returned %d locks, want %d", len(locks), len(tasks))
	}
}

func TestManager_IsStale(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Claim a task
	err := mgr.Claim("task-001", "alice", 12345)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Should not be stale with 1 hour timeout
	stale, err := mgr.IsStale("task-001", 1*time.Hour)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if stale {
		t.Error("task should not be stale with 1 hour timeout")
	}

	// Should be stale with 1 millisecond timeout
	time.Sleep(2 * time.Millisecond)
	stale, err = mgr.IsStale("task-001", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Error("task should be stale with 1ms timeout")
	}
}

func TestManager_ListStale(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Claim multiple tasks
	for i := 1; i <= 3; i++ {
		taskID := fmt.Sprintf("task-%03d", i)
		err := mgr.Claim(taskID, "agent", 0)
		if err != nil {
			t.Fatalf("Claim failed: %v", err)
		}
	}

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// All should be stale with 5ms timeout
	staleLocks, err := mgr.ListStale(5 * time.Millisecond)
	if err != nil {
		t.Fatalf("ListStale failed: %v", err)
	}

	if len(staleLocks) != 3 {
		t.Errorf("ListStale returned %d locks, want 3", len(staleLocks))
	}

	// None should be stale with 1 hour timeout
	staleLocks, err = mgr.ListStale(1 * time.Hour)
	if err != nil {
		t.Fatalf("ListStale failed: %v", err)
	}

	if len(staleLocks) != 0 {
		t.Errorf("ListStale returned %d locks, want 0", len(staleLocks))
	}
}

func TestManager_CleanStale(t *testing.T) {
	mgr, tmpDir := setupTestManager(t)
	defer cleanup(tmpDir)

	// Claim multiple tasks
	for i := 1; i <= 3; i++ {
		taskID := fmt.Sprintf("task-%03d", i)
		err := mgr.Claim(taskID, "agent", 0)
		if err != nil {
			t.Fatalf("Claim failed: %v", err)
		}
	}

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Clean stale locks with 5ms timeout
	removed, err := mgr.CleanStale(5 * time.Millisecond)
	if err != nil {
		t.Fatalf("CleanStale failed: %v", err)
	}

	if len(removed) != 3 {
		t.Errorf("CleanStale removed %d locks, want 3", len(removed))
	}

	// Verify all locks are gone
	locks, err := mgr.ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(locks) != 0 {
		t.Errorf("ListAll returned %d locks after cleanup, want 0", len(locks))
	}
}
