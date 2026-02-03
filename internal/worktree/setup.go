package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tdstein/wt/internal/git"
)

// ManagerOptions specifies options for repository management
type ManagerOptions struct {
	Partial      bool // Use partial clone (--filter=blob:none)
	ShallowDepth int  // Clone depth (0 = full history)
}

// Manager handles repository operations
type Manager struct {
	targetPath string         // Path to the worktree root (e.g., ~/wt/my-project)
	repoURL    string         // Repository URL (empty for local projects)
	opts       ManagerOptions // Clone options
}

// NewManager creates a new repository manager
func NewManager(targetPath, repoURL string) *Manager {
	return &Manager{
		targetPath: targetPath,
		repoURL:    repoURL,
		opts:       ManagerOptions{},
	}
}

// NewManagerWithOptions creates a new repository manager with clone options
func NewManagerWithOptions(targetPath, repoURL string, opts ManagerOptions) *Manager {
	return &Manager{
		targetPath: targetPath,
		repoURL:    repoURL,
		opts:       opts,
	}
}

// TargetExists checks if the target directory exists
func (m *Manager) TargetExists() bool {
	_, err := os.Stat(m.targetPath)
	return err == nil
}

// RemoveTarget removes the target directory and all its contents
func (m *Manager) RemoveTarget() error {
	return os.RemoveAll(m.targetPath)
}

// CreateTarget creates the target directory
func (m *Manager) CreateTarget() error {
	return os.MkdirAll(m.targetPath, 0755)
}

// barePath returns the path to the bare repository
func (m *Manager) barePath() string {
	return filepath.Join(m.targetPath, ".bare")
}

// gitPointerPath returns the path to the .git pointer file
func (m *Manager) gitPointerPath() string {
	return filepath.Join(m.targetPath, ".git")
}

// mainPath returns the path to the main worktree
func (m *Manager) mainPath() string {
	return filepath.Join(m.targetPath, "main")
}

// InitLocalBare initializes a bare repository for a local project
func (m *Manager) InitLocalBare() error {
	if err := git.Init(m.barePath(), true); err != nil {
		return fmt.Errorf("failed to init bare repository: %w", err)
	}

	// Set default branch to main
	if err := git.SymbolicRef(m.barePath(), "HEAD", "refs/heads/main"); err != nil {
		return fmt.Errorf("failed to set symbolic ref: %w", err)
	}

	return nil
}

// CloneRemoteBare clones a bare repository from a remote URL
func (m *Manager) CloneRemoteBare() error {
	opts := git.CloneOptions{
		Bare:         true,
		Partial:      m.opts.Partial,
		ShallowDepth: m.opts.ShallowDepth,
	}

	if err := git.CloneWithOptions(m.repoURL, m.barePath(), opts); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Configure fetch refspec to include all branches
	if err := git.Config(m.barePath(), "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"); err != nil {
		return fmt.Errorf("failed to configure remote: %w", err)
	}

	// Fetch to populate remote tracking branches
	// Note: This is needed to create refs/remotes/origin/* for branch tracking
	if err := git.Fetch(m.barePath(), "origin"); err != nil {
		return fmt.Errorf("failed to fetch remote branches: %w", err)
	}

	return nil
}

// CreateGitPointer creates a .git pointer file pointing to .bare
func (m *Manager) CreateGitPointer() error {
	content := "gitdir: ./.bare\n"
	if err := os.WriteFile(m.gitPointerPath(), []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create .git pointer: %w", err)
	}
	return nil
}

// GetRemoteDefaultBranch retrieves the default branch from the remote
// Uses local refs first to avoid network calls, with fallback to git remote show
func (m *Manager) GetRemoteDefaultBranch() (string, error) {
	// Try 1: Parse local symbolic ref (fastest, no network)
	branch, err := m.getRemoteDefaultBranchFromSymbolicRef()
	if err == nil {
		return branch, nil
	}

	// Try 2: Parse from local HEAD file
	branch, err = m.getRemoteDefaultBranchFromFile()
	if err == nil {
		return branch, nil
	}

	// Try 3: Check for common branch names locally
	branch, err = m.getRemoteDefaultBranchFromCommon()
	if err == nil {
		return branch, nil
	}

	// Fallback 4: Use git remote show (requires network call)
	return m.getRemoteDefaultBranchFromRemoteShow()
}

// getRemoteDefaultBranchFromSymbolicRef reads refs/remotes/origin/HEAD symbolic ref
func (m *Manager) getRemoteDefaultBranchFromSymbolicRef() (string, error) {
	headPath := filepath.Join(m.barePath(), "refs", "remotes", "origin", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return "", fmt.Errorf("could not read symbolic ref: %w", err)
	}

	// File format: "ref: refs/remotes/origin/<branch-name>"
	text := strings.TrimSpace(string(content))
	if strings.HasPrefix(text, "ref: refs/remotes/origin/") {
		branch := strings.TrimPrefix(text, "ref: refs/remotes/origin/")
		if branch != "" {
			return branch, nil
		}
	}

	return "", fmt.Errorf("invalid symbolic ref format")
}

// getRemoteDefaultBranchFromFile reads from .bare/refs/remotes/origin/HEAD
func (m *Manager) getRemoteDefaultBranchFromFile() (string, error) {
	// This attempts alternative file locations
	alternativePaths := []string{
		filepath.Join(m.barePath(), "packed-refs"),
	}

	for _, path := range alternativePaths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Look for "refs/remotes/origin/HEAD"
		for _, line := range strings.Split(string(content), "\n") {
			if strings.Contains(line, "refs/remotes/origin/HEAD") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					ref := parts[len(parts)-1]
					if strings.HasPrefix(ref, "refs/remotes/origin/") {
						branch := strings.TrimPrefix(ref, "refs/remotes/origin/")
						if branch != "" {
							return branch, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("HEAD ref not found in files")
}

// getRemoteDefaultBranchFromCommon checks for common branch names (main, master, develop)
func (m *Manager) getRemoteDefaultBranchFromCommon() (string, error) {
	commonBranches := []string{"main", "master", "develop", "development"}

	for _, branch := range commonBranches {
		refPath := filepath.Join(m.barePath(), "refs", "remotes", "origin", branch)
		if _, err := os.Stat(refPath); err == nil {
			return branch, nil
		}
	}

	return "", fmt.Errorf("no common branch names found")
}

// getRemoteDefaultBranchFromRemoteShow uses git remote show (network call)
func (m *Manager) getRemoteDefaultBranchFromRemoteShow() (string, error) {
	output, err := git.RemoteShow(m.barePath(), "origin")
	if err != nil {
		return "", fmt.Errorf("failed to get remote info: %w", err)
	}

	// Parse output for "HEAD branch: <branch>"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "HEAD branch:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2], nil
			}
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}

// CreateLocalWorktree creates the primary worktree for a local project
func (m *Manager) CreateLocalWorktree() error {
	// Create worktree with new branch 'main'
	if err := git.WorktreeAdd(m.targetPath, "main", "main", true); err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	// Create initial commit
	if err := git.Commit(m.mainPath(), "Initial commit", true); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	return nil
}

// CreateRemoteWorktree creates the primary worktree for a remote project
func (m *Manager) CreateRemoteWorktree(branch string) error {
	if err := git.WorktreeAdd(m.targetPath, "main", branch, false); err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	// Set upstream branch for push/pull operations
	if err := git.SetUpstream(m.mainPath(), branch, "origin", branch); err != nil {
		return fmt.Errorf("failed to set upstream: %w", err)
	}

	return nil
}

// ListWorktrees lists all worktrees
func (m *Manager) ListWorktrees() (string, error) {
	return git.WorktreeList(m.targetPath)
}

// GetSizes returns the sizes of .bare and main directories
func (m *Manager) GetSizes() (string, error) {
	cmd := exec.Command("du", "-sh", m.barePath(), m.mainPath())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get sizes: %w", err)
	}
	return string(output), nil
}

// wtStateDir returns the path to the .wt state directory
func (m *Manager) wtStateDir() string {
	return filepath.Join(m.targetPath, ".wt")
}

// EnsureWtStateDir ensures the .wt state directory and subdirectories exist
func (m *Manager) EnsureWtStateDir() error {
	subdirs := []string{"metadata"}

	for _, subdir := range subdirs {
		dir := filepath.Join(m.wtStateDir(), subdir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create .wt directory structure: %w", err)
		}
	}

	return nil
}

// SetupLocal performs full setup for a local project
func (m *Manager) SetupLocal() error {
	if err := m.CreateTarget(); err != nil {
		return fmt.Errorf("failed to create target: %w", err)
	}

	if err := m.InitLocalBare(); err != nil {
		return fmt.Errorf("failed to init bare repository: %w", err)
	}

	if err := m.CreateGitPointer(); err != nil {
		return fmt.Errorf("failed to create git pointer: %w", err)
	}

	if err := m.EnsureWtStateDir(); err != nil {
		return fmt.Errorf("failed to ensure .wt state directory: %w", err)
	}

	if err := m.CreateLocalWorktree(); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// SetupRemote performs full setup for a remote project
func (m *Manager) SetupRemote() error {
	if err := m.CreateTarget(); err != nil {
		return fmt.Errorf("failed to create target: %w", err)
	}

	if err := m.CloneRemoteBare(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	if err := m.CreateGitPointer(); err != nil {
		return fmt.Errorf("failed to create git pointer: %w", err)
	}

	if err := m.EnsureWtStateDir(); err != nil {
		return fmt.Errorf("failed to ensure .wt state directory: %w", err)
	}

	defaultBranch, err := m.GetRemoteDefaultBranch()
	if err != nil {
		// Fallback to "main" if we can't determine the default branch
		defaultBranch = "main"
	}

	if err := m.CreateRemoteWorktree(defaultBranch); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}
