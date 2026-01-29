package worktree

import (
	"fmt"
	"os"
	"path/filepath"
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
