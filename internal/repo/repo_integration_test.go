package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tdstein/wt/internal/agent"
	"github.com/tdstein/wt/internal/locking"
	"github.com/tdstein/wt/internal/migrate"
	"github.com/tdstein/wt/internal/queue"
)

// TestSetupLocal_CreatesWtDirectory verifies that SetupLocal creates .wt subdirectories
func TestSetupLocal_CreatesWtDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewManager(tmpDir, "")

	if err := rm.SetupLocal(); err != nil {
		t.Fatalf("SetupLocal() failed: %v", err)
	}

	// Verify .wt subdirectories exist
	subdirs := []string{"metadata", "queue", "locks"}
	for _, subdir := range subdirs {
		path := filepath.Join(tmpDir, ".wt", subdir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf(".wt/%s directory not created", subdir)
		}
	}
}

// TestSetupLocal_ManagersUseWtPaths verifies managers use .wt/ paths after setup
func TestSetupLocal_ManagersUseWtPaths(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewManager(tmpDir, "")

	if err := rm.SetupLocal(); err != nil {
		t.Fatalf("SetupLocal() failed: %v", err)
	}

	// Test MetadataManager can initialize with .wt/metadata
	mm := agent.NewMetadataManager(tmpDir)
	if err := mm.Init(); err != nil {
		t.Errorf("MetadataManager.Init() failed: %v", err)
	}

	// Test QueueManager can initialize with .wt/queue
	qm := queue.NewManager(tmpDir)
	if err := qm.Init(); err != nil {
		t.Errorf("QueueManager.Init() failed: %v", err)
	}

	// Test LockingManager can initialize with .wt/locks
	lm := locking.NewManager(tmpDir)
	if err := lm.Init(); err != nil {
		t.Errorf("LockingManager.Init() failed: %v", err)
	}

	// Verify .wt subdirectories exist and are accessible
	wtPath := filepath.Join(tmpDir, ".wt")
	for _, subdir := range []string{"metadata", "queue", "locks"} {
		path := filepath.Join(wtPath, subdir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected directory not found: %s", path)
		}
	}
}

// TestMigration_FromBareToWt verifies migration works correctly
func TestMigration_FromBareToWt(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old directory structure (simulate legacy installation)
	oldMetadataDir := filepath.Join(tmpDir, ".bare", "worktree-metadata")
	oldQueueDir := filepath.Join(tmpDir, ".bare", "task-queue")
	oldLocksDir := filepath.Join(tmpDir, ".bare", "locks")

	// Create old directories
	for _, dir := range []string{oldMetadataDir, oldQueueDir, oldLocksDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create old directory: %v", err)
		}
	}

	// Create dummy files in old locations
	testFiles := map[string]string{
		filepath.Join(oldMetadataDir, "alice.json"):     "{}",
		filepath.Join(oldQueueDir, "pending", "task1"):  "{}",
		filepath.Join(oldLocksDir, "task1.json"):        "{}",
	}

	for path, content := range testFiles {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	// Verify migration is needed
	if !migrate.IsMigrationNeeded(tmpDir) {
		t.Error("IsMigrationNeeded() should return true for legacy installation")
	}

	// Create .wt directories
	rm := NewManager(tmpDir, "")
	if err := rm.EnsureWtStateDir(); err != nil {
		t.Fatalf("EnsureWtStateDir() failed: %v", err)
	}

	// Run migration
	if err := migrate.MigrateStateFromBareToWt(tmpDir); err != nil {
		t.Fatalf("MigrateStateFromBareToWt() failed: %v", err)
	}

	// Verify old directories are gone
	for _, dir := range []string{oldMetadataDir, oldQueueDir, oldLocksDir} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("old directory still exists: %s", dir)
		}
	}

	// Verify files are in new locations
	if _, err := os.Stat(filepath.Join(tmpDir, ".wt", "metadata", "alice.json")); err != nil {
		t.Errorf("metadata file not migrated: %v", err)
	}

	// Verify migration is no longer needed
	if migrate.IsMigrationNeeded(tmpDir) {
		t.Error("IsMigrationNeeded() should return false after migration")
	}
}

// TestMigration_FreshInstallation verifies migration is skipped for new installations
func TestMigration_FreshInstallation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create bare directory (new installation, no old state)
	bareDir := filepath.Join(tmpDir, ".bare")
	if err := os.MkdirAll(bareDir, 0755); err != nil {
		t.Fatalf("failed to create bare dir: %v", err)
	}

	// Migration should not be needed
	if migrate.IsMigrationNeeded(tmpDir) {
		t.Error("IsMigrationNeeded() should return false for fresh installation")
	}

	// Migration should be no-op
	if err := migrate.MigrateStateFromBareToWt(tmpDir); err != nil {
		t.Fatalf("MigrateStateFromBareToWt() failed: %v", err)
	}
}

