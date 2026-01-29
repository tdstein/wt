package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tdstein/wt/internal/git"
)

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

	// Create branch name: <agent-name>
	branchName := opts.AgentName

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
