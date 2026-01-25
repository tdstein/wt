package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/tdstein/wt/internal/git"
)

func TestRootCommand(t *testing.T) {
	cmd := newRootCmd()

	if cmd.Use != "wt [repo-url|name] [name]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "wt [repo-url|name] [name]")
	}

	if cmd.Short != "Git worktree setup for parallel Claude Code agents" {
		t.Errorf("Short description mismatch")
	}
}

func TestAgentCommand(t *testing.T) {
	cmd := newAgentCmd()

	if cmd.Use != "agent" {
		t.Errorf("Use = %q, want %q", cmd.Use, "agent")
	}

	// Check that all subcommands are registered
	expectedSubcommands := []string{"create", "remove", "list", "check", "sync", "prune", "status"}
	for _, name := range expectedSubcommands {
		found := false
		for _, subcmd := range cmd.Commands() {
			if subcmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand %q not found", name)
		}
	}
}

func TestFindWtRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp directory with .bare subdirectory
	tempDir, err := os.MkdirTemp("", "wt-cli-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	barePath := filepath.Join(tempDir, ".bare")
	if err := os.Mkdir(barePath, 0755); err != nil {
		t.Fatalf("Failed to create .bare dir: %v", err)
	}

	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test finding wt root
	root, err := findWtRoot()
	if err != nil {
		t.Fatalf("findWtRoot() failed: %v", err)
	}

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	rootResolved, _ := filepath.EvalSymlinks(root)
	tempDirResolved, _ := filepath.EvalSymlinks(tempDir)

	if rootResolved != tempDirResolved {
		t.Errorf("findWtRoot() = %q, want %q", root, tempDir)
	}
}

func TestFindWtRoot_NotInWtDir(t *testing.T) {
	// Create temp directory without .bare
	tempDir, err := os.MkdirTemp("", "wt-cli-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test finding wt root (should fail)
	_, err = findWtRoot()
	if err == nil {
		t.Error("findWtRoot() should fail when not in wt directory")
	}
}

func TestRepeatString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"empty", "", 5, ""},
		{"single char", "-", 5, "-----"},
		{"multi char", "ab", 3, "ababab"},
		{"zero count", "x", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repeatString(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("repeatString(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
		})
	}
}

func TestLogFunctions(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logInfo("test message")
	logSuccess("success message")
	logWarn("warning message")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()
	if output == "" {
		t.Error("Log functions produced no output")
	}
}

func TestIsInteractive(t *testing.T) {
	// This test just verifies the function doesn't crash
	_ = isInteractive()
}

func TestIsColorEnabled(t *testing.T) {
	// This test just verifies the function doesn't crash
	_ = isColorEnabled()
}

func setupTestWtRepo(t *testing.T) (string, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "wt-cli-integration-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	projectPath := filepath.Join(tempDir, "test-project")

	// Initialize bare repository structure
	barePath := filepath.Join(projectPath, ".bare")
	if err := git.Init(barePath, true); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	// Set default branch
	if err := git.SymbolicRef(barePath, "HEAD", "refs/heads/main"); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to set symbolic ref: %v", err)
	}

	// Create .git pointer
	gitPointer := filepath.Join(projectPath, ".git")
	if err := os.WriteFile(gitPointer, []byte("gitdir: ./.bare\n"), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create .git pointer: %v", err)
	}

	// Create main worktree
	if err := git.WorktreeAdd(projectPath, "main", "main", true); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create main worktree: %v", err)
	}

	mainPath := filepath.Join(projectPath, "main")
	if err := git.Commit(mainPath, "Initial commit", true); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tempDir, projectPath
}

func TestAgentCommands_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, projectPath := setupTestWtRepo(t)
	defer os.RemoveAll(tempDir)

	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Change to project directory
	if err := os.Chdir(projectPath); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test agent create
	t.Run("create", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"agent", "create", "test-agent", "123", "main"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("agent create failed: %v", err)
		}

		// Verify agent directory exists
		agentPath := filepath.Join(projectPath, "test-agent")
		if _, err := os.Stat(agentPath); os.IsNotExist(err) {
			t.Error("Agent directory was not created")
		}
	})

	// Test agent list
	t.Run("list", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"agent", "list"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("agent list failed: %v", err)
		}
	})

	// Test agent status
	t.Run("status", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"agent", "status"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("agent status failed: %v", err)
		}
	})

	// Test agent check
	t.Run("check", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"agent", "check", "test-agent"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("agent check failed: %v", err)
		}
	})

	// Test agent remove
	t.Run("remove", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"agent", "remove", "test-agent"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("agent remove failed: %v", err)
		}

		// Verify agent directory was removed
		agentPath := filepath.Join(projectPath, "test-agent")
		if _, err := os.Stat(agentPath); !os.IsNotExist(err) {
			t.Error("Agent directory still exists after removal")
		}
	})
}
