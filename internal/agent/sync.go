package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tdstein/wt/internal/sync"
)

// SyncOptions contains options for syncing an agent
type SyncOptions struct {
	AgentName  string
	AutoRebase bool
}

// Sync synchronizes an agent worktree with its base branch
func (m *Manager) Sync(opts SyncOptions) (*sync.Result, error) {
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

	// Create syncer and sync
	syncer := sync.NewSyncer(m.targetPath)
	return syncer.Sync(opts.AgentName, baseBranch, sync.Options{
		AutoRebase: opts.AutoRebase,
	})
}
