package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/posit-dev/wt/internal/git"
)

func TestValidateAgentName(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		wantError bool
	}{
		{"valid alphanumeric", "alice", false},
		{"valid with hyphen", "alice-123", false},
		{"valid with underscore", "alice_123", false},
		{"valid mixed", "test-agent_1", false},
		{"invalid with space", "alice bob", true},
		{"invalid with slash", "alice/bob", true},
		{"invalid with dot", "alice.bob", true},
		{"invalid special chars", "alice@bob", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentName(tt.agentName)

			if tt.wantError && err == nil {
				t.Errorf("ValidateAgentName(%q) expected error, got nil", tt.agentName)
			}

			if !tt.wantError && err != nil {
				t.Errorf("ValidateAgentName(%q) unexpected error: %v", tt.agentName, err)
			}
		})
	}
}

func setupTestAgentManager(t *testing.T) (*Manager, string, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "agent-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	projectPath := filepath.Join(tempDir, "test-project")

	// Initialize bare repository structure (like wt does)
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

	mgr := NewAgentManager(projectPath)
	return mgr, tempDir, projectPath
}

func cleanupTestAgentManager(t *testing.T, tempDir string) {
	t.Helper()
	os.RemoveAll(tempDir)
}

func TestNewAgentManager(t *testing.T) {
	mgr := NewAgentManager("/home/user/wt/test-project")

	if mgr.targetPath != "/home/user/wt/test-project" {
		t.Errorf("targetPath = %q, want %q", mgr.targetPath, "/home/user/wt/test-project")
	}

	if mgr.metadata == nil {
		t.Error("metadata manager is nil")
	}

	if mgr.conflict == nil {
		t.Error("conflict checker is nil")
	}
}

func TestManager_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir, projectPath := setupTestAgentManager(t)
	defer cleanupTestAgentManager(t, tempDir)

	// Create an agent worktree
	err := mgr.Create(CreateOptions{
		AgentName:  "alice",
		TaskID:     "1234",
		BaseBranch: "main",
	})

	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify worktree was created
	worktreePath := filepath.Join(projectPath, "alice")
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Errorf("Worktree was not created: %s", worktreePath)
	}

	// Verify metadata was created
	if !mgr.metadata.Exists("alice") {
		t.Error("Metadata was not created for alice")
	}

	// Verify metadata content
	metadata, err := mgr.metadata.Get("alice")
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	if metadata.TaskID != "1234" {
		t.Errorf("metadata.TaskID = %q, want %q", metadata.TaskID, "1234")
	}

	if metadata.Branch != "task/1234/alice" {
		t.Errorf("metadata.Branch = %q, want %q", metadata.Branch, "task/1234/alice")
	}

	if metadata.BaseBranch != "main" {
		t.Errorf("metadata.BaseBranch = %q, want %q", metadata.BaseBranch, "main")
	}
}

func TestManager_Create_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir, _ := setupTestAgentManager(t)
	defer cleanupTestAgentManager(t, tempDir)

	tests := []struct {
		name    string
		opts    CreateOptions
		wantErr bool
	}{
		{
			name:    "missing agent name",
			opts:    CreateOptions{TaskID: "1234"},
			wantErr: true,
		},
		{
			name:    "missing task ID",
			opts:    CreateOptions{AgentName: "alice"},
			wantErr: true,
		},
		{
			name: "invalid agent name",
			opts: CreateOptions{
				AgentName: "alice bob",
				TaskID:    "1234",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.Create(tt.opts)

			if tt.wantErr && err == nil {
				t.Error("Create() expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Create() unexpected error: %v", err)
			}
		})
	}
}

func TestManager_Create_Duplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir, _ := setupTestAgentManager(t)
	defer cleanupTestAgentManager(t, tempDir)

	opts := CreateOptions{
		AgentName:  "alice",
		TaskID:     "1234",
		BaseBranch: "main",
	}

	// First create should succeed
	if err := mgr.Create(opts); err != nil {
		t.Fatalf("First Create() failed: %v", err)
	}

	// Second create should fail
	err := mgr.Create(opts)
	if err == nil {
		t.Error("Second Create() should fail for duplicate agent")
	}
}

func TestManager_Remove(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir, projectPath := setupTestAgentManager(t)
	defer cleanupTestAgentManager(t, tempDir)

	// Create an agent
	err := mgr.Create(CreateOptions{
		AgentName:  "bob",
		TaskID:     "5678",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Remove the agent
	err = mgr.Remove(RemoveOptions{AgentName: "bob"})
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Verify worktree was removed
	worktreePath := filepath.Join(projectPath, "bob")
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Error("Worktree still exists after removal")
	}

	// Verify metadata was removed
	if mgr.metadata.Exists("bob") {
		t.Error("Metadata still exists after removal")
	}
}

func TestManager_List_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir, _ := setupTestAgentManager(t)
	defer cleanupTestAgentManager(t, tempDir)

	agents, err := mgr.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(agents) != 0 {
		t.Errorf("List() returned %d agents, want 0", len(agents))
	}
}

func TestManager_List_MultipleAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir, _ := setupTestAgentManager(t)
	defer cleanupTestAgentManager(t, tempDir)

	// Create multiple agents
	agents := []string{"alice", "bob", "charlie"}
	for i, agent := range agents {
		err := mgr.Create(CreateOptions{
			AgentName:  agent,
			TaskID:     fmt.Sprintf("%d", i+1),
			BaseBranch: "main",
		})
		if err != nil {
			t.Fatalf("Create(%s) failed: %v", agent, err)
		}
	}

	// List agents
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(list) != len(agents) {
		t.Errorf("List() returned %d agents, want %d", len(list), len(agents))
	}

	// Verify each agent is in the list
	for _, agent := range agents {
		found := false
		for _, info := range list {
			if info.Agent == agent {
				found = true
				if !info.Exists {
					t.Errorf("Agent %s marked as not existing", agent)
				}
				if info.Status != "active" {
					t.Errorf("Agent %s status = %q, want active", agent, info.Status)
				}
				break
			}
		}
		if !found {
			t.Errorf("Agent %s not found in list", agent)
		}
	}
}

func TestManager_Check(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir, _ := setupTestAgentManager(t)
	defer cleanupTestAgentManager(t, tempDir)

	// Create an agent
	err := mgr.Create(CreateOptions{
		AgentName:  "alice",
		TaskID:     "1234",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Check the agent
	result, err := mgr.Check("alice")
	if err != nil {
		t.Fatalf("Check() failed: %v", err)
	}

	if result == nil {
		t.Fatal("Check() returned nil result")
	}

	// A newly created agent should not have conflicts
	if result.HasConflicts {
		t.Error("Check() HasConflicts = true for new agent, want false")
	}
}

func TestManager_GetStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping git operation test in short mode")
	}

	mgr, tempDir, _ := setupTestAgentManager(t)
	defer cleanupTestAgentManager(t, tempDir)

	// Initially empty
	status, err := mgr.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() failed: %v", err)
	}

	if status.TotalCount != 0 {
		t.Errorf("GetStatus() TotalCount = %d, want 0", status.TotalCount)
	}

	if status.ActiveCount != 0 {
		t.Errorf("GetStatus() ActiveCount = %d, want 0", status.ActiveCount)
	}

	// Create agents
	mgr.Create(CreateOptions{AgentName: "alice", TaskID: "1", BaseBranch: "main"})
	mgr.Create(CreateOptions{AgentName: "bob", TaskID: "2", BaseBranch: "main"})

	// Get status again
	status, err = mgr.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() failed: %v", err)
	}

	if status.TotalCount != 2 {
		t.Errorf("GetStatus() TotalCount = %d, want 2", status.TotalCount)
	}

	if status.ActiveCount != 2 {
		t.Errorf("GetStatus() ActiveCount = %d, want 2", status.ActiveCount)
	}
}

func TestParseOlderThan(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		want      int
		wantError bool
	}{
		{"7 days", "7d", 7, false},
		{"14 days", "14d", 14, false},
		{"without d suffix", "30", 30, false},
		{"zero", "0d", 0, false},
		{"invalid", "abc", 0, true},
		{"negative", "-5d", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOlderThan(tt.arg)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseOlderThan(%q) expected error, got nil", tt.arg)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseOlderThan(%q) unexpected error: %v", tt.arg, err)
				return
			}

			if got != tt.want {
				t.Errorf("ParseOlderThan(%q) = %d, want %d", tt.arg, got, tt.want)
			}
		})
	}
}
