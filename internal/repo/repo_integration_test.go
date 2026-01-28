package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tdstein/wt/internal/agent"
)

// TestSetupLocal_CreatesWtDirectory verifies that SetupLocal creates .wt subdirectories
func TestSetupLocal_CreatesWtDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewManager(tmpDir, "")

	if err := rm.SetupLocal(); err != nil {
		t.Fatalf("SetupLocal() failed: %v", err)
	}

	// Verify .wt subdirectories exist
	subdirs := []string{"metadata"}
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

	// Verify .wt subdirectories exist and are accessible
	wtPath := filepath.Join(tmpDir, ".wt")
	for _, subdir := range []string{"metadata"} {
		path := filepath.Join(wtPath, subdir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected directory not found: %s", path)
		}
	}
}
