package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindRoot searches for .bare directory in current or parent directories
func FindRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for currentDir != "/" {
		barePath := filepath.Join(currentDir, ".bare")
		if info, err := os.Stat(barePath); err == nil && info.IsDir() {
			return currentDir, nil
		}
		currentDir = filepath.Dir(currentDir)
	}

	return "", fmt.Errorf("not in a wt directory (no .bare/ found in parent directories)")
}

// GetCurrentWorktree returns the name of the current worktree based on the git branch
func GetCurrentWorktree() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branchName := strings.TrimSpace(string(output))
	if branchName == "" {
		return "", fmt.Errorf("failed to determine current branch")
	}

	return branchName, nil
}
