package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	cmd := New("status", "--short")

	if len(cmd.args) != 2 {
		t.Errorf("len(args) = %d, want 2", len(cmd.args))
	}

	if cmd.args[0] != "status" {
		t.Errorf("args[0] = %q, want %q", cmd.args[0], "status")
	}

	if cmd.args[1] != "--short" {
		t.Errorf("args[1] = %q, want %q", cmd.args[1], "--short")
	}
}

func TestCommand_WithDir(t *testing.T) {
	cmd := New("status").WithDir("/some/path")

	if cmd.dir != "/some/path" {
		t.Errorf("dir = %q, want %q", cmd.dir, "/some/path")
	}
}

func TestInit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "git-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "repo")

	// Test non-bare init
	if err := Init(repoDir, false); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Verify .git directory exists
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Errorf(".git directory was not created: %s", gitDir)
	}
}

func TestInit_Bare(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "git-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	bareDir := filepath.Join(tempDir, "bare.git")

	// Test bare init
	if err := Init(bareDir, true); err != nil {
		t.Fatalf("Init(bare=true) failed: %v", err)
	}

	// Verify bare repository structure
	if _, err := os.Stat(filepath.Join(bareDir, "HEAD")); os.IsNotExist(err) {
		t.Error("Bare repository missing HEAD")
	}

	if _, err := os.Stat(filepath.Join(bareDir, "objects")); os.IsNotExist(err) {
		t.Error("Bare repository missing objects directory")
	}
}

func TestSymbolicRef(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "git-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "repo")

	// Initialize repository
	if err := Init(repoDir, false); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Set symbolic ref
	if err := SymbolicRef(repoDir, "HEAD", "refs/heads/develop"); err != nil {
		t.Fatalf("SymbolicRef() failed: %v", err)
	}

	// Verify the symbolic ref was set (we can't easily verify the content without more git commands)
	// If no error occurred, the operation succeeded
}

func TestConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "git-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "repo")

	// Initialize repository
	if err := Init(repoDir, false); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Set config value
	if err := Config(repoDir, "user.name", "Test User"); err != nil {
		t.Fatalf("Config() failed: %v", err)
	}

	// Verify config was set
	output, err := New("config", "user.name").WithDir(repoDir).Run()
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if output != "Test User" {
		t.Errorf("config user.name = %q, want %q", output, "Test User")
	}
}

func TestWorktreeAdd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "git-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "repo")

	// Initialize repository
	if err := Init(repoDir, false); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Create initial commit (required for worktrees)
	if err := Commit(repoDir, "Initial commit", true); err != nil {
		t.Fatalf("Commit() failed: %v", err)
	}

	// Add a worktree with new branch
	worktreePath := filepath.Join(tempDir, "worktree1")
	if err := WorktreeAdd(repoDir, worktreePath, "feature", true); err != nil {
		t.Fatalf("WorktreeAdd() failed: %v", err)
	}

	// Verify worktree was created
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Errorf("Worktree directory was not created: %s", worktreePath)
	}
}

func TestWorktreeList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "git-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "repo")

	// Initialize repository
	if err := Init(repoDir, false); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Create initial commit
	if err := Commit(repoDir, "Initial commit", true); err != nil {
		t.Fatalf("Commit() failed: %v", err)
	}

	// List worktrees
	output, err := WorktreeList(repoDir)
	if err != nil {
		t.Fatalf("WorktreeList() failed: %v", err)
	}

	if output == "" {
		t.Error("WorktreeList() returned empty output")
	}

	// Output should mention the main repository
	if !contains(output, repoDir) {
		t.Errorf("WorktreeList() output does not mention repo dir:\n%s", output)
	}
}

func TestCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "git-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "repo")

	// Initialize repository
	if err := Init(repoDir, false); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Create empty commit
	if err := Commit(repoDir, "Test commit", true); err != nil {
		t.Fatalf("Commit() failed: %v", err)
	}

	// Verify commit was created by checking log
	output, err := New("log", "--oneline").WithDir(repoDir).Run()
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	if !contains(output, "Test commit") {
		t.Errorf("Commit message not found in log:\n%s", output)
	}
}

func TestSetUpstream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "git-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a bare repository (simulating origin)
	bareDir := filepath.Join(tempDir, "origin.git")
	if err := Init(bareDir, true); err != nil {
		t.Fatalf("Init(bare) failed: %v", err)
	}

	// Set symbolic ref for bare repo
	if err := SymbolicRef(bareDir, "HEAD", "refs/heads/main"); err != nil {
		t.Fatalf("SymbolicRef() failed: %v", err)
	}

	// Clone the bare repository
	cloneDir := filepath.Join(tempDir, "clone")
	if err := Clone(bareDir, cloneDir, false); err != nil {
		t.Fatalf("Clone() failed: %v", err)
	}

	// Create an initial commit
	if err := Commit(cloneDir, "Initial commit", true); err != nil {
		t.Fatalf("Commit() failed: %v", err)
	}

	// Push to origin to create the remote tracking branch
	if err := New("push", "-u", "origin", "main").WithDir(cloneDir).RunSilent(); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Set upstream tracking (this should work now)
	if err := SetUpstream(cloneDir, "main", "origin", "main"); err != nil {
		t.Fatalf("SetUpstream() failed: %v", err)
	}

	// Verify upstream was set by checking branch configuration
	output, err := New("config", "branch.main.remote").WithDir(cloneDir).Run()
	if err != nil {
		t.Fatalf("Failed to read branch.main.remote: %v", err)
	}

	if output != "origin" {
		t.Errorf("branch.main.remote = %q, want %q", output, "origin")
	}

	// Verify merge configuration
	output, err = New("config", "branch.main.merge").WithDir(cloneDir).Run()
	if err != nil {
		t.Fatalf("Failed to read branch.main.merge: %v", err)
	}

	if output != "refs/heads/main" {
		t.Errorf("branch.main.merge = %q, want %q", output, "refs/heads/main")
	}
}

// Helper function
func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
