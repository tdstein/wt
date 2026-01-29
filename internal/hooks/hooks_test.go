package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHooksInSync verifies that embedded templates match source files in .claude/
// This test prevents drift between development hooks and distributed hooks
func TestHooksInSync(t *testing.T) {
	// Find repository root
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	claudeDir := filepath.Join(repoRoot, ".claude")

	// Define files to check
	files := []string{
		"settings.json",
		"hooks/session-start.sh",
		"hooks/session-end.sh",
		"hooks/setup-init.sh",
		"hooks/subagent-start.sh",
		"hooks/subagent-stop.sh",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			// Read source file from .claude/
			sourcePath := filepath.Join(claudeDir, file)
			sourceContent, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("Failed to read source file %s: %v", sourcePath, err)
			}

			// Read embedded template file
			templatePath := filepath.Join("templates", file)
			embeddedContent, err := templatesFS.ReadFile(templatePath)
			if err != nil {
				t.Fatalf("Failed to read embedded file %s: %v", templatePath, err)
			}

			// Compare content
			if string(sourceContent) != string(embeddedContent) {
				t.Errorf("Hook file %s is out of sync!\n\nSource: %s\nEmbedded: %s\n\nRun 'make sync-hooks' to sync templates.",
					file, sourcePath, templatePath)
			}
		})
	}
}

// findRepoRoot walks up the directory tree to find the repository root
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Check if .git exists (file or directory)
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, nil
		}

		// Check if we've reached the root
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
