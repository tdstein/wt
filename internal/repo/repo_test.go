package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager("/home/user/wt/test-project", "https://github.com/user/repo.git")

	if mgr.targetPath != "/home/user/wt/test-project" {
		t.Errorf("targetPath = %q, want %q", mgr.targetPath, "/home/user/wt/test-project")
	}

	if mgr.repoURL != "https://github.com/user/repo.git" {
		t.Errorf("repoURL = %q, want %q", mgr.repoURL, "https://github.com/user/repo.git")
	}
}

func TestManager_Paths(t *testing.T) {
	mgr := NewManager("/home/user/wt/test-project", "")

	tests := []struct {
		name     string
		method   func() string
		expected string
	}{
		{
			name:     "barePath",
			method:   mgr.barePath,
			expected: filepath.Join("/home/user/wt/test-project", ".bare"),
		},
		{
			name:     "gitPointerPath",
			method:   mgr.gitPointerPath,
			expected: filepath.Join("/home/user/wt/test-project", ".git"),
		},
		{
			name:     "mainPath",
			method:   mgr.mainPath,
			expected: filepath.Join("/home/user/wt/test-project", "main"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.method()
			if got != tt.expected {
				t.Errorf("%s() = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func setupTestRepo(t *testing.T) (*Manager, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "wt-repo-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	targetPath := filepath.Join(tempDir, "test-project")
	return NewManager(targetPath, ""), tempDir
}

func cleanupTestRepo(t *testing.T, tempDir string) {
	t.Helper()
	os.RemoveAll(tempDir)
}

func TestManager_TargetExists(t *testing.T) {
	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Should not exist initially
	if mgr.TargetExists() {
		t.Error("TargetExists() = true, want false for non-existent directory")
	}

	// Create the target
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	// Should exist now
	if !mgr.TargetExists() {
		t.Error("TargetExists() = false, want true after creating directory")
	}
}

func TestManager_CreateTarget(t *testing.T) {
	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	if _, err := os.Stat(mgr.targetPath); os.IsNotExist(err) {
		t.Errorf("Target directory was not created: %s", mgr.targetPath)
	}
}

func TestManager_RemoveTarget(t *testing.T) {
	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create target
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	// Remove target
	if err := mgr.RemoveTarget(); err != nil {
		t.Fatalf("RemoveTarget() failed: %v", err)
	}

	// Should not exist after removal
	if mgr.TargetExists() {
		t.Error("Target still exists after RemoveTarget()")
	}
}

func TestManager_CreateGitPointer(t *testing.T) {
	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create target directory first
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	// Create git pointer
	if err := mgr.CreateGitPointer(); err != nil {
		t.Fatalf("CreateGitPointer() failed: %v", err)
	}

	// Verify pointer file exists
	gitPointerPath := mgr.gitPointerPath()
	if _, err := os.Stat(gitPointerPath); os.IsNotExist(err) {
		t.Errorf(".git pointer file was not created: %s", gitPointerPath)
	}

	// Verify content
	content, err := os.ReadFile(gitPointerPath)
	if err != nil {
		t.Fatalf("Failed to read .git pointer: %v", err)
	}

	expected := "gitdir: ./.bare\n"
	if string(content) != expected {
		t.Errorf(".git pointer content = %q, want %q", string(content), expected)
	}
}

func TestManager_InitLocalBare(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create target directory first
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	// Initialize bare repository
	if err := mgr.InitLocalBare(); err != nil {
		t.Fatalf("InitLocalBare() failed: %v", err)
	}

	// Verify bare repository was created
	barePath := mgr.barePath()
	if _, err := os.Stat(filepath.Join(barePath, "HEAD")); os.IsNotExist(err) {
		t.Errorf("Bare repository was not initialized: %s", barePath)
	}

	// Verify it's a bare repository
	if _, err := os.Stat(filepath.Join(barePath, "objects")); os.IsNotExist(err) {
		t.Error("Bare repository missing objects directory")
	}
}

func TestManager_SetupLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Run full local setup
	if err := mgr.SetupLocal(); err != nil {
		t.Fatalf("SetupLocal() failed: %v", err)
	}

	// Verify target directory exists
	if !mgr.TargetExists() {
		t.Error("Target directory was not created")
	}

	// Verify bare repository exists
	barePath := mgr.barePath()
	if _, err := os.Stat(filepath.Join(barePath, "HEAD")); os.IsNotExist(err) {
		t.Error("Bare repository was not created")
	}

	// Verify .git pointer exists
	gitPointerPath := mgr.gitPointerPath()
	if _, err := os.Stat(gitPointerPath); os.IsNotExist(err) {
		t.Error(".git pointer was not created")
	}

	// Verify main worktree exists
	mainPath := mgr.mainPath()
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Error("Main worktree was not created")
	}

	// Verify main worktree has .git file
	mainGitPath := filepath.Join(mainPath, ".git")
	if _, err := os.Stat(mainGitPath); os.IsNotExist(err) {
		t.Error("Main worktree .git file was not created")
	}
}

func TestManager_ListWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Setup local project
	if err := mgr.SetupLocal(); err != nil {
		t.Fatalf("SetupLocal() failed: %v", err)
	}

	// List worktrees
	output, err := mgr.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() failed: %v", err)
	}

	// Output should contain "main"
	if output == "" {
		t.Error("ListWorktrees() returned empty output")
	}

	// Should mention the main worktree path
	mainPath := mgr.mainPath()
	if !contains(output, mainPath) && !contains(output, "main") {
		t.Errorf("ListWorktrees() output does not mention main worktree:\n%s", output)
	}
}

func TestManager_GetSizes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping du operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Setup local project
	if err := mgr.SetupLocal(); err != nil {
		t.Fatalf("SetupLocal() failed: %v", err)
	}

	// Get sizes
	output, err := mgr.GetSizes()
	if err != nil {
		t.Fatalf("GetSizes() failed: %v", err)
	}

	if output == "" {
		t.Error("GetSizes() returned empty output")
	}

	// Output should mention .bare and main
	if !contains(output, ".bare") {
		t.Error("GetSizes() output does not mention .bare")
	}
	if !contains(output, "main") {
		t.Error("GetSizes() output does not mention main")
	}
}

func TestManager_SetupLocal_IdempotencyPrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// First setup should succeed
	if err := mgr.SetupLocal(); err != nil {
		t.Fatalf("First SetupLocal() failed: %v", err)
	}

	// Second setup should fail because directory already exists
	err := mgr.SetupLocal()
	if err == nil {
		t.Error("Second SetupLocal() should fail when directory exists")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
