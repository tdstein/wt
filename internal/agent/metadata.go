package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Checkpoint represents a saved state in the agent's work
type Checkpoint struct {
	Name      string    `json:"name"`
	Commit    string    `json:"commit"`
	Timestamp time.Time `json:"timestamp"`
}

// Metadata represents the JSON metadata for a worktree agent
type Metadata struct {
	Agent        string       `json:"agent"`
	TaskID       string       `json:"task_id"`
	Branch       string       `json:"branch"`
	BaseBranch   string       `json:"base_branch"`
	Created      time.Time    `json:"created"`
	LastActivity time.Time    `json:"last_activity"`
	Status       string       `json:"status"`

	// Enhanced fields (backwards compatible with omitempty)
	Progress     int          `json:"progress,omitempty"`      // 0-100 percentage
	State        string       `json:"state,omitempty"`         // claimed, working, testing, blocked, failed, completed
	ErrorMessage string       `json:"error_message,omitempty"` // Error details if failed
	PID          int          `json:"pid,omitempty"`           // Process ID for health monitoring
	LogFile      string       `json:"log_file,omitempty"`      // Path to agent's work log
	Checkpoints  []Checkpoint `json:"checkpoints,omitempty"`   // Saved states for rollback
}

// MetadataManager handles metadata operations for a worktree
type MetadataManager struct {
	targetPath string // Path to the worktree root (e.g., ~/wt/my-project)
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager(targetPath string) *MetadataManager {
	return &MetadataManager{targetPath: targetPath}
}

// metadataDir returns the metadata directory path
func (m *MetadataManager) metadataDir() string {
	return filepath.Join(m.targetPath, ".bare", "worktree-metadata")
}

// metadataFile returns the metadata file path for an agent
func (m *MetadataManager) metadataFile(agent string) string {
	return filepath.Join(m.metadataDir(), agent+".json")
}

// Init ensures the metadata directory exists
func (m *MetadataManager) Init() error {
	metadataDir := m.metadataDir()
	return os.MkdirAll(metadataDir, 0755)
}

// Create creates metadata for an agent worktree
func (m *MetadataManager) Create(agent, taskID, branch, baseBranch string) error {
	if err := m.Init(); err != nil {
		return fmt.Errorf("failed to initialize metadata directory: %w", err)
	}

	now := time.Now().UTC()
	metadata := Metadata{
		Agent:        agent,
		TaskID:       taskID,
		Branch:       branch,
		BaseBranch:   baseBranch,
		Created:      now,
		LastActivity: now,
		Status:       "active",
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataFile := m.metadataFile(agent)
	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// Touch updates the last activity timestamp for an agent
func (m *MetadataManager) Touch(agent string) error {
	metadata, err := m.Get(agent)
	if err != nil {
		return err
	}

	metadata.LastActivity = time.Now().UTC()

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataFile := m.metadataFile(agent)
	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// Get retrieves metadata for an agent
func (m *MetadataManager) Get(agent string) (*Metadata, error) {
	metadataFile := m.metadataFile(agent)

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("metadata not found for agent %q", agent)
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// GetField retrieves a specific field value from agent metadata
func (m *MetadataManager) GetField(agent, field string) (string, error) {
	metadata, err := m.Get(agent)
	if err != nil {
		return "", err
	}

	switch field {
	case "agent":
		return metadata.Agent, nil
	case "task_id":
		return metadata.TaskID, nil
	case "branch":
		return metadata.Branch, nil
	case "base_branch":
		return metadata.BaseBranch, nil
	case "created":
		return metadata.Created.Format(time.RFC3339), nil
	case "last_activity":
		return metadata.LastActivity.Format(time.RFC3339), nil
	case "status":
		return metadata.Status, nil
	default:
		return "", fmt.Errorf("unknown field: %s", field)
	}
}

// Remove deletes metadata for an agent
func (m *MetadataManager) Remove(agent string) error {
	metadataFile := m.metadataFile(agent)

	if err := os.Remove(metadataFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove metadata file: %w", err)
	}

	return nil
}

// List returns all metadata files sorted by agent name
func (m *MetadataManager) List() ([]string, error) {
	metadataDir := m.metadataDir()

	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			files = append(files, filepath.Join(metadataDir, entry.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}

// Exists checks if metadata exists for an agent
func (m *MetadataManager) Exists(agent string) bool {
	metadataFile := m.metadataFile(agent)
	_, err := os.Stat(metadataFile)
	return err == nil
}

// Age returns the age of a worktree in seconds (since last activity)
func (m *MetadataManager) Age(agent string) (int64, error) {
	metadata, err := m.Get(agent)
	if err != nil {
		return 0, err
	}

	age := time.Since(metadata.LastActivity)
	return int64(age.Seconds()), nil
}

// AgeHuman returns the age formatted in human-readable form
func AgeHuman(ageSeconds int64) string {
	const (
		minute = 60
		hour   = 60 * minute
		day    = 24 * hour
	)

	switch {
	case ageSeconds < minute:
		return fmt.Sprintf("%ds", ageSeconds)
	case ageSeconds < hour:
		minutes := ageSeconds / minute
		return fmt.Sprintf("%dm", minutes)
	case ageSeconds < day:
		hours := ageSeconds / hour
		return fmt.Sprintf("%dh", hours)
	default:
		days := ageSeconds / day
		return fmt.Sprintf("%dd", days)
	}
}

// save is a helper method to save metadata to disk
func (m *MetadataManager) save(metadata *Metadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataFile := m.metadataFile(metadata.Agent)
	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// UpdateProgress updates the progress percentage for an agent
func (m *MetadataManager) UpdateProgress(agent string, progress int) error {
	if progress < 0 || progress > 100 {
		return fmt.Errorf("progress must be between 0 and 100")
	}

	metadata, err := m.Get(agent)
	if err != nil {
		return err
	}

	metadata.Progress = progress
	metadata.LastActivity = time.Now().UTC()

	return m.save(metadata)
}

// UpdateState updates the state for an agent
func (m *MetadataManager) UpdateState(agent, state string) error {
	validStates := map[string]bool{
		"claimed":   true,
		"working":   true,
		"testing":   true,
		"blocked":   true,
		"failed":    true,
		"completed": true,
	}

	if !validStates[state] {
		return fmt.Errorf("invalid state: %s (must be one of: claimed, working, testing, blocked, failed, completed)", state)
	}

	metadata, err := m.Get(agent)
	if err != nil {
		return err
	}

	metadata.State = state
	metadata.LastActivity = time.Now().UTC()

	return m.save(metadata)
}

// SetError sets an error message for an agent
func (m *MetadataManager) SetError(agent, errorMsg string) error {
	metadata, err := m.Get(agent)
	if err != nil {
		return err
	}

	metadata.ErrorMessage = errorMsg
	metadata.State = "failed"
	metadata.LastActivity = time.Now().UTC()

	return m.save(metadata)
}

// SetPID sets the process ID for an agent
func (m *MetadataManager) SetPID(agent string, pid int) error {
	if pid < 0 {
		return fmt.Errorf("PID must be non-negative")
	}

	metadata, err := m.Get(agent)
	if err != nil {
		return err
	}

	metadata.PID = pid
	metadata.LastActivity = time.Now().UTC()

	return m.save(metadata)
}

// AddCheckpoint adds a checkpoint to an agent's metadata
func (m *MetadataManager) AddCheckpoint(agent, name, commit string) error {
	if name == "" {
		return fmt.Errorf("checkpoint name is required")
	}
	if commit == "" {
		return fmt.Errorf("checkpoint commit is required")
	}

	metadata, err := m.Get(agent)
	if err != nil {
		return err
	}

	checkpoint := Checkpoint{
		Name:      name,
		Commit:    commit,
		Timestamp: time.Now().UTC(),
	}

	metadata.Checkpoints = append(metadata.Checkpoints, checkpoint)
	metadata.LastActivity = time.Now().UTC()

	return m.save(metadata)
}

// GetCheckpoints retrieves all checkpoints for an agent
func (m *MetadataManager) GetCheckpoints(agent string) ([]Checkpoint, error) {
	metadata, err := m.Get(agent)
	if err != nil {
		return nil, err
	}

	if metadata.Checkpoints == nil {
		return []Checkpoint{}, nil
	}

	return metadata.Checkpoints, nil
}
