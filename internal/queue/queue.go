package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Priority represents task priority levels
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

// State represents task states
type State string

const (
	StatePending   State = "pending"
	StateClaimed   State = "claimed"
	StateCompleted State = "completed"
	StateFailed    State = "failed"
)

// Task represents a task in the queue
type Task struct {
	TaskID       string    `json:"task_id"`
	Description  string    `json:"description"`
	Priority     Priority  `json:"priority"`
	State        State     `json:"state"`
	Dependencies []string  `json:"dependencies"`
	MergeAfter   []string  `json:"merge_after,omitempty"`
	Created      time.Time `json:"created"`
	ClaimedBy    string    `json:"claimed_by,omitempty"`
	ClaimedAt    time.Time `json:"claimed_at,omitempty"`
	BaseBranch   string    `json:"base_branch"`
	Tags         []string  `json:"tags,omitempty"`
}

// Manager handles task queue operations
type Manager struct {
	targetPath string // Path to the worktree root (e.g., ~/wt/my-project)
}

// NewManager creates a new queue manager
func NewManager(targetPath string) *Manager {
	return &Manager{targetPath: targetPath}
}

// queueDir returns the queue directory path
func (m *Manager) queueDir() string {
	return filepath.Join(m.targetPath, ".wt", "queue")
}

// stateDir returns the directory path for a specific state
func (m *Manager) stateDir(state State) string {
	return filepath.Join(m.queueDir(), string(state))
}

// taskFile returns the task file path for a task in a specific state
func (m *Manager) taskFile(taskID string, state State) string {
	return filepath.Join(m.stateDir(state), taskID+".json")
}

// Init ensures the queue directories exist
func (m *Manager) Init() error {
	states := []State{StatePending, StateClaimed, StateCompleted, StateFailed}
	for _, state := range states {
		dir := m.stateDir(state)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// Add adds a new task to the queue
func (m *Manager) Add(opts AddOptions) error {
	// Validate inputs
	if opts.TaskID == "" {
		return fmt.Errorf("task ID is required")
	}
	if opts.Priority == "" {
		opts.Priority = PriorityNormal
	}
	if opts.BaseBranch == "" {
		opts.BaseBranch = "main"
	}

	// Validate priority
	if opts.Priority != PriorityHigh && opts.Priority != PriorityNormal && opts.Priority != PriorityLow {
		return fmt.Errorf("invalid priority: %s (must be high, normal, or low)", opts.Priority)
	}

	// Initialize queue directories
	if err := m.Init(); err != nil {
		return err
	}

	// Check if task already exists in any state
	if m.Exists(opts.TaskID) {
		return fmt.Errorf("task already exists: %s", opts.TaskID)
	}

	// Create task
	task := Task{
		TaskID:       opts.TaskID,
		Description:  opts.Description,
		Priority:     opts.Priority,
		State:        StatePending,
		Dependencies: opts.Dependencies,
		MergeAfter:   opts.MergeAfter,
		Created:      time.Now().UTC(),
		BaseBranch:   opts.BaseBranch,
		Tags:         opts.Tags,
	}

	// Write task file
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	taskFile := m.taskFile(opts.TaskID, StatePending)
	if err := os.WriteFile(taskFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}

// AddOptions contains options for adding a task
type AddOptions struct {
	TaskID       string
	Description  string
	Priority     Priority
	Dependencies []string
	MergeAfter   []string
	BaseBranch   string
	Tags         []string
}

// Get retrieves a task by ID
func (m *Manager) Get(taskID string) (*Task, error) {
	// Check all states
	states := []State{StatePending, StateClaimed, StateCompleted, StateFailed}
	for _, state := range states {
		taskFile := m.taskFile(taskID, state)
		data, err := os.ReadFile(taskFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read task file: %w", err)
		}

		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, fmt.Errorf("failed to parse task: %w", err)
		}

		return &task, nil
	}

	return nil, fmt.Errorf("task not found: %s", taskID)
}

// List returns all tasks, optionally filtered by state
func (m *Manager) List(state State) ([]Task, error) {
	var tasks []Task
	var statesToCheck []State

	if state == "" {
		// List all states
		statesToCheck = []State{StatePending, StateClaimed, StateCompleted, StateFailed}
	} else {
		statesToCheck = []State{state}
	}

	for _, s := range statesToCheck {
		dir := m.stateDir(s)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			taskID := strings.TrimSuffix(entry.Name(), ".json")
			task, err := m.Get(taskID)
			if err != nil {
				continue
			}

			tasks = append(tasks, *task)
		}
	}

	// Sort by priority (high -> normal -> low) then by created time
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			// Priority order: high < normal < low (for sorting)
			priorityOrder := map[Priority]int{
				PriorityHigh:   0,
				PriorityNormal: 1,
				PriorityLow:    2,
			}
			return priorityOrder[tasks[i].Priority] < priorityOrder[tasks[j].Priority]
		}
		return tasks[i].Created.Before(tasks[j].Created)
	})

	return tasks, nil
}

// Remove removes a task from the queue
func (m *Manager) Remove(taskID string) error {
	// Find and remove from any state
	states := []State{StatePending, StateClaimed, StateCompleted, StateFailed}
	found := false

	for _, state := range states {
		taskFile := m.taskFile(taskID, state)
		if _, err := os.Stat(taskFile); err == nil {
			if err := os.Remove(taskFile); err != nil {
				return fmt.Errorf("failed to remove task file: %w", err)
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("task not found: %s", taskID)
	}

	return nil
}

// UpdateState moves a task to a new state
func (m *Manager) UpdateState(taskID string, newState State, claimedBy string) error {
	// Get current task
	task, err := m.Get(taskID)
	if err != nil {
		return err
	}

	// Validate state transition
	if task.State == newState {
		return fmt.Errorf("task is already in state: %s", newState)
	}

	// Update task fields
	oldState := task.State
	task.State = newState

	if newState == StateClaimed {
		if claimedBy == "" {
			return fmt.Errorf("claimed_by is required when claiming a task")
		}
		task.ClaimedBy = claimedBy
		task.ClaimedAt = time.Now().UTC()
	}

	// Remove from old state directory
	oldFile := m.taskFile(taskID, oldState)
	if err := os.Remove(oldFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old task file: %w", err)
	}

	// Write to new state directory
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Ensure new state directory exists
	if err := os.MkdirAll(m.stateDir(newState), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	newFile := m.taskFile(taskID, newState)
	if err := os.WriteFile(newFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}

// Exists checks if a task exists in any state
func (m *Manager) Exists(taskID string) bool {
	_, err := m.Get(taskID)
	return err == nil
}

// Claim claims a task for an agent
func (m *Manager) Claim(taskID, agentName string) error {
	task, err := m.Get(taskID)
	if err != nil {
		return err
	}

	if task.State != StatePending {
		return fmt.Errorf("task is not in pending state (current: %s)", task.State)
	}

	// Check dependencies
	if len(task.Dependencies) > 0 {
		for _, depID := range task.Dependencies {
			dep, err := m.Get(depID)
			if err != nil {
				return fmt.Errorf("dependency %s not found", depID)
			}
			if dep.State != StateCompleted {
				return fmt.Errorf("dependency %s is not completed (current: %s)", depID, dep.State)
			}
		}
	}

	return m.UpdateState(taskID, StateClaimed, agentName)
}

// Complete marks a task as completed
func (m *Manager) Complete(taskID string) error {
	return m.UpdateState(taskID, StateCompleted, "")
}

// Fail marks a task as failed
func (m *Manager) Fail(taskID string) error {
	return m.UpdateState(taskID, StateFailed, "")
}

// GetNextAvailable returns the next available task (highest priority, no dependencies)
func (m *Manager) GetNextAvailable() (*Task, error) {
	tasks, err := m.List(StatePending)
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		// Check if dependencies are satisfied
		allDepsComplete := true
		for _, depID := range task.Dependencies {
			dep, err := m.Get(depID)
			if err != nil || dep.State != StateCompleted {
				allDepsComplete = false
				break
			}
		}

		if allDepsComplete {
			return &task, nil
		}
	}

	return nil, fmt.Errorf("no available tasks")
}
