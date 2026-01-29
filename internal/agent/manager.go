package agent

import (
	"fmt"
	"regexp"

	"github.com/tdstein/wt/internal/check"
	"github.com/tdstein/wt/internal/metadata"
)

// Manager handles agent worktree operations
type Manager struct {
	targetPath string                  // Path to the worktree root
	metadata   *metadata.MetadataManager  // Metadata manager
	checker    *check.Checker           // Conflict checker
}

// NewManager creates a new agent manager
func NewManager(targetPath string) *Manager {
	return &Manager{
		targetPath: targetPath,
		metadata:   metadata.NewMetadataManager(targetPath),
		checker:    check.NewChecker(targetPath),
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
