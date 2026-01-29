package sync

import (
	"fmt"
	"path/filepath"

	"github.com/tdstein/wt/internal/check"
	"github.com/tdstein/wt/internal/git"
)

// Syncer handles branch synchronization
type Syncer struct {
	targetPath string         // Path to the worktree root (e.g., ~/wt/my-project)
	checker    *check.Checker // Checker for conflict detection
}

// NewSyncer creates a new syncer
func NewSyncer(targetPath string) *Syncer {
	return &Syncer{
		targetPath: targetPath,
		checker:    check.NewChecker(targetPath),
	}
}

// Options contains options for syncing a worktree
type Options struct {
	AutoRebase bool
}

// Result contains the result of a sync operation
type Result struct {
	AlreadyUpToDate bool
	Divergence      check.Divergence
	Rebased         bool
	Error           error
}

// Sync synchronizes a worktree with its base branch
func (s *Syncer) Sync(agentName, baseBranch string, opts Options) (*Result, error) {
	worktreePath := filepath.Join(s.targetPath, agentName)

	// Get current branch
	currentBranch, err := s.checker.GetCurrentBranch(worktreePath)
	if err != nil {
		return nil, err
	}

	// Fetch latest from origin
	git.New("fetch", "origin", baseBranch).
		WithDir(s.targetPath).
		RunSilent() // Ignore errors from fetch

	// Get divergence
	divergence, err := s.checker.GetDivergence(baseBranch, currentBranch)
	if err != nil {
		return nil, err
	}

	// Already up to date
	if divergence.Behind == 0 {
		return &Result{
			AlreadyUpToDate: true,
			Divergence:      divergence,
		}, nil
	}

	// Check for uncommitted changes
	hasChanges, err := s.checker.HasUncommittedChanges(worktreePath)
	if err != nil {
		return nil, err
	}

	if hasChanges {
		return &Result{
			Divergence: divergence,
			Error:      fmt.Errorf("uncommitted changes detected, commit or stash before syncing"),
		}, nil
	}

	// Auto-rebase if requested
	if opts.AutoRebase {
		err := git.New("rebase", baseBranch).
			WithDir(worktreePath).
			RunSilent()

		if err != nil {
			// Abort rebase on failure
			git.New("rebase", "--abort").
				WithDir(worktreePath).
				RunSilent()

			return &Result{
				Divergence: divergence,
				Error:      fmt.Errorf("rebase failed: %w", err),
			}, nil
		}

		return &Result{
			Divergence: divergence,
			Rebased:    true,
		}, nil
	}

	return &Result{
		Divergence: divergence,
	}, nil
}
