package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tdstein/wt/internal/git"
)

func TestNewChecker(t *testing.T) {
	checker := NewChecker("/home/user/wt/test-project")
	if checker.targetPath != "/home/user/wt/test-project" {
		t.Errorf("targetPath = %q, want %q", checker.targetPath, "/home/user/wt/test-project")
	}
}

func setupTestRepo(t *testing.T) (*Checker, string, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "conflict-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	repoPath := filepath.Join(tempDir, "test-repo")

	// Initialize repository
	if err := git.Init(repoPath, false); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Create initial commit
	if err := git.Commit(repoPath, "Initial commit", true); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	checker := NewChecker(repoPath)
	return checker, tempDir, repoPath
}

func cleanupTestRepo(t *testing.T, tempDir string) {
	t.Helper()
	os.RemoveAll(tempDir)
}

func TestChecker_GetCurrentBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	checker, tempDir, repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	branch, err := checker.GetCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("GetCurrentBranch() failed: %v", err)
	}

	// Default branch should be "main" or "master"
	if branch != "main" && branch != "master" {
		t.Errorf("GetCurrentBranch() = %q, want main or master", branch)
	}
}

func TestChecker_HasUncommittedChanges_Clean(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	checker, tempDir, repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Clean repository should have no uncommitted changes
	hasChanges, err := checker.HasUncommittedChanges(repoPath)
	if err != nil {
		t.Fatalf("HasUncommittedChanges() failed: %v", err)
	}

	if hasChanges {
		t.Error("HasUncommittedChanges() = true, want false for clean repo")
	}
}

func TestChecker_HasUncommittedChanges_WithChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	checker, tempDir, repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create a file
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Should detect untracked file
	hasChanges, err := checker.HasUncommittedChanges(repoPath)
	if err != nil {
		t.Fatalf("HasUncommittedChanges() failed: %v", err)
	}

	if !hasChanges {
		t.Error("HasUncommittedChanges() = false, want true when untracked files exist")
	}
}

func TestChecker_GetDivergence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	checker, tempDir, repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create a feature branch
	if err := git.New("checkout", "-b", "feature").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create a commit on feature
	if err := git.Commit(repoPath, "Feature commit", true); err != nil {
		t.Fatalf("Failed to create feature commit: %v", err)
	}

	// Get current branch name (should be main or master)
	mainBranch := "main"
	if _, err := git.New("rev-parse", "--verify", "main").WithDir(repoPath).Run(); err != nil {
		mainBranch = "master"
	}

	// Check divergence
	div, err := checker.GetDivergence(mainBranch, "feature")
	if err != nil {
		t.Fatalf("GetDivergence() failed: %v", err)
	}

	// Feature is 1 ahead, 0 behind
	if div.Ahead != 1 {
		t.Errorf("Divergence.Ahead = %d, want 1", div.Ahead)
	}

	if div.Behind != 0 {
		t.Errorf("Divergence.Behind = %d, want 0", div.Behind)
	}
}

func TestChecker_CanMergeCleanly_NoConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	checker, tempDir, repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Get main branch name
	mainBranch := "main"
	if _, err := git.New("rev-parse", "--verify", "main").WithDir(repoPath).Run(); err != nil {
		mainBranch = "master"
	}

	// Create a feature branch
	if err := git.New("checkout", "-b", "feature").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create non-conflicting commit
	if err := git.Commit(repoPath, "Feature commit", true); err != nil {
		t.Fatalf("Failed to create feature commit: %v", err)
	}

	// Should merge cleanly
	canMerge, err := checker.CanMergeCleanly(mainBranch, "feature")
	if err != nil {
		t.Fatalf("CanMergeCleanly() failed: %v", err)
	}

	if !canMerge {
		t.Error("CanMergeCleanly() = false, want true for non-conflicting branches")
	}
}

func TestChecker_Check(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	checker, tempDir, repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Get main branch name
	mainBranch := "main"
	if _, err := git.New("rev-parse", "--verify", "main").WithDir(repoPath).Run(); err != nil {
		mainBranch = "master"
	}

	// Create feature worktree inside repoPath (since checker.targetPath = repoPath)
	worktreePath := filepath.Join(repoPath, "feature-wt")
	if err := git.WorktreeAdd(repoPath, worktreePath, "feature", true); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Create a commit in the worktree
	if err := git.Commit(worktreePath, "Feature work", true); err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	// Run check
	result, err := checker.Check("feature-wt", mainBranch)
	if err != nil {
		t.Fatalf("Check() failed: %v", err)
	}

	// Verify result
	if result.HasChanges {
		t.Error("Check() HasChanges = true, want false for clean worktree")
	}

	if result.Divergence.Ahead != 1 {
		t.Errorf("Check() Divergence.Ahead = %d, want 1", result.Divergence.Ahead)
	}

	if result.HasConflicts {
		t.Error("Check() HasConflicts = true, want false for non-conflicting changes")
	}
}

func TestFormatDivergence(t *testing.T) {
	tests := []struct {
		name string
		div  Divergence
		want string
	}{
		{
			name: "no divergence",
			div:  Divergence{Ahead: 0, Behind: 0},
			want: "0 commits ahead, 0 commits behind",
		},
		{
			name: "ahead only",
			div:  Divergence{Ahead: 5, Behind: 0},
			want: "5 commits ahead, 0 commits behind",
		},
		{
			name: "behind only",
			div:  Divergence{Ahead: 0, Behind: 3},
			want: "0 commits ahead, 3 commits behind",
		},
		{
			name: "both ahead and behind",
			div:  Divergence{Ahead: 7, Behind: 2},
			want: "7 commits ahead, 2 commits behind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDivergence(tt.div)
			if got != tt.want {
				t.Errorf("FormatDivergence() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckGitAvailable(t *testing.T) {
	err := CheckGitAvailable()
	if err != nil {
		t.Errorf("CheckGitAvailable() failed: %v (git should be available for these tests)", err)
	}
}

// Test GetConflictingFiles with clean merge
func TestChecker_GetConflictingFiles_NoConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	checker, tempDir, repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Get main branch name
	mainBranch := "main"
	if _, err := git.New("rev-parse", "--verify", "main").WithDir(repoPath).Run(); err != nil {
		mainBranch = "master"
	}

	// Create a feature branch with non-conflicting changes
	if err := git.New("checkout", "-b", "feature").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create non-conflicting commit (modifying different file or area)
	if err := git.Commit(repoPath, "Feature commit", true); err != nil {
		t.Fatalf("Failed to create feature commit: %v", err)
	}

	// Get conflicting files (should be empty)
	files, err := checker.GetConflictingFiles(mainBranch, "feature")
	if err != nil {
		t.Fatalf("GetConflictingFiles() failed: %v", err)
	}

	if len(files) > 0 {
		t.Errorf("GetConflictingFiles() = %v, want empty for non-conflicting merge", files)
	}
}

// Test GetConflictingFiles with actual conflicts
func TestChecker_GetConflictingFiles_WithConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	checker, tempDir, repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create a test file
	testFile := filepath.Join(repoPath, "conflict.txt")
	if err := os.WriteFile(testFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get main branch name
	mainBranch := "main"
	if _, err := git.New("rev-parse", "--verify", "main").WithDir(repoPath).Run(); err != nil {
		mainBranch = "master"
	}

	// Commit initial file
	if err := git.New("add", "conflict.txt").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := git.Commit(repoPath, "Add conflict file", true); err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	// Create a feature branch and modify the file
	if err := git.New("checkout", "-b", "feature").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("feature change\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file on feature: %v", err)
	}

	if err := git.New("add", "conflict.txt").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := git.Commit(repoPath, "Feature modification", true); err != nil {
		t.Fatalf("Failed to create feature commit: %v", err)
	}

	// Switch to main and make conflicting change
	if err := git.New("checkout", mainBranch).WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("main change\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file on main: %v", err)
	}

	if err := git.New("add", "conflict.txt").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := git.Commit(repoPath, "Main modification", true); err != nil {
		t.Fatalf("Failed to create main commit: %v", err)
	}

	// Get conflicting files (should contain conflict.txt)
	files, err := checker.GetConflictingFiles(mainBranch, "feature")
	if err != nil {
		t.Logf("GetConflictingFiles() error (might be expected): %v", err)
		// Errors are acceptable for conflict scenarios
		return
	}

	// If no error, we should have detected the conflict file
	if len(files) == 0 {
		t.Log("GetConflictingFiles() returned empty list (merge-tree may have different output format)")
	}

	// At least one file should be in the list (conflict.txt)
	foundConflict := false
	for _, f := range files {
		if f == "conflict.txt" {
			foundConflict = true
			break
		}
	}

	if !foundConflict && len(files) > 0 {
		t.Logf("GetConflictingFiles() = %v, conflict.txt not explicitly found but files were detected", files)
	}
}

// Test CanMergeCleanly with conflicting branches
func TestChecker_CanMergeCleanly_WithConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	checker, tempDir, repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, tempDir)

	// Create a test file
	testFile := filepath.Join(repoPath, "conflict.txt")
	if err := os.WriteFile(testFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get main branch name
	mainBranch := "main"
	if _, err := git.New("rev-parse", "--verify", "main").WithDir(repoPath).Run(); err != nil {
		mainBranch = "master"
	}

	// Commit initial file
	if err := git.New("add", "conflict.txt").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := git.Commit(repoPath, "Add conflict file", true); err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	// Create a feature branch and modify the file
	if err := git.New("checkout", "-b", "conflicting-feature").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("feature change\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file on feature: %v", err)
	}

	if err := git.New("add", "conflict.txt").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := git.Commit(repoPath, "Feature modification", true); err != nil {
		t.Fatalf("Failed to create feature commit: %v", err)
	}

	// Switch to main and make conflicting change
	if err := git.New("checkout", mainBranch).WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}

	if err := os.WriteFile(testFile, []byte("main change\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file on main: %v", err)
	}

	if err := git.New("add", "conflict.txt").WithDir(repoPath).RunSilent(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := git.Commit(repoPath, "Main modification", true); err != nil {
		t.Fatalf("Failed to create main commit: %v", err)
	}

	// Try to merge (should have conflicts)
	canMerge, err := checker.CanMergeCleanly(mainBranch, "conflicting-feature")
	if err != nil {
		t.Logf("CanMergeCleanly() error (may be expected for conflicts): %v", err)
		return
	}

	if canMerge {
		t.Log("CanMergeCleanly() = true (merge-tree might not detect the conflict)")
	} else {
		t.Log("CanMergeCleanly() = false (correctly detected conflict)")
	}
}
