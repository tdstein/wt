package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tdstein/wt/internal/git"
)

func TestDetectBaseBranch(t *testing.T) {
	tests := []struct {
		name           string
		explicit       string
		setupFunc      func(t *testing.T) string // Returns repo path
		expectedBranch string
		wantError      bool
	}{
		{
			name:           "explicit branch provided",
			explicit:       "feature-x",
			setupFunc:      nil,
			expectedBranch: "feature-x",
			wantError:      false,
		},
		{
			name:     "current branch detection",
			explicit: "",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Initialize repo
				git.New("init", "--initial-branch=main").WithDir(tmpDir).Run()
				git.New("config", "user.name", "Test").WithDir(tmpDir).Run()
				git.New("config", "user.email", "test@example.com").WithDir(tmpDir).Run()
				// Create initial commit
				os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
				git.New("add", ".").WithDir(tmpDir).Run()
				git.New("commit", "-m", "initial").WithDir(tmpDir).Run()
				// Create and checkout feature branch
				git.New("checkout", "-b", "feature-test").WithDir(tmpDir).Run()
				return tmpDir
			},
			expectedBranch: "feature-test",
			wantError:      false,
		},
		{
			name:     "fallback to main when no current branch",
			explicit: "",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Initialize repo but don't create any commits (no branch)
				git.New("init", "--initial-branch=main").WithDir(tmpDir).Run()
				return tmpDir
			},
			expectedBranch: "main",
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var repoPath string
			if tt.setupFunc != nil {
				repoPath = tt.setupFunc(t)
			} else {
				repoPath = t.TempDir()
			}

			branch, err := DetectBaseBranch(repoPath, tt.explicit)
			if (err != nil) != tt.wantError {
				t.Errorf("DetectBaseBranch() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if branch != tt.expectedBranch {
				t.Errorf("DetectBaseBranch() = %v, want %v", branch, tt.expectedBranch)
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize repo
	git.New("init", "--initial-branch=main").WithDir(tmpDir).Run()
	git.New("config", "user.name", "Test").WithDir(tmpDir).Run()
	git.New("config", "user.email", "test@example.com").WithDir(tmpDir).Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)
	git.New("add", ".").WithDir(tmpDir).Run()
	git.New("commit", "-m", "initial").WithDir(tmpDir).Run()

	// Test on main branch
	branch, err := GetCurrentBranch(tmpDir)
	if err != nil {
		t.Errorf("GetCurrentBranch() error = %v", err)
	}
	if branch != "main" {
		t.Errorf("GetCurrentBranch() = %v, want main", branch)
	}

	// Create and checkout new branch
	git.New("checkout", "-b", "feature").WithDir(tmpDir).Run()
	branch, err = GetCurrentBranch(tmpDir)
	if err != nil {
		t.Errorf("GetCurrentBranch() error = %v", err)
	}
	if branch != "feature" {
		t.Errorf("GetCurrentBranch() = %v, want feature", branch)
	}
}
