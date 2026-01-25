package conflict

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/posit-dev/wt/internal/git"
)

// Checker handles conflict detection and branch synchronization
type Checker struct {
	targetPath string // Path to the worktree root (e.g., ~/wt/my-project)
}

// NewChecker creates a new conflict checker
func NewChecker(targetPath string) *Checker {
	return &Checker{targetPath: targetPath}
}

// Divergence represents commits ahead/behind between branches
type Divergence struct {
	Ahead  int
	Behind int
}

// ConflictCheckResult contains the results of a conflict check
type ConflictCheckResult struct {
	HasConflicts     bool
	HasChanges       bool
	Divergence       Divergence
	ConflictingFiles []string
}

// CanMergeCleanly checks if a feature branch can merge cleanly into base branch
// Uses git merge-tree to simulate merge without touching working directory
func (c *Checker) CanMergeCleanly(baseBranch, featureBranch string) (bool, error) {
	// Get merge base
	mergeBase, err := git.New("merge-base", baseBranch, featureBranch).
		WithDir(c.targetPath).
		Run()
	if err != nil {
		return false, fmt.Errorf("failed to find merge base: %w", err)
	}

	// Run merge-tree to simulate merge
	mergeResult, err := git.New("merge-tree", mergeBase, baseBranch, featureBranch).
		WithDir(c.targetPath).
		Run()
	if err != nil {
		return false, fmt.Errorf("failed to run merge-tree: %w", err)
	}

	// Check for conflict markers
	hasConflicts := strings.Contains(mergeResult, "\n+<<<<<<< ")
	return !hasConflicts, nil
}

// GetConflictingFiles returns list of files with conflicts between branches
func (c *Checker) GetConflictingFiles(baseBranch, featureBranch string) ([]string, error) {
	// Get merge base
	mergeBase, err := git.New("merge-base", baseBranch, featureBranch).
		WithDir(c.targetPath).
		Run()
	if err != nil {
		return nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	// Run merge-tree
	mergeResult, err := git.New("merge-tree", mergeBase, baseBranch, featureBranch).
		WithDir(c.targetPath).
		Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run merge-tree: %w", err)
	}

	// Parse merge-tree output for file markers
	files := make(map[string]bool)
	lines := strings.Split(mergeResult, "\n")
	for _, line := range lines {
		// Look for diff markers like "+++" or "---"
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			// Extract filename: "+++ b/path/to/file" or "--- a/path/to/file"
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				file := strings.TrimPrefix(parts[1], "b/")
				file = strings.TrimPrefix(file, "a/")
				if file != "/dev/null" {
					files[file] = true
				}
			}
		}
	}

	// Convert map to sorted slice
	var result []string
	for file := range files {
		result = append(result, file)
	}

	return result, nil
}

// GetDivergence returns commits ahead/behind between branches
func (c *Checker) GetDivergence(baseBranch, featureBranch string) (Divergence, error) {
	// Get commits ahead (in feature but not in base)
	aheadOutput, err := git.New("rev-list", "--count", baseBranch+".."+featureBranch).
		WithDir(c.targetPath).
		Run()
	if err != nil {
		return Divergence{}, fmt.Errorf("failed to count ahead commits: %w", err)
	}

	ahead, err := strconv.Atoi(strings.TrimSpace(aheadOutput))
	if err != nil {
		ahead = 0
	}

	// Get commits behind (in base but not in feature)
	behindOutput, err := git.New("rev-list", "--count", featureBranch+".."+baseBranch).
		WithDir(c.targetPath).
		Run()
	if err != nil {
		return Divergence{}, fmt.Errorf("failed to count behind commits: %w", err)
	}

	behind, err := strconv.Atoi(strings.TrimSpace(behindOutput))
	if err != nil {
		behind = 0
	}

	return Divergence{
		Ahead:  ahead,
		Behind: behind,
	}, nil
}

// HasUncommittedChanges checks if a worktree has uncommitted changes
func (c *Checker) HasUncommittedChanges(worktreePath string) (bool, error) {
	// Check for staged or unstaged changes
	err := git.New("diff-index", "--quiet", "HEAD", "--").
		WithDir(worktreePath).
		RunSilent()
	if err != nil {
		// Non-zero exit means there are changes
		return true, nil
	}

	// Check for untracked files
	output, err := git.New("ls-files", "--others", "--exclude-standard").
		WithDir(worktreePath).
		Run()
	if err != nil {
		return false, fmt.Errorf("failed to check untracked files: %w", err)
	}

	hasUntracked := strings.TrimSpace(output) != ""
	return hasUntracked, nil
}

// GetCurrentBranch returns the current branch name for a worktree
func (c *Checker) GetCurrentBranch(worktreePath string) (string, error) {
	branch, err := git.New("rev-parse", "--abbrev-ref", "HEAD").
		WithDir(worktreePath).
		Run()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(branch), nil
}

// SyncOptions contains options for syncing a worktree
type SyncOptions struct {
	AutoRebase bool
}

// SyncResult contains the result of a sync operation
type SyncResult struct {
	AlreadyUpToDate bool
	Divergence      Divergence
	Rebased         bool
	Error           error
}

// Sync synchronizes a worktree with its base branch
func (c *Checker) Sync(agentName, baseBranch string, opts SyncOptions) (*SyncResult, error) {
	worktreePath := filepath.Join(c.targetPath, agentName)

	// Get current branch
	currentBranch, err := c.GetCurrentBranch(worktreePath)
	if err != nil {
		return nil, err
	}

	// Fetch latest from origin
	git.New("fetch", "origin", baseBranch).
		WithDir(c.targetPath).
		RunSilent() // Ignore errors from fetch

	// Get divergence
	divergence, err := c.GetDivergence(baseBranch, currentBranch)
	if err != nil {
		return nil, err
	}

	// Already up to date
	if divergence.Behind == 0 {
		return &SyncResult{
			AlreadyUpToDate: true,
			Divergence:      divergence,
		}, nil
	}

	// Check for uncommitted changes
	hasChanges, err := c.HasUncommittedChanges(worktreePath)
	if err != nil {
		return nil, err
	}

	if hasChanges {
		return &SyncResult{
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

			return &SyncResult{
				Divergence: divergence,
				Error:      fmt.Errorf("rebase failed: %w", err),
			}, nil
		}

		return &SyncResult{
			Divergence: divergence,
			Rebased:    true,
		}, nil
	}

	return &SyncResult{
		Divergence: divergence,
	}, nil
}

// Check performs a comprehensive conflict check for an agent worktree
func (c *Checker) Check(agentName, baseBranch string) (*ConflictCheckResult, error) {
	worktreePath := filepath.Join(c.targetPath, agentName)

	// Get current branch
	featureBranch, err := c.GetCurrentBranch(worktreePath)
	if err != nil {
		return nil, err
	}

	// Check for uncommitted changes
	hasChanges, err := c.HasUncommittedChanges(worktreePath)
	if err != nil {
		return nil, err
	}

	// Get divergence
	divergence, err := c.GetDivergence(baseBranch, featureBranch)
	if err != nil {
		return nil, err
	}

	// Check for merge conflicts
	canMerge, err := c.CanMergeCleanly(baseBranch, featureBranch)
	if err != nil {
		return nil, err
	}

	result := &ConflictCheckResult{
		HasConflicts: !canMerge,
		HasChanges:   hasChanges,
		Divergence:   divergence,
	}

	// Get conflicting files if there are conflicts
	if result.HasConflicts {
		files, err := c.GetConflictingFiles(baseBranch, featureBranch)
		if err == nil {
			result.ConflictingFiles = files
		}
	}

	return result, nil
}

// FormatDivergence returns a human-readable divergence string
func FormatDivergence(div Divergence) string {
	return fmt.Sprintf("%d commits ahead, %d commits behind", div.Ahead, div.Behind)
}

// CheckGitAvailable verifies that git is installed and available
func CheckGitAvailable() error {
	_, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}
	return nil
}
