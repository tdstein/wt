package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tdstein/wt/internal/check"
)

// Check performs a merge conflict check for an agent
func (m *Manager) Check(agentName string) (*check.Result, error) {
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
	return m.checker.Check(agentName, baseBranch)
}
