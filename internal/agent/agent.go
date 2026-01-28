package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tdstein/wt/internal/conflict"
	"github.com/tdstein/wt/internal/git"
)

// Manager handles agent worktree operations
type Manager struct {
	targetPath string            // Path to the worktree root
	metadata   *MetadataManager  // Metadata manager
	conflict   *conflict.Checker // Conflict checker
}

// NewAgentManager creates a new agent manager
func NewAgentManager(targetPath string) *Manager {
	return &Manager{
		targetPath: targetPath,
		metadata:   NewMetadataManager(targetPath),
		conflict:   conflict.NewChecker(targetPath),
	}
}

// ValidateAgentName checks if an agent name is valid (alphanumeric, hyphens, underscores)
func ValidateAgentName(name string) error {
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("agent name must be alphanumeric with hyphens/underscores only")
	}
	return nil
}

// CreateOptions contains options for creating an agent worktree
type CreateOptions struct {
	AgentName  string
	BaseBranch string
}

// Create creates a new agent worktree
func (m *Manager) Create(opts CreateOptions) error {
	// Validate inputs
	if opts.AgentName == "" {
		return fmt.Errorf("agent name is required")
	}
	if opts.BaseBranch == "" {
		opts.BaseBranch = "main"
	}

	// Validate agent name
	if err := ValidateAgentName(opts.AgentName); err != nil {
		return err
	}

	worktreePath := filepath.Join(m.targetPath, opts.AgentName)

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree already exists: %s", worktreePath)
	}

	// Create branch name: work/<agent-name>
	branchName := fmt.Sprintf("work/%s", opts.AgentName)

	// Check if branch already exists
	_, err := git.New("rev-parse", "--verify", branchName).
		WithDir(m.targetPath).
		Run()
	if err == nil {
		return fmt.Errorf("branch already exists: %s", branchName)
	}

	// Create worktree
	err = git.WorktreeAdd(m.targetPath, opts.AgentName, branchName, true)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Set base branch in worktree config
	git.New("branch", "--set-upstream-to", opts.BaseBranch).
		WithDir(worktreePath).
		RunSilent() // Ignore errors

	// Create metadata
	err = m.metadata.Create(opts.AgentName, branchName, opts.BaseBranch)
	if err != nil {
		// Clean up worktree if metadata creation fails
		git.New("worktree", "remove", opts.AgentName).
			WithDir(m.targetPath).
			RunSilent()
		return fmt.Errorf("failed to create metadata: %w", err)
	}

	return nil
}

// RemoveOptions contains options for removing an agent worktree
type RemoveOptions struct {
	AgentName    string
	DeleteBranch bool
}

// Remove removes an agent worktree
func (m *Manager) Remove(opts RemoveOptions) error {
	if opts.AgentName == "" {
		return fmt.Errorf("agent name is required")
	}

	worktreePath := filepath.Join(m.targetPath, opts.AgentName)

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree does not exist: %s", worktreePath)
	}

	// Get branch name from metadata if available
	var branchName string
	if m.metadata.Exists(opts.AgentName) {
		metadata, err := m.metadata.Get(opts.AgentName)
		if err == nil {
			branchName = metadata.Branch
		}
	} else {
		// Fallback: get current branch from worktree
		branch, _ := m.conflict.GetCurrentBranch(worktreePath)
		branchName = branch
	}

	// Remove worktree
	err := git.New("worktree", "remove", opts.AgentName).
		WithDir(m.targetPath).
		RunSilent()
	if err != nil {
		return fmt.Errorf("failed to remove worktree (try: git worktree remove --force %s): %w",
			opts.AgentName, err)
	}

	// Remove metadata
	m.metadata.Remove(opts.AgentName)

	// Delete branch if requested and branch name is known
	if opts.DeleteBranch && branchName != "" {
		// Check if branch is merged
		mergedBranches, err := git.New("branch", "--merged", "main").
			WithDir(m.targetPath).
			Run()

		isMerged := err == nil && strings.Contains(mergedBranches, branchName)

		if isMerged {
			// Delete merged branch
			err = git.New("branch", "-d", branchName).
				WithDir(m.targetPath).
				RunSilent()
			if err != nil {
				return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
			}
		} else {
			return fmt.Errorf("branch %s is not merged into main (use: git branch -D %s for force delete)",
				branchName, branchName)
		}
	}

	return nil
}

// AgentInfo contains information about an agent worktree
type AgentInfo struct {
	Agent    string
	Branch   string
	Age      int64 // Age in seconds
	AgeHuman string
	Status   string
	Exists   bool
}

// List lists all agent worktrees
func (m *Manager) List() ([]AgentInfo, error) {
	metadataFiles, err := m.metadata.List()
	if err != nil {
		return nil, err
	}

	if len(metadataFiles) == 0 {
		return []AgentInfo{}, nil
	}

	var agents []AgentInfo
	for _, metadataFile := range metadataFiles {
		agent := strings.TrimSuffix(filepath.Base(metadataFile), ".json")

		metadata, err := m.metadata.Get(agent)
		if err != nil {
			continue
		}

		age, _ := m.metadata.Age(agent)
		ageHuman := AgeHuman(age)

		worktreePath := filepath.Join(m.targetPath, agent)
		exists := false
		if _, err := os.Stat(worktreePath); err == nil {
			exists = true
		}

		status := metadata.Status
		if !exists {
			status = "missing"
		}

		agents = append(agents, AgentInfo{
			Agent:    agent,
			Branch:   metadata.Branch,
			Age:      age,
			AgeHuman: ageHuman,
			Status:   status,
			Exists:   exists,
		})
	}

	return agents, nil
}

// Check performs a merge conflict check for an agent
func (m *Manager) Check(agentName string) (*conflict.ConflictCheckResult, error) {
	if agentName == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	worktreePath := filepath.Join(m.targetPath, agentName)

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("worktree does not exist: %s", worktreePath)
	}

	// Get base branch from metadata
	baseBranch := "main"
	if m.metadata.Exists(agentName) {
		metadata, err := m.metadata.Get(agentName)
		if err == nil {
			baseBranch = metadata.BaseBranch
		}
	}

	// Update last activity
	m.metadata.Touch(agentName)

	// Run conflict check
	return m.conflict.Check(agentName, baseBranch)
}

// SyncOptions contains options for syncing an agent
type SyncOptions struct {
	AgentName  string
	AutoRebase bool
}

// Sync synchronizes an agent worktree with its base branch
func (m *Manager) Sync(opts SyncOptions) (*conflict.SyncResult, error) {
	if opts.AgentName == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	worktreePath := filepath.Join(m.targetPath, opts.AgentName)

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("worktree does not exist: %s", worktreePath)
	}

	// Get base branch from metadata
	baseBranch := "main"
	if m.metadata.Exists(opts.AgentName) {
		metadata, err := m.metadata.Get(opts.AgentName)
		if err == nil {
			baseBranch = metadata.BaseBranch
		}
	}

	// Update last activity
	m.metadata.Touch(opts.AgentName)

	// Sync
	return m.conflict.Sync(opts.AgentName, baseBranch, conflict.SyncOptions{
		AutoRebase: opts.AutoRebase,
	})
}

// PruneOptions contains options for pruning stale worktrees
type PruneOptions struct {
	OlderThanDays int
	DryRun        bool
	Interactive   bool
}

// PruneResult contains the result of a prune operation
type PruneResult struct {
	StaleAgents []string
	Removed     []string
	Errors      map[string]error
}

// Prune removes stale agent worktrees
func (m *Manager) Prune(opts PruneOptions) (*PruneResult, error) {
	if opts.OlderThanDays == 0 {
		opts.OlderThanDays = 7
	}

	olderThanSeconds := int64(opts.OlderThanDays * 86400)

	result := &PruneResult{
		StaleAgents: []string{},
		Removed:     []string{},
		Errors:      make(map[string]error),
	}

	agents, err := m.List()
	if err != nil {
		return result, err
	}

	for _, agent := range agents {
		if agent.Age > olderThanSeconds {
			result.StaleAgents = append(result.StaleAgents, agent.Agent)

			if opts.DryRun {
				continue
			}

			shouldRemove := true
			if opts.Interactive {
				shouldRemove = promptYesNo(fmt.Sprintf("Remove %s (%s old)?", agent.Agent, agent.AgeHuman))
			}

			if shouldRemove {
				err := m.Remove(RemoveOptions{AgentName: agent.Agent})
				if err != nil {
					result.Errors[agent.Agent] = err
				} else {
					result.Removed = append(result.Removed, agent.Agent)
				}
			}
		}
	}

	return result, nil
}

// Status returns dashboard information about all agents
type Status struct {
	TotalCount  int
	ActiveCount int
	Agents      []AgentInfo
}

// GetStatus returns status dashboard information
func (m *Manager) GetStatus() (*Status, error) {
	agents, err := m.List()
	if err != nil {
		return nil, err
	}

	activeCount := 0
	for _, agent := range agents {
		if agent.Exists {
			activeCount++
		}
	}

	return &Status{
		TotalCount:  len(agents),
		ActiveCount: activeCount,
		Agents:      agents,
	}, nil
}

// Helper function to prompt for yes/no
func promptYesNo(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// ParseOlderThan parses an --older-than argument (e.g., "7d", "14d")
func ParseOlderThan(arg string) (int, error) {
	arg = strings.TrimSuffix(arg, "d")
	days, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("invalid --older-than value: %s", arg)
	}
	if days < 0 {
		return 0, fmt.Errorf("--older-than must be positive")
	}
	return days, nil
}
