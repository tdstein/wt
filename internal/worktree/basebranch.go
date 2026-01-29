package worktree

import (
	"fmt"
	"strings"

	"github.com/tdstein/wt/internal/git"
)

// DetectBaseBranch determines the base branch with the following priority:
// 1. Explicit argument (if provided)
// 2. Current branch (if in a git repository and not detached HEAD)
// 3. Remote HEAD (for repositories with remotes)
// 4. "main" (fallback)
func DetectBaseBranch(targetPath string, explicitBranch string) (string, error) {
	// Priority 1: Explicit argument
	if explicitBranch != "" {
		return explicitBranch, nil
	}

	// Priority 2: Current branch
	currentBranch, err := GetCurrentBranch(targetPath)
	if err == nil && currentBranch != "" {
		return currentBranch, nil
	}

	// Priority 3: Remote HEAD
	remoteHead, err := GetRemoteHead(targetPath)
	if err == nil && remoteHead != "" {
		return remoteHead, nil
	}

	// Priority 4: Fallback to "main"
	return "main", nil
}

// GetCurrentBranch returns the current branch name, or empty string if detached HEAD
func GetCurrentBranch(targetPath string) (string, error) {
	output, err := git.New("branch", "--show-current").
		WithDir(targetPath).
		Run()
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(output)
	// Empty string indicates detached HEAD
	if branch == "" {
		return "", fmt.Errorf("detached HEAD")
	}

	return branch, nil
}

// GetRemoteHead queries the remote repository to determine the default branch
func GetRemoteHead(targetPath string) (string, error) {
	// Try to get remote HEAD
	output, err := git.New("symbolic-ref", "refs/remotes/origin/HEAD", "--short").
		WithDir(targetPath).
		Run()
	if err == nil && output != "" {
		// Output format: "origin/main" -> extract "main"
		branch := strings.TrimSpace(output)
		parts := strings.Split(branch, "/")
		if len(parts) >= 2 {
			return parts[len(parts)-1], nil
		}
		return branch, nil
	}

	// Fallback: Query remote directly
	output, err = git.New("remote", "show", "origin").
		WithDir(targetPath).
		Run()
	if err != nil {
		return "", err
	}

	// Parse output for "HEAD branch: <branch>"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "HEAD branch:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				branch := strings.TrimSpace(parts[1])
				return branch, nil
			}
		}
	}

	return "", fmt.Errorf("could not determine remote HEAD")
}
