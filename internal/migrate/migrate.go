package migrate

import (
	"fmt"
	"os"
	"path/filepath"
)

// MigrateStateFromBareToWt migrates state files from .bare/ to .wt/
// This enables seamless upgrades from the old directory structure to the new one.
func MigrateStateFromBareToWt(targetPath string) error {
	migrations := []struct {
		name   string
		oldDir string
		newDir string
	}{
		{
			name:   "metadata",
			oldDir: filepath.Join(targetPath, ".bare", "worktree-metadata"),
			newDir: filepath.Join(targetPath, ".wt", "metadata"),
		},
		{
			name:   "queue",
			oldDir: filepath.Join(targetPath, ".bare", "task-queue"),
			newDir: filepath.Join(targetPath, ".wt", "queue"),
		},
		{
			name:   "locks",
			oldDir: filepath.Join(targetPath, ".bare", "locks"),
			newDir: filepath.Join(targetPath, ".wt", "locks"),
		},
	}

	for _, m := range migrations {
		if err := migrateDirectory(m.oldDir, m.newDir); err != nil {
			return fmt.Errorf("failed to migrate %s: %w", m.name, err)
		}
	}

	return nil
}

// migrateDirectory moves contents from oldDir to newDir.
// If oldDir doesn't exist, it's a no-op (already migrated or fresh installation).
func migrateDirectory(oldDir, newDir string) error {
	// Check if old directory exists
	_, err := os.Stat(oldDir)
	if os.IsNotExist(err) {
		// No migration needed - fresh installation or already migrated
		return nil
	}
	if err != nil {
		return err
	}

	// Create new directory if it doesn't exist
	if err := os.MkdirAll(newDir, 0755); err != nil {
		return err
	}

	// Read old directory contents
	entries, err := os.ReadDir(oldDir)
	if err != nil {
		return err
	}

	// Move each file/directory to new location
	for _, entry := range entries {
		oldPath := filepath.Join(oldDir, entry.Name())
		newPath := filepath.Join(newDir, entry.Name())

		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to move %s: %w", entry.Name(), err)
		}
	}

	// Remove old directory (should be empty now)
	if err := os.Remove(oldDir); err != nil {
		return fmt.Errorf("failed to remove old directory: %w", err)
	}

	return nil
}

// IsMigrationNeeded checks if state files exist in old .bare/ location.
// Returns true if migration is needed, false if already migrated or fresh installation.
func IsMigrationNeeded(targetPath string) bool {
	oldMetadataPath := filepath.Join(targetPath, ".bare", "worktree-metadata")
	_, err := os.Stat(oldMetadataPath)
	return err == nil
}
