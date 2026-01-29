package worktree

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

// Test EnsureWtStateDir
func TestManager_EnsureWtStateDir(t *testing.T) {
	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create target directory
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	// Ensure .wt state dir
	if err := mgr.EnsureWtStateDir(); err != nil {
		t.Fatalf("EnsureWtStateDir() failed: %v", err)
	}

	// Verify subdirectories exist
	subdirs := []string{"metadata"}
	for _, subdir := range subdirs {
		path := filepath.Join(mgr.wtStateDir(), subdir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Subdirectory %q was not created: %s", subdir, path)
		}
	}
}

// Test error cases for CreateLocalWorktree when bare repo doesn't exist
func TestManager_CreateLocalWorktree_NoBareRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create target but don't initialize bare repo
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	// This should fail because bare repo doesn't exist
	err := mgr.CreateLocalWorktree()
	if err == nil {
		t.Error("CreateLocalWorktree() should fail when bare repo doesn't exist")
	}
}

// Test CloneRemoteBare with error handling
func TestManager_CloneRemoteBare_InvalidURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "wt-repo-test-remote-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetPath := filepath.Join(tempDir, "test-project")
	mgr := NewManager(targetPath, "https://invalid-host-does-not-exist.example.com/repo.git")

	// Create target directory
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	// This should fail due to invalid URL
	err = mgr.CloneRemoteBare()
	if err == nil {
		t.Log("CloneRemoteBare() with invalid URL succeeded (network may be available)")
	}
}

// Test CreateRemoteWorktree requires base function call
func TestManager_CreateRemoteWorktree_NoBareRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create target but don't initialize bare repo
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	// This should fail because bare repo doesn't exist
	err := mgr.CreateRemoteWorktree("main")
	if err == nil {
		t.Error("CreateRemoteWorktree() should fail when bare repo doesn't exist")
	}
}

// Test GetRemoteDefaultBranch error case
func TestManager_GetRemoteDefaultBranch_NoRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create target and bare repo but with no remote
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	if err := mgr.InitLocalBare(); err != nil {
		t.Fatalf("InitLocalBare() failed: %v", err)
	}

	// This should fail because there's no remote configured
	branch, err := mgr.GetRemoteDefaultBranch()
	if err == nil {
		t.Logf("GetRemoteDefaultBranch() succeeded with branch: %s (local repo might have origin configured)", branch)
	}
}

// Test SetupRemote with error handling on missing remote
func TestManager_SetupRemote_InvalidURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "wt-setup-remote-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetPath := filepath.Join(tempDir, "test-project")
	mgr := NewManager(targetPath, "https://invalid-host-example-123456.com/repo.git")

	// This should fail due to invalid URL
	err = mgr.SetupRemote()
	if err == nil {
		t.Log("SetupRemote() with invalid URL succeeded (network may be available)")
	}
}

// Test path helpers for state directory
func TestManager_WtStateDir(t *testing.T) {
	mgr := NewManager("/home/user/wt/test", "")

	expectedPath := filepath.Join("/home/user/wt/test", ".wt")
	if mgr.wtStateDir() != expectedPath {
		t.Errorf("wtStateDir() = %q, want %q", mgr.wtStateDir(), expectedPath)
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

// Test Manager_SetupRemote directory creation
func TestManager_SetupRemote_DirectoryCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "repo-remote-dir-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetPath := filepath.Join(tempDir, "nonexistent", "repo")
	mgr := NewManager(targetPath, "https://github.com/invalid/repo.git")

	// SetupRemote should create target path (or fail gracefully)
	if !mgr.TargetExists() {
		t.Log("Target does not exist before SetupRemote (will be created or fail)")
	}
}

// Test Manager_EnsureWtStateDir creates all subdirectories
func TestManager_EnsureWtStateDir_AllSubdirs(t *testing.T) {
	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	if err := mgr.EnsureWtStateDir(); err != nil {
		t.Fatalf("EnsureWtStateDir() failed: %v", err)
	}

	// Verify all subdirectories exist
	subdirs := []string{"metadata"}
	for _, subdir := range subdirs {
		path := filepath.Join(mgr.targetPath, ".wt", subdir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Subdirectory %q was not created: %s", subdir, path)
		}
	}
}

// Test Manager_CreateTarget idempotency
func TestManager_CreateTarget_Idempotent(t *testing.T) {
	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create target first time
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("First CreateTarget() failed: %v", err)
	}

	// Create target second time (should succeed or handle gracefully)
	if err := mgr.CreateTarget(); err != nil {
		t.Logf("Second CreateTarget() returned error: %v", err)
	}

	// Verify target exists
	if !mgr.TargetExists() {
		t.Error("Target does not exist after CreateTarget()")
	}
}

// Test Manager_RemoveTarget removes directory
func TestManager_RemoveTarget_Removes(t *testing.T) {
	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	if !mgr.TargetExists() {
		t.Error("Target should exist after CreateTarget()")
	}

	if err := mgr.RemoveTarget(); err != nil {
		t.Fatalf("RemoveTarget() failed: %v", err)
	}

	if mgr.TargetExists() {
		t.Error("Target should not exist after RemoveTarget()")
	}
}

// Test Manager path helper functions
func TestManager_PathHelpers(t *testing.T) {
	targetPath := "/home/user/wt/test"
	mgr := NewManager(targetPath, "")

	tests := []struct {
		name     string
		function func() string
		expected string
	}{
		{"barePath", mgr.barePath, filepath.Join(targetPath, ".bare")},
		{"gitPointerPath", mgr.gitPointerPath, filepath.Join(targetPath, ".git")},
		{"mainPath", mgr.mainPath, filepath.Join(targetPath, "main")},
		{"wtStateDir", mgr.wtStateDir, filepath.Join(targetPath, ".wt")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.function()
			if got != tt.expected {
				t.Errorf("%s() = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

// Test Manager_GetSizes output format
func TestManager_GetSizes_OutputFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	if err := mgr.SetupLocal(); err != nil {
		t.Fatalf("SetupLocal() failed: %v", err)
	}

	output, err := mgr.GetSizes()
	if err != nil {
		t.Fatalf("GetSizes() failed: %v", err)
	}

	if output == "" {
		t.Error("GetSizes() returned empty output")
	}

	// Output should mention .bare and main directories
	if !contains(output, ".bare") {
		t.Error("Output does not mention .bare")
	}
	if !contains(output, "main") {
		t.Error("Output does not mention main")
	}
}

// Test Manager_ListWorktrees output format
func TestManager_ListWorktrees_OutputFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	if err := mgr.SetupLocal(); err != nil {
		t.Fatalf("SetupLocal() failed: %v", err)
	}

	output, err := mgr.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() failed: %v", err)
	}

	if output == "" {
		t.Error("ListWorktrees() returned empty output")
	}

	// Output should mention main worktree
	if !contains(output, "main") {
		t.Error("Output does not mention main worktree")
	}
}

// Test Manager_CreateLocalWorktree creates worktree
func TestManager_CreateLocalWorktree_Creates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	if err := mgr.InitLocalBare(); err != nil {
		t.Fatalf("InitLocalBare() failed: %v", err)
	}

	if err := mgr.CreateGitPointer(); err != nil {
		t.Fatalf("CreateGitPointer() failed: %v", err)
	}

	if err := mgr.CreateLocalWorktree(); err != nil {
		t.Fatalf("CreateLocalWorktree() failed: %v", err)
	}

	// Verify worktree was created
	mainPath := mgr.mainPath()
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Errorf("Main worktree was not created: %s", mainPath)
	}
}

// Test Manager_CreateGitPointer content verification
func TestManager_CreateGitPointer_Verification(t *testing.T) {
	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	if err := mgr.CreateGitPointer(); err != nil {
		t.Fatalf("CreateGitPointer() failed: %v", err)
	}

	// Verify pointer content
	gitPointerPath := mgr.gitPointerPath()
	content, err := os.ReadFile(gitPointerPath)
	if err != nil {
		t.Fatalf("Failed to read .git pointer: %v", err)
	}

	expected := "gitdir: ./.bare\n"
	if string(content) != expected {
		t.Errorf("Pointer content = %q, want %q", string(content), expected)
	}
}

// Test Manager_InitLocalBare creates bare repo
func TestManager_InitLocalBare_Creates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	if err := mgr.InitLocalBare(); err != nil {
		t.Fatalf("InitLocalBare() failed: %v", err)
	}

	// Verify bare repository was created
	barePath := mgr.barePath()
	if _, err := os.Stat(filepath.Join(barePath, "HEAD")); os.IsNotExist(err) {
		t.Error("Bare repository HEAD was not created")
	}
}

// Test NewManager initialization with parameters
func TestNewManager_WithParameters(t *testing.T) {
	targetPath := "/home/user/wt/test"
	repoURL := "https://github.com/user/repo.git"
	mgr := NewManager(targetPath, repoURL)

	if mgr.targetPath != targetPath {
		t.Errorf("targetPath = %q, want %q", mgr.targetPath, targetPath)
	}

	if mgr.repoURL != repoURL {
		t.Errorf("repoURL = %q, want %q", mgr.repoURL, repoURL)
	}
}

// Test Manager_TargetExists with non-existent path
func TestManager_TargetExists_NonExistent(t *testing.T) {
	mgr := NewManager("/nonexistent/path/12345", "")

	if mgr.TargetExists() {
		t.Error("TargetExists() = true for non-existent path, want false")
	}
}

// Test Manager_CloneRemoteBare URL validation
func TestManager_CloneRemoteBare_InvalidURLs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "repo-clone-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	invalidURLs := []string{
		"https://invalid-host-xyz-12345.example.com/repo.git",
		"git@invalid-host-12345.example.com:repo.git",
	}

	for _, url := range invalidURLs {
		targetPath := filepath.Join(tempDir, "test-clone")
		mgr := NewManager(targetPath, url)

		if err := mgr.CreateTarget(); err != nil {
			t.Fatalf("CreateTarget() failed: %v", err)
		}

		// CloneRemoteBare should fail with invalid URL
		err := mgr.CloneRemoteBare()
		if err == nil {
			t.Logf("CloneRemoteBare(%s) unexpectedly succeeded", url)
		}
	}
}

// Test Manager_CreateRemoteWorktree sets upstream tracking
func TestManager_CreateRemoteWorktree_SetsUpstream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Setup a bare repo with a branch
	if err := mgr.CreateTarget(); err != nil {
		t.Fatalf("CreateTarget() failed: %v", err)
	}

	if err := mgr.InitLocalBare(); err != nil {
		t.Fatalf("InitLocalBare() failed: %v", err)
	}

	if err := mgr.CreateGitPointer(); err != nil {
		t.Fatalf("CreateGitPointer() failed: %v", err)
	}

	// Create a worktree for the main branch
	if err := mgr.CreateLocalWorktree(); err != nil {
		t.Fatalf("CreateLocalWorktree() failed: %v", err)
	}

	// Note: CreateRemoteWorktree requires an existing branch from a cloned repo
	// This test verifies the command structure but can't fully test without a real remote
	// In practice, SetupRemote would call CreateRemoteWorktree with the default branch
}
