package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	mgr := NewMetadataManager("/home/user/wt/test-project")
	if mgr.targetPath != "/home/user/wt/test-project" {
		t.Errorf("targetPath = %q, want %q", mgr.targetPath, "/home/user/wt/test-project")
	}
}

func TestMetadataManager_metadataDir(t *testing.T) {
	mgr := NewMetadataManager("/home/user/wt/test-project")
	got := mgr.metadataDir()
	want := filepath.Join("/home/user/wt/test-project", ".bare", "worktree-metadata")
	if got != want {
		t.Errorf("metadataDir() = %q, want %q", got, want)
	}
}

func TestMetadataManager_metadataFile(t *testing.T) {
	mgr := NewMetadataManager("/home/user/wt/test-project")
	got := mgr.metadataFile("alice")
	want := filepath.Join("/home/user/wt/test-project", ".bare", "worktree-metadata", "alice.json")
	if got != want {
		t.Errorf("metadataFile(\"alice\") = %q, want %q", got, want)
	}
}

func setupTestManager(t *testing.T) (*MetadataManager, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "wt-metadata-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	return NewMetadataManager(tempDir), tempDir
}

func cleanupTestManager(t *testing.T, tempDir string) {
	t.Helper()
	os.RemoveAll(tempDir)
}

func TestMetadataManager_Init(t *testing.T) {
	mgr, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	if err := mgr.Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	metadataDir := mgr.metadataDir()
	if _, err := os.Stat(metadataDir); os.IsNotExist(err) {
		t.Errorf("metadata directory was not created: %s", metadataDir)
	}
}

func TestMetadataManager_Create(t *testing.T) {
	mgr, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	err := mgr.Create("alice", "1234", "task/1234/alice", "main")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify file was created
	metadataFile := mgr.metadataFile("alice")
	if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
		t.Errorf("metadata file was not created: %s", metadataFile)
	}

	// Verify content
	metadata, err := mgr.Get("alice")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if metadata.Agent != "alice" {
		t.Errorf("Agent = %q, want %q", metadata.Agent, "alice")
	}
	if metadata.TaskID != "1234" {
		t.Errorf("TaskID = %q, want %q", metadata.TaskID, "1234")
	}
	if metadata.Branch != "task/1234/alice" {
		t.Errorf("Branch = %q, want %q", metadata.Branch, "task/1234/alice")
	}
	if metadata.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want %q", metadata.BaseBranch, "main")
	}
	if metadata.Status != "active" {
		t.Errorf("Status = %q, want %q", metadata.Status, "active")
	}
	if metadata.Created.IsZero() {
		t.Error("Created timestamp is zero")
	}
	if metadata.LastActivity.IsZero() {
		t.Error("LastActivity timestamp is zero")
	}
}

func TestMetadataManager_Touch(t *testing.T) {
	mgr, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	// Create initial metadata
	err := mgr.Create("bob", "5678", "task/5678/bob", "main")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Get initial metadata
	initialMetadata, err := mgr.Get("bob")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Touch the metadata
	err = mgr.Touch("bob")
	if err != nil {
		t.Fatalf("Touch() failed: %v", err)
	}

	// Get updated metadata
	updatedMetadata, err := mgr.Get("bob")
	if err != nil {
		t.Fatalf("Get() failed after Touch: %v", err)
	}

	// LastActivity should be updated
	if !updatedMetadata.LastActivity.After(initialMetadata.LastActivity) {
		t.Errorf("LastActivity was not updated: initial=%v, updated=%v",
			initialMetadata.LastActivity, updatedMetadata.LastActivity)
	}

	// Created should remain unchanged
	if !updatedMetadata.Created.Equal(initialMetadata.Created) {
		t.Errorf("Created timestamp changed: initial=%v, updated=%v",
			initialMetadata.Created, updatedMetadata.Created)
	}
}

func TestMetadataManager_Get_NotFound(t *testing.T) {
	mgr, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	_, err := mgr.Get("nonexistent")
	if err == nil {
		t.Error("Get() succeeded for nonexistent agent, expected error")
	}
}

func TestMetadataManager_GetField(t *testing.T) {
	mgr, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	err := mgr.Create("charlie", "9999", "task/9999/charlie", "develop")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	tests := []struct {
		name      string
		field     string
		want      string
		wantError bool
	}{
		{"agent field", "agent", "charlie", false},
		{"task_id field", "task_id", "9999", false},
		{"branch field", "branch", "task/9999/charlie", false},
		{"base_branch field", "base_branch", "develop", false},
		{"status field", "status", "active", false},
		{"unknown field", "unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mgr.GetField("charlie", tt.field)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetField(%q) expected error, got nil", tt.field)
				}
				return
			}

			if err != nil {
				t.Errorf("GetField(%q) unexpected error: %v", tt.field, err)
				return
			}

			if got != tt.want {
				t.Errorf("GetField(%q) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}

	// Test timestamp fields separately (just verify they're not empty)
	created, err := mgr.GetField("charlie", "created")
	if err != nil {
		t.Errorf("GetField(\"created\") failed: %v", err)
	}
	if created == "" {
		t.Error("GetField(\"created\") returned empty string")
	}

	lastActivity, err := mgr.GetField("charlie", "last_activity")
	if err != nil {
		t.Errorf("GetField(\"last_activity\") failed: %v", err)
	}
	if lastActivity == "" {
		t.Error("GetField(\"last_activity\") returned empty string")
	}
}

func TestMetadataManager_Remove(t *testing.T) {
	mgr, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	// Create metadata
	err := mgr.Create("david", "1111", "task/1111/david", "main")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify it exists
	if !mgr.Exists("david") {
		t.Error("Metadata should exist before removal")
	}

	// Remove it
	err = mgr.Remove("david")
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Verify it's gone
	if mgr.Exists("david") {
		t.Error("Metadata should not exist after removal")
	}

	// Removing nonexistent should not error
	err = mgr.Remove("nonexistent")
	if err != nil {
		t.Errorf("Remove() of nonexistent agent failed: %v", err)
	}
}

func TestMetadataManager_List(t *testing.T) {
	mgr, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	// List should return empty when no metadata exists
	files, err := mgr.List()
	if err != nil {
		t.Fatalf("List() failed on empty directory: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("List() returned %d files, want 0", len(files))
	}

	// Create some metadata
	agents := []string{"alice", "bob", "charlie"}
	for i, agent := range agents {
		err := mgr.Create(agent, fmt.Sprintf("%d", i), fmt.Sprintf("task/%d/%s", i, agent), "main")
		if err != nil {
			t.Fatalf("Create(%q) failed: %v", agent, err)
		}
	}

	// List should return all files sorted
	files, err = mgr.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(files) != len(agents) {
		t.Errorf("List() returned %d files, want %d", len(files), len(agents))
	}

	// Verify files are sorted
	for i := 1; i < len(files); i++ {
		if files[i-1] >= files[i] {
			t.Errorf("Files not sorted: %q >= %q", files[i-1], files[i])
		}
	}
}

func TestMetadataManager_Exists(t *testing.T) {
	mgr, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	// Should not exist initially
	if mgr.Exists("emily") {
		t.Error("Exists() returned true for nonexistent agent")
	}

	// Create metadata
	err := mgr.Create("emily", "2222", "task/2222/emily", "main")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Should exist now
	if !mgr.Exists("emily") {
		t.Error("Exists() returned false for existing agent")
	}
}

func TestMetadataManager_Age(t *testing.T) {
	mgr, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	// Create metadata
	err := mgr.Create("frank", "3333", "task/3333/frank", "main")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Age should be close to 0 (just created)
	age, err := mgr.Age("frank")
	if err != nil {
		t.Fatalf("Age() failed: %v", err)
	}

	if age < 0 {
		t.Errorf("Age() = %d, should be non-negative", age)
	}

	if age > 5 {
		t.Errorf("Age() = %d seconds, expected close to 0 for just-created metadata", age)
	}

	// Age for nonexistent agent should error
	_, err = mgr.Age("nonexistent")
	if err == nil {
		t.Error("Age() succeeded for nonexistent agent, expected error")
	}
}

func TestAgeHuman(t *testing.T) {
	tests := []struct {
		name    string
		seconds int64
		want    string
	}{
		{"0 seconds", 0, "0s"},
		{"30 seconds", 30, "30s"},
		{"59 seconds", 59, "59s"},
		{"1 minute", 60, "1m"},
		{"5 minutes", 300, "5m"},
		{"59 minutes", 3540, "59m"},
		{"1 hour", 3600, "1h"},
		{"5 hours", 18000, "5h"},
		{"23 hours", 82800, "23h"},
		{"1 day", 86400, "1d"},
		{"7 days", 604800, "7d"},
		{"30 days", 2592000, "30d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AgeHuman(tt.seconds)
			if got != tt.want {
				t.Errorf("AgeHuman(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}
