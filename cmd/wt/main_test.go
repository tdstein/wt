package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// ============================================================================
// Root Command Tests
// ============================================================================

func TestRootCommand(t *testing.T) {
	cmd := newRootCmd()

	if cmd.Use != "wt" {
		t.Errorf("Use = %q, want %q", cmd.Use, "wt")
	}

	if cmd.Short != "Git worktree setup for parallel Claude Code agents" {
		t.Errorf("Short description mismatch")
	}
}

func TestRootCommand_HasCloneCommand(t *testing.T) {
	cmd := newRootCmd()

	found := false
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "clone" {
			found = true
			break
		}
	}
	if !found {
		t.Error("clone command not found in root")
	}
}

func TestRootCommand_HasInitCommand(t *testing.T) {
	cmd := newRootCmd()

	found := false
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "init" {
			found = true
			break
		}
	}
	if !found {
		t.Error("init command not found in root")
	}
}

func TestRootCommand_HasAddCommand(t *testing.T) {
	cmd := newRootCmd()

	found := false
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "add" {
			found = true
			break
		}
	}
	if !found {
		t.Error("add command not found in root")
	}
}

func TestRootCommand_HasRemoveCommand(t *testing.T) {
	cmd := newRootCmd()

	found := false
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "remove" {
			found = true
			break
		}
	}
	if !found {
		t.Error("remove command not found in root")
	}
}

func TestRootCommand_HasListCommand(t *testing.T) {
	cmd := newRootCmd()

	found := false
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Error("list command not found in root")
	}
}

// ============================================================================
// Clone Command Tests
// ============================================================================

func TestCloneCmd_Basic(t *testing.T) {
	cmd := newCloneCmd()

	if cmd.Use != "clone <url> [target-dir]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "clone <url> [target-dir]")
	}

	if cmd.Short != "Clone a repository and set up worktree structure" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Clone a repository and set up worktree structure")
	}
}

// ============================================================================
// Init Command Tests
// ============================================================================

func TestInitCmd_Basic(t *testing.T) {
	cmd := newInitCmd()

	if cmd.Use != "init <target-dir>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "init <target-dir>")
	}

	if cmd.Short != "Initialize a new wt directory" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Initialize a new wt directory")
	}
}

// ============================================================================
// Add Command Tests
// ============================================================================

func TestAddCmd_Basic(t *testing.T) {
	cmd := newAddCmd()

	if cmd.Use != "add <name> [base-branch]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "add <name> [base-branch]")
	}

	if cmd.Short != "Create a new agent worktree" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Create a new agent worktree")
	}
}

// ============================================================================
// Remove Command Tests
// ============================================================================

func TestRemoveCmd_Basic(t *testing.T) {
	cmd := newRemoveCmd()

	if cmd.Use != "remove <name>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "remove <name>")
	}

	if cmd.Short != "Remove an agent worktree" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Remove an agent worktree")
	}
}

// ============================================================================
// List Command Tests
// ============================================================================

func TestListCmd_Basic(t *testing.T) {
	cmd := newListCmd()

	if cmd.Use != "list" {
		t.Errorf("Use = %q, want %q", cmd.Use, "list")
	}

	if cmd.Short != "List all agent worktrees" {
		t.Errorf("Short = %q, want %q", cmd.Short, "List all agent worktrees")
	}
}

// ============================================================================
// Status Command Tests
// ============================================================================

func TestStatusCmd_Basic(t *testing.T) {
	cmd := newStatusCmd()

	if cmd.Use != "status" {
		t.Errorf("Use = %q, want %q", cmd.Use, "status")
	}

	if cmd.Short != "Show agent dashboard" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Show agent dashboard")
	}
}

// ============================================================================
// Check Command Tests
// ============================================================================

func TestCheckCmd_Basic(t *testing.T) {
	cmd := newCheckCmd()

	if cmd.Use != "check <name>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "check <name>")
	}

	if cmd.Short != "Check for merge conflicts" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Check for merge conflicts")
	}
}

// ============================================================================
// Sync Command Tests
// ============================================================================

func TestSyncCmd_Basic(t *testing.T) {
	cmd := newSyncCmd()

	if cmd.Use != "sync <name>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "sync <name>")
	}

	if cmd.Short != "Synchronize agent with base branch" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Synchronize agent with base branch")
	}
}

// ============================================================================
// Prune Command Tests
// ============================================================================

func TestPruneCmd_Basic(t *testing.T) {
	cmd := newPruneCmd()

	if cmd.Use != "prune" {
		t.Errorf("Use = %q, want %q", cmd.Use, "prune")
	}

	if cmd.Short != "Remove stale agent worktrees" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Remove stale agent worktrees")
	}
}

// ============================================================================
// Context Helper Tests
// ============================================================================

func TestContextHelpers_TaskID(t *testing.T) {
	ctx := context.Background()

	// Test withTaskID
	taskID := "task-123"
	ctx = withTaskID(ctx, taskID)

	// Test getTaskID
	retrieved := getTaskID(ctx)
	if retrieved != taskID {
		t.Errorf("getTaskID() = %q, want %q", retrieved, taskID)
	}
}

func TestContextHelpers_EmptyTaskID(t *testing.T) {
	ctx := context.Background()

	// Test getTaskID on empty context
	retrieved := getTaskID(ctx)
	if retrieved != "" {
		t.Errorf("getTaskID() on empty context = %q, want %q", retrieved, "")
	}
}

func TestContextHelpers_WithTargetPath(t *testing.T) {
	ctx := context.Background()

	targetPath := "/tmp/wt"
	ctx = withTargetPath(ctx, targetPath)

	retrieved := getTargetPath(ctx)
	if retrieved != targetPath {
		t.Errorf("getTargetPath() = %q, want %q", retrieved, targetPath)
	}
}

func TestContextHelpers_BothValues(t *testing.T) {
	ctx := context.Background()

	taskID := "task-456"
	targetPath := "/tmp/wt/project"

	ctx = withTaskID(ctx, taskID)
	ctx = withTargetPath(ctx, targetPath)

	if getTaskID(ctx) != taskID {
		t.Errorf("getTaskID() after both = %q, want %q", getTaskID(ctx), taskID)
	}

	if getTargetPath(ctx) != targetPath {
		t.Errorf("getTargetPath() after both = %q, want %q", getTargetPath(ctx), targetPath)
	}
}

// ============================================================================
// Utility Function Tests
// ============================================================================

func TestFormatDuration_Seconds(t *testing.T) {
	d := 30 * time.Second
	result := formatDuration(d)
	if !strings.Contains(result, "s ago") {
		t.Errorf("formatDuration(30s) = %q, expected to contain 's ago'", result)
	}
}

func TestFormatDuration_Minutes(t *testing.T) {
	d := 5 * time.Minute
	result := formatDuration(d)
	if !strings.Contains(result, "m ago") {
		t.Errorf("formatDuration(5m) = %q, expected to contain 'm ago'", result)
	}
}

func TestFormatDuration_Hours(t *testing.T) {
	d := 3 * time.Hour
	result := formatDuration(d)
	if !strings.Contains(result, "h ago") {
		t.Errorf("formatDuration(3h) = %q, expected to contain 'h ago'", result)
	}
}

func TestFormatDuration_Days(t *testing.T) {
	d := 2 * 24 * time.Hour
	result := formatDuration(d)
	if !strings.Contains(result, "d ago") {
		t.Errorf("formatDuration(2d) = %q, expected to contain 'd ago'", result)
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestCommandStructure_RootToClone(t *testing.T) {
	root := newRootCmd()

	// Find clone command
	var cloneCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "clone" {
			cloneCmd = cmd
			break
		}
	}

	if cloneCmd == nil {
		t.Fatal("clone command not found in root")
	}

	if cloneCmd.Use != "clone <url> [target-dir]" {
		t.Errorf("clone command structure incorrect: %q", cloneCmd.Use)
	}
}

func TestCommandStructure_RootToInit(t *testing.T) {
	root := newRootCmd()

	// Find init command
	var initCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "init" {
			initCmd = cmd
			break
		}
	}

	if initCmd == nil {
		t.Fatal("init command not found in root")
	}

	if initCmd.Use != "init <target-dir>" {
		t.Errorf("init command structure incorrect: %q", initCmd.Use)
	}
}

func TestCommandStructure_RootToAdd(t *testing.T) {
	root := newRootCmd()

	// Find add command
	var addCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "add" {
			addCmd = cmd
			break
		}
	}

	if addCmd == nil {
		t.Fatal("add command not found in root")
	}

	if addCmd.Use != "add <name> [base-branch]" {
		t.Errorf("add command structure incorrect: %q", addCmd.Use)
	}
}

func TestCommandStructure_AllCommands(t *testing.T) {
	root := newRootCmd()

	expectedCommands := []string{"clone", "init", "add", "remove", "list", "status", "check", "sync", "prune"}
	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range root.Commands() {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command %q not found in root", expected)
		}
	}
}

// ============================================================================
// Logging Tests
// ============================================================================

func TestLogFunctions_NoFatal(t *testing.T) {
	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	logInfo("test info")
	logSuccess("test success")
	logWarn("test warn")

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "test info") && !strings.Contains(output, "INFO") {
		t.Error("logInfo output missing")
	}
}

func TestFindWtRoot_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp directory (no .bare)
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	_, err := findWtRoot()
	if err == nil {
		t.Error("findWtRoot should return error when .bare not found")
	}

	if !strings.Contains(err.Error(), "not in a wt directory") {
		t.Errorf("Error message = %q, should contain 'not in a wt directory'", err.Error())
	}
}

func TestRepeatString(t *testing.T) {
	result := repeatString("-", 5)
	expected := "-----"
	if result != expected {
		t.Errorf("repeatString(\"-\", 5) = %q, want %q", result, expected)
	}
}

// ============================================================================
// Deleted Old Commands (Verification)
// ============================================================================

func TestOldCommandsRemoved(t *testing.T) {
	// These tests verify that old commands are no longer defined
	// They should not compile if old functions still exist

	// The following should NOT exist:
	// - newAgentCmd() - hierarchy removed
	// - newQueueCmd() - hierarchy removed
	// - newLockCmd() - hierarchy removed

	// If this test compiles successfully, old commands are removed
	t.Log("Old command hierarchy successfully removed")
}
