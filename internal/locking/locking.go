package locking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Lock represents a task lock
type Lock struct {
	TaskID     string    `json:"task_id"`
	AgentName  string    `json:"agent_name"`
	ClaimedAt  time.Time `json:"claimed_at"`
	PID        int       `json:"pid,omitempty"`
	LastActive time.Time `json:"last_active"`
}

// Manager handles task locking operations
type Manager struct {
	targetPath string // Path to the worktree root (e.g., ~/wt/my-project)
}

// NewManager creates a new locking manager
func NewManager(targetPath string) *Manager {
	return &Manager{targetPath: targetPath}
}

// lockDir returns the lock directory path
func (m *Manager) lockDir() string {
	return filepath.Join(m.targetPath, ".bare", "locks")
}

// lockFile returns the lock file path for a task
func (m *Manager) lockFile(taskID string) string {
	return filepath.Join(m.lockDir(), taskID+".json")
}

// Init ensures the lock directory exists
func (m *Manager) Init() error {
	lockDir := m.lockDir()
	return os.MkdirAll(lockDir, 0755)
}

// Claim attempts to claim a lock for a task
func (m *Manager) Claim(taskID, agentName string, pid int) error {
	if taskID == "" {
		return fmt.Errorf("task ID is required")
	}
	if agentName == "" {
		return fmt.Errorf("agent name is required")
	}

	if err := m.Init(); err != nil {
		return fmt.Errorf("failed to initialize lock directory: %w", err)
	}

	lockFile := m.lockFile(taskID)

	// Check if lock already exists
	if _, err := os.Stat(lockFile); err == nil {
		// Lock exists - check if it's stale
		existingLock, err := m.Get(taskID)
		if err != nil {
			return fmt.Errorf("failed to read existing lock: %w", err)
		}

		// If claimed by same agent, update the lock
		if existingLock.AgentName == agentName {
			return m.Touch(taskID)
		}

		return fmt.Errorf("task %s is already claimed by %s", taskID, existingLock.AgentName)
	}

	// Create new lock
	now := time.Now().UTC()
	lock := Lock{
		TaskID:     taskID,
		AgentName:  agentName,
		ClaimedAt:  now,
		PID:        pid,
		LastActive: now,
	}

	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock: %w", err)
	}

	if err := os.WriteFile(lockFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

// Release releases a lock for a task
func (m *Manager) Release(taskID, agentName string) error {
	lock, err := m.Get(taskID)
	if err != nil {
		return err
	}

	// Verify the agent releasing the lock is the one that claimed it
	if lock.AgentName != agentName {
		return fmt.Errorf("task %s is claimed by %s, cannot be released by %s", taskID, lock.AgentName, agentName)
	}

	lockFile := m.lockFile(taskID)
	if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	return nil
}

// ForceRelease forcibly releases a lock (for stale locks or admin operations)
func (m *Manager) ForceRelease(taskID string) error {
	lockFile := m.lockFile(taskID)

	if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	return nil
}

// Get retrieves a lock for a task
func (m *Manager) Get(taskID string) (*Lock, error) {
	lockFile := m.lockFile(taskID)

	data, err := os.ReadFile(lockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("lock not found for task %s", taskID)
		}
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lock Lock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse lock: %w", err)
	}

	return &lock, nil
}

// IsLocked checks if a task is currently locked
func (m *Manager) IsLocked(taskID string) bool {
	_, err := m.Get(taskID)
	return err == nil
}

// Touch updates the last active timestamp for a lock
func (m *Manager) Touch(taskID string) error {
	lock, err := m.Get(taskID)
	if err != nil {
		return err
	}

	lock.LastActive = time.Now().UTC()

	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock: %w", err)
	}

	lockFile := m.lockFile(taskID)
	if err := os.WriteFile(lockFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

// ListAll returns all active locks
func (m *Manager) ListAll() ([]*Lock, error) {
	lockDir := m.lockDir()

	entries, err := os.ReadDir(lockDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Lock{}, nil
		}
		return nil, fmt.Errorf("failed to read lock directory: %w", err)
	}

	var locks []*Lock
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		taskID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
		lock, err := m.Get(taskID)
		if err != nil {
			continue // Skip invalid locks
		}

		locks = append(locks, lock)
	}

	return locks, nil
}

// IsStale checks if a lock is stale based on timeout duration
func (m *Manager) IsStale(taskID string, timeout time.Duration) (bool, error) {
	lock, err := m.Get(taskID)
	if err != nil {
		return false, err
	}

	age := time.Since(lock.LastActive)
	return age > timeout, nil
}

// ListStale returns all stale locks based on timeout duration
func (m *Manager) ListStale(timeout time.Duration) ([]*Lock, error) {
	allLocks, err := m.ListAll()
	if err != nil {
		return nil, err
	}

	var staleLocks []*Lock
	for _, lock := range allLocks {
		age := time.Since(lock.LastActive)
		if age > timeout {
			staleLocks = append(staleLocks, lock)
		}
	}

	return staleLocks, nil
}

// CleanStale removes all stale locks based on timeout duration
func (m *Manager) CleanStale(timeout time.Duration) ([]string, error) {
	staleLocks, err := m.ListStale(timeout)
	if err != nil {
		return nil, err
	}

	var removed []string
	for _, lock := range staleLocks {
		if err := m.ForceRelease(lock.TaskID); err == nil {
			removed = append(removed, lock.TaskID)
		}
	}

	return removed, nil
}
