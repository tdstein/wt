package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdstein/wt/internal/git"
)

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
		branch, _ := m.checker.GetCurrentBranch(worktreePath)
		branchName = branch
	}

	// Safety check: ensure no uncommitted changes
	hasChanges, err := m.checker.HasUncommittedChanges(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("worktree has uncommitted changes, commit or stash before removing")
	}

	// Safety check: ensure all commits are merged into main
	if branchName != "" {
		baseBranch := "main"
		if m.metadata.Exists(opts.AgentName) {
			metadata, err := m.metadata.Get(opts.AgentName)
			if err == nil && metadata.BaseBranch != "" {
				baseBranch = metadata.BaseBranch
			}
		}

		divergence, err := m.checker.GetDivergence(baseBranch, branchName)
		if err != nil {
			return fmt.Errorf("failed to check branch divergence: %w", err)
		}

		if divergence.Ahead > 0 {
			return fmt.Errorf("branch %s has %d unmerged commit(s), merge into %s before removing",
				branchName, divergence.Ahead, baseBranch)
		}
	}

	// Remove worktree
	err = git.New("worktree", "remove", opts.AgentName).
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
