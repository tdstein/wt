package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsMigrationNeeded_NoOldState(t *testing.T) {
	tmpDir := t.TempDir()

	// Create bare directory but no old state
	bareDir := filepath.Join(tmpDir, ".bare")
	if err := os.MkdirAll(bareDir, 0755); err != nil {
		t.Fatalf("failed to create bare dir: %v", err)
	}

	if IsMigrationNeeded(tmpDir) {
		t.Error("IsMigrationNeeded() returned true for fresh installation")
	}
}

func TestIsMigrationNeeded_WithOldState(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old state directory
	oldMetadataDir := filepath.Join(tmpDir, ".bare", "worktree-metadata")
	if err := os.MkdirAll(oldMetadataDir, 0755); err != nil {
		t.Fatalf("failed to create old metadata dir: %v", err)
	}

	if !IsMigrationNeeded(tmpDir) {
		t.Error("IsMigrationNeeded() returned false with old state")
	}
}

func TestMigrateDirectory_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old directory with files
	oldDir := filepath.Join(tmpDir, "old")
	newDir := filepath.Join(tmpDir, "new")

	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatalf("failed to create old dir: %v", err)
	}

	// Create test files
	testFiles := []string{"file1.json", "file2.json", "subdir/file3.json"}
	for _, file := range testFiles {
		path := filepath.Join(oldDir, file)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	// Run migration
	if err := migrateDirectory(oldDir, newDir); err != nil {
		t.Fatalf("migrateDirectory() failed: %v", err)
	}

	// Verify files were moved
	for _, file := range testFiles {
		newPath := filepath.Join(newDir, file)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			t.Errorf("file not migrated: %s", newPath)
		}

		oldPath := filepath.Join(oldDir, file)
		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Errorf("old file still exists: %s", oldPath)
		}
	}

	// Verify old directory is removed
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("old directory not removed")
	}
}

func TestMigrateDirectory_NoOldDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to migrate non-existent directory (should be no-op)
	oldDir := filepath.Join(tmpDir, "nonexistent")
	newDir := filepath.Join(tmpDir, "new")

	if err := migrateDirectory(oldDir, newDir); err != nil {
		t.Fatalf("migrateDirectory() failed for non-existent old dir: %v", err)
	}

	// New directory should not be created if old doesn't exist
	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		t.Error("new directory should not be created when old directory doesn't exist")
	}
}

func TestMigrateDirectory_ExistingNewDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old and new directories
	oldDir := filepath.Join(tmpDir, "old")
	newDir := filepath.Join(tmpDir, "new")

	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatalf("failed to create old dir: %v", err)
	}
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatalf("failed to create new dir: %v", err)
	}

	// Create test files
	if err := os.WriteFile(filepath.Join(oldDir, "old.json"), []byte("old"), 0644); err != nil {
		t.Fatalf("failed to write old file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "existing.json"), []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	// Run migration (should succeed, moving into existing directory)
	if err := migrateDirectory(oldDir, newDir); err != nil {
		t.Fatalf("migrateDirectory() failed: %v", err)
	}

	// Verify files are in new directory
	if _, err := os.Stat(filepath.Join(newDir, "old.json")); os.IsNotExist(err) {
		t.Error("old file not migrated")
	}
	if _, err := os.Stat(filepath.Join(newDir, "existing.json")); os.IsNotExist(err) {
		t.Error("existing file lost")
	}
}

func TestMigrateStateFromBareToWt_Complete(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old directory structure
	oldDirs := []string{
		filepath.Join(tmpDir, ".bare", "worktree-metadata"),
		filepath.Join(tmpDir, ".bare", "task-queue", "pending"),
		filepath.Join(tmpDir, ".bare", "locks"),
	}

	for _, dir := range oldDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	// Create test files
	testFiles := map[string]string{
		filepath.Join(tmpDir, ".bare", "worktree-metadata", "alice.json"): "{}",
		filepath.Join(tmpDir, ".bare", "task-queue", "pending", "task1"):   "{}",
		filepath.Join(tmpDir, ".bare", "locks", "task1.json"):              "{}",
	}

	for path, content := range testFiles {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	// Create .wt directory structure
	newDirs := []string{
		filepath.Join(tmpDir, ".wt", "metadata"),
		filepath.Join(tmpDir, ".wt", "queue"),
		filepath.Join(tmpDir, ".wt", "locks"),
	}

	for _, dir := range newDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	// Run migration
	if err := MigrateStateFromBareToWt(tmpDir); err != nil {
		t.Fatalf("MigrateStateFromBareToWt() failed: %v", err)
	}

	// Verify new locations
	newFiles := []string{
		filepath.Join(tmpDir, ".wt", "metadata", "alice.json"),
		filepath.Join(tmpDir, ".wt", "queue", "pending", "task1"),
		filepath.Join(tmpDir, ".wt", "locks", "task1.json"),
	}

	for _, path := range newFiles {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file not migrated: %s", path)
		}
	}

	// Verify old directories are removed
	for _, dir := range oldDirs {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("old directory still exists: %s", dir)
		}
	}
}

func TestMigrateStateFromBareToWt_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .wt directory structure (already migrated)
	newDirs := []string{
		filepath.Join(tmpDir, ".wt", "metadata"),
		filepath.Join(tmpDir, ".wt", "queue"),
		filepath.Join(tmpDir, ".wt", "locks"),
	}

	for _, dir := range newDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	// Create test files in new location
	if err := os.WriteFile(filepath.Join(tmpDir, ".wt", "metadata", "bob.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Run migration (should be no-op for already-migrated state)
	if err := MigrateStateFromBareToWt(tmpDir); err != nil {
		t.Fatalf("MigrateStateFromBareToWt() failed: %v", err)
	}

	// Verify file is still there
	if _, err := os.Stat(filepath.Join(tmpDir, ".wt", "metadata", "bob.json")); os.IsNotExist(err) {
		t.Error("file lost during migration")
	}
}
