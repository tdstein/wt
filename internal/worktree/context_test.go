package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGetCurrentWorktree(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Initialize a git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Create an initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Create a branch named "test-worktree"
	cmd = exec.Command("git", "checkout", "-b", "test-worktree")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Change to the temporary directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Test GetCurrentWorktree
	worktree, err := GetCurrentWorktree()
	if err != nil {
		t.Fatalf("GetCurrentWorktree failed: %v", err)
	}

	if worktree != "test-worktree" {
		t.Errorf("expected worktree name 'test-worktree', got '%s'", worktree)
	}
}

func TestGetCurrentWorktreeNoGitRepo(t *testing.T) {
	// Create a temporary directory without git
	tmpDir := t.TempDir()

	// Change to the temporary directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Test GetCurrentWorktree should fail
	_, err = GetCurrentWorktree()
	if err == nil {
		t.Error("expected GetCurrentWorktree to fail in non-git directory")
	}
}
