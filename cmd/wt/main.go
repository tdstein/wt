package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/tdstein/wt/internal/agent"
	"github.com/tdstein/wt/internal/check"
	"github.com/tdstein/wt/internal/hooks"
	"github.com/tdstein/wt/internal/prune"
	"github.com/tdstein/wt/internal/worktree"
)

var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// ============================================================================
// Root Command - Task-Centric Structure
// ============================================================================

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wt",
		Short: "Git worktree setup for parallel Claude Code agents",
		Long: `wt creates a bare git repository structure optimized for multiple
agents working in parallel worktrees.

Initialize or clone a repository:
  wt clone <url> [target-dir]         Clone from remote
  wt init <target-dir>                 Initialize local project

Manage agent worktrees:
  wt add <name> [base-branch]          Create agent worktree
  wt remove <name>                     Remove agent worktree
  wt list                              List all agents
  wt status                            Show agent dashboard
  wt switch <name>                     Switch to agent worktree
  wt check <name>                      Check for conflicts
  wt sync [name]                       Sync with base branch (defaults to current)
  wt prune [--older-than 7d]          Remove stale worktrees

Claude Code integration:
  wt hooks install                     Install Claude Code hooks
  wt hooks config                      Print hooks configuration
  wt hooks list                        List available hook scripts`,
		Version: version,
	}

	cmd.AddCommand(newCloneCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newCheckCmd())
	cmd.AddCommand(newSyncCmd())
	cmd.AddCommand(newPruneCmd())
	cmd.AddCommand(newSwitchCmd())
	cmd.AddCommand(newHooksCmd())

	return cmd
}

// ============================================================================
// Clone Command - Remote Repositories
// ============================================================================

func newCloneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone <url> [target-dir]",
		Short: "Clone a repository and set up worktree structure",
		Long: `Clone a remote repository and initialize it as a wt directory with bare repository structure.

If target-dir is not specified, it will be derived from the repository URL.

Examples:
  wt clone https://github.com/user/repo
  wt clone https://github.com/user/repo ./my-project`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runClone,
	}
	return cmd
}

func runClone(cmd *cobra.Command, args []string) error {
	// Check prerequisites
	if err := check.CheckGitAvailable(); err != nil {
		return err
	}

	repoURL := args[0]
	if !worktree.IsURL(repoURL) {
		return fmt.Errorf("invalid repository URL: %s", repoURL)
	}

	// Parse arguments
	result, err := worktree.ParseArgs(args)
	if err != nil {
		return err
	}

	logInfo("Target: %s", result.TargetPath)

	// Create repository manager
	mgr := worktree.NewManager(result.TargetPath, result.RepoURL)

	// Check if target exists
	if mgr.TargetExists() {
		if isInteractive() {
			if !confirmRemove(result.TargetPath) {
				logWarn("Aborted")
				return nil
			}
		} else {
			return fmt.Errorf("target exists: %s", result.TargetPath)
		}
		if err := mgr.RemoveTarget(); err != nil {
			return fmt.Errorf("failed to remove target: %w", err)
		}
	}

	// Clone from remote
	logInfo("Cloning from %s...", result.RepoURL)
	if err := mgr.SetupRemote(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Report success
	fmt.Println()
	logSuccess("Worktree ready: %s", result.TargetPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Println("1. Navigate to the repository:")
	fmt.Printf("   cd %s\n", result.TargetPath)
	fmt.Println()
	fmt.Println("2. Set up Claude Code hooks (recommended):")
	fmt.Println("   wt hooks install")
	fmt.Println()
	fmt.Println("3. Create agent worktrees:")
	fmt.Println("   wt add <agent-name> [base-branch]")

	return nil
}

// ============================================================================
// Init Command - Local Projects
// ============================================================================

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <target-dir>",
		Short: "Initialize a new wt directory",
		Long: `Initialize a new local project as a wt directory with bare repository structure.

Examples:
  wt init .
  wt init ./my-project`,
		Args: cobra.ExactArgs(1),
		RunE: runInit,
	}
	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check prerequisites
	if err := check.CheckGitAvailable(); err != nil {
		return err
	}

	targetPath := args[0]

	// Expand to absolute path
	result, err := worktree.ParseArgs([]string{targetPath})
	if err != nil {
		return err
	}

	logInfo("Target: %s", result.TargetPath)

	// Create repository manager
	mgr := worktree.NewManager(result.TargetPath, "")

	// Check if target exists
	if mgr.TargetExists() {
		if isInteractive() {
			if !confirmRemove(result.TargetPath) {
				logWarn("Aborted")
				return nil
			}
		} else {
			return fmt.Errorf("target exists: %s", result.TargetPath)
		}
		if err := mgr.RemoveTarget(); err != nil {
			return fmt.Errorf("failed to remove target: %w", err)
		}
	}

	// Initialize local repository
	logInfo("Initializing local repository...")
	if err := mgr.SetupLocal(); err != nil {
		return fmt.Errorf("failed to setup local repository: %w", err)
	}

	// Report success
	fmt.Println()
	logSuccess("Worktree ready: %s", result.TargetPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Println("1. Navigate to the repository:")
	fmt.Printf("   cd %s\n", result.TargetPath)
	fmt.Println()
	fmt.Println("2. Set up Claude Code hooks (recommended):")
	fmt.Println("   wt hooks install")
	fmt.Println()
	fmt.Println("3. Create agent worktrees:")
	fmt.Println("   wt add <agent-name> [base-branch]")

	return nil
}

// ============================================================================
// Add Command - Create Agent Worktree
// ============================================================================

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> [base-branch]",
		Short: "Create a new agent worktree",
		Long: `Create a new agent worktree with branch <name>.

Examples:
  wt add alice              Create worktree from main
  wt add bob develop        Create worktree from develop`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runAdd,
	}
	return cmd
}

func runAdd(cmd *cobra.Command, args []string) error {
	targetPath, err := findWtRoot()
	if err != nil {
		return err
	}

	mgr := agent.NewManager(targetPath)
	agentName := args[0]

	// Get explicit base branch if provided
	var explicitBaseBranch string
	if len(args) > 1 {
		explicitBaseBranch = args[1]
	}

	// Get current working directory for base branch detection
	cwd, err := os.Getwd()
	if err != nil {
		cwd = targetPath
	}

	// Detect base branch with priority: explicit > current > remote > "main"
	baseBranch, err := worktree.DetectBaseBranch(cwd, explicitBaseBranch)
	if err != nil {
		return fmt.Errorf("failed to detect base branch: %w", err)
	}

	opts := agent.CreateOptions{
		AgentName:  agentName,
		BaseBranch: baseBranch,
	}

	logInfo("Creating agent worktree: %s (base: %s)", agentName, baseBranch)
	if err := mgr.Create(opts); err != nil {
		return err
	}

	logSuccess("Agent worktree created: %s", agentName)
	fmt.Printf("Branch: %s\n", agentName)
	fmt.Printf("Base: %s\n", baseBranch)
	fmt.Printf("Path: %s/%s\n", targetPath, agentName)
	return nil
}

// ============================================================================
// Remove Command - Remove Agent Worktree
// ============================================================================

func newRemoveCmd() *cobra.Command {
	var deleteBranch bool

	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove an agent worktree",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}

			mgr := agent.NewManager(targetPath)
			opts := agent.RemoveOptions{
				AgentName:    args[0],
				DeleteBranch: deleteBranch,
			}

			logInfo("Removing agent worktree: %s", args[0])
			if err := mgr.Remove(opts); err != nil {
				return err
			}

			logSuccess("Agent worktree removed: %s", args[0])
			return nil
		},
	}

	cmd.Flags().BoolVar(&deleteBranch, "delete-branch", false, "Delete the branch after removing worktree")
	return cmd
}

// ============================================================================
// List Command - List All Agent Worktrees
// ============================================================================

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all agent worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}

			mgr := agent.NewManager(targetPath)
			agents, err := mgr.List()
			if err != nil {
				return err
			}

			if len(agents) == 0 {
				fmt.Println("No agent worktrees found")
				return nil
			}

			// Print table header
			fmt.Printf("%-20s %-30s %-10s %-10s\n",
				"AGENT", "BRANCH", "AGE", "STATUS")
			fmt.Println(repeatString("-", 75))

			// Print each agent
			for _, a := range agents {
				fmt.Printf("%-20s %-30s %-10s %-10s\n",
					a.Agent, a.Branch, a.AgeHuman, a.Status)
			}

			return nil
		},
	}

	return cmd
}

// ============================================================================
// Status Command - Show Agent Dashboard
// ============================================================================

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show agent dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}

			mgr := agent.NewManager(targetPath)
			status, err := mgr.GetStatus()
			if err != nil {
				return err
			}

			fmt.Println("Agent Dashboard")
			fmt.Println(repeatString("=", 50))
			fmt.Printf("Total agents: %d\n", status.TotalCount)
			fmt.Printf("Active agents: %d\n", status.ActiveCount)
			fmt.Println()

			if len(status.Agents) == 0 {
				fmt.Println("No agents found")
				return nil
			}

			// Print table
			fmt.Printf("%-20s %-10s %-10s\n",
				"AGENT", "AGE", "STATUS")
			fmt.Println(repeatString("-", 45))

			for _, a := range status.Agents {
				statusColor := ""
				if a.Status == "active" {
					statusColor = colorGreen
				} else {
					statusColor = colorYellow
				}
				fmt.Printf("%-20s %-10s %s%-10s%s\n",
					a.Agent, a.AgeHuman, statusColor, a.Status, colorReset)
			}

			return nil
		},
	}

	return cmd
}

// ============================================================================
// Check Command - Check for Conflicts
// ============================================================================

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check <name>",
		Short: "Check for merge conflicts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}

			mgr := agent.NewManager(targetPath)
			agentName := args[0]
			logInfo("Checking agent: %s", agentName)

			result, err := mgr.Check(agentName)
			if err != nil {
				return err
			}

			// Print results
			fmt.Println()
			fmt.Printf("Agent: %s\n", agentName)
			fmt.Printf("Uncommitted changes: %v\n", result.HasChanges)
			fmt.Printf("Divergence: %s\n", check.FormatDivergence(result.Divergence))
			fmt.Printf("Merge conflicts: %v\n", result.HasConflicts)

			if result.HasConflicts && len(result.ConflictingFiles) > 0 {
				fmt.Println("\nConflicting files:")
				for _, file := range result.ConflictingFiles {
					fmt.Printf("  - %s\n", file)
				}
			}

			if result.HasConflicts {
				logWarn("Agent has merge conflicts with base branch")
				return nil
			}

			logSuccess("No conflicts detected")
			return nil
		},
	}

	return cmd
}

// ============================================================================
// Sync Command - Synchronize with Base Branch
// ============================================================================

func newSyncCmd() *cobra.Command {
	var autoRebase bool

	cmd := &cobra.Command{
		Use:   "sync [name]",
		Short: "Synchronize agent with base branch",
		Long: `Synchronize an agent worktree with its base branch.

If no agent name is provided, the current worktree is used.

Examples:
  wt sync              Sync current worktree
  wt sync alice        Sync specific agent worktree`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}

			// Determine agent name
			var agentName string
			if len(args) > 0 {
				agentName = args[0]
			} else {
				// Auto-detect from current worktree
				detected, err := worktree.GetCurrentWorktree()
				if err != nil {
					return fmt.Errorf("no agent name provided and failed to detect current worktree: %w", err)
				}
				agentName = detected
				logInfo("Using current worktree: %s", agentName)
			}

			mgr := agent.NewManager(targetPath)
			opts := agent.SyncOptions{
				AgentName:  agentName,
				AutoRebase: autoRebase,
			}

			logInfo("Syncing agent: %s", agentName)

			result, err := mgr.Sync(opts)
			if err != nil {
				return err
			}

			if result.Error != nil {
				logError("Sync failed: %v", result.Error)
				return result.Error
			}

			if result.AlreadyUpToDate {
				logSuccess("Already up to date")
				return nil
			}

			if result.Rebased {
				logSuccess("Successfully rebased onto base branch")
			} else {
				logWarn("Base branch has changes - run with --auto-rebase to rebase")
			}

			fmt.Printf("Divergence: %s\n", check.FormatDivergence(result.Divergence))
			return nil
		},
	}

	cmd.Flags().BoolVar(&autoRebase, "auto-rebase", false, "Automatically rebase onto base branch")
	return cmd
}

// ============================================================================
// Prune Command - Remove Stale Worktrees
// ============================================================================

func newPruneCmd() *cobra.Command {
	var olderThan string
	var dryRun bool
	var interactive bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove stale agent worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}

			pruner := prune.NewPruner(targetPath)

			days := 7
			if olderThan != "" {
				var err error
				days, err = prune.ParseOlderThan(olderThan)
				if err != nil {
					return err
				}
			}

			opts := prune.Options{
				OlderThanDays: days,
				DryRun:        dryRun,
				Interactive:   interactive,
			}

			logInfo("Pruning stale agents (older than %d days)...", days)

			result, err := pruner.Prune(opts)
			if err != nil {
				return err
			}

			if len(result.StaleAgents) == 0 {
				fmt.Println("No stale agents found")
				return nil
			}

			if dryRun {
				fmt.Printf("Found %d stale agent(s) (dry run):\n", len(result.StaleAgents))
				for _, name := range result.StaleAgents {
					fmt.Printf("  - %s\n", name)
				}
				return nil
			}

			if len(result.Removed) > 0 {
				logSuccess("Removed %d agent(s):", len(result.Removed))
				for _, name := range result.Removed {
					fmt.Printf("  - %s\n", name)
				}
			}

			if len(result.Errors) > 0 {
				logError("Failed to remove %d agent(s):", len(result.Errors))
				for name, err := range result.Errors {
					fmt.Printf("  - %s: %v\n", name, err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&olderThan, "older-than", "7d", "Remove agents older than N days (e.g., 7d, 14d)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed without removing")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Prompt before removing each agent")

	return cmd
}

// ============================================================================
// Switch Command - Switch Between Agent Worktrees
// ============================================================================

func newSwitchCmd() *cobra.Command {
	var printPath bool

	cmd := &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch to an agent worktree",
		Long: `Switch to an agent worktree by printing the path.

Examples:
  wt switch alice              Print cd command to switch
  wt switch alice --path       Print only the path
  cd $(wt switch alice --path) Use in shell command substitution`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}

			mgr := agent.NewManager(targetPath)
			agentName := args[0]

			// Check if agent exists
			agents, err := mgr.List()
			if err != nil {
				return err
			}

			var agentInfo *agent.Info
			for _, a := range agents {
				if a.Agent == agentName {
					agentInfo = &a
					break
				}
			}

			if agentInfo == nil {
				return fmt.Errorf("agent not found: %s", agentName)
			}

			if !agentInfo.Exists {
				return fmt.Errorf("agent worktree does not exist: %s", agentName)
			}

			worktreePath := filepath.Join(targetPath, agentName)

			// Output format based on flags
			if printPath {
				// Just print the path for shell substitution
				fmt.Println(worktreePath)
			} else {
				// Print a friendly message with the cd command
				logSuccess("To switch to agent '%s', run:", agentName)
				fmt.Printf("  cd %s\n", worktreePath)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&printPath, "path", "p", false, "Print only the path (for shell substitution)")
	return cmd
}

// ============================================================================
// Hooks Command - Claude Code Integration
// ============================================================================

func newHooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage Claude Code hooks for wt integration",
		Long: `Install or view Claude Code hooks that automatically integrate wt
into the agent lifecycle.

These hooks enable:
- Automatic worktree creation for sessions and subagents
- Conflict detection and prevention
- Safe cleanup of stale worktrees
- True parallel agent execution

Subcommands:
  wt hooks install          Install hooks to current project
  wt hooks config           Print settings.json configuration
  wt hooks list             List available hook scripts`,
	}

	cmd.AddCommand(newHooksInstallCmd())
	cmd.AddCommand(newHooksConfigCmd())
	cmd.AddCommand(newHooksListCmd())

	return cmd
}

func newHooksInstallCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "install [target-dir]",
		Short: "Install Claude Code hooks to a project",
		Long: `Install Claude Code hooks to the specified directory (or current directory).

This creates:
  .claude/settings.json      Hook configuration
  .claude/hooks/*.sh         Hook scripts

The hooks automatically integrate wt with Claude Code, enabling parallel
agent execution without manual workspace management.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir := "."
			if len(args) > 0 {
				targetDir = args[0]
			}

			// Convert to absolute path
			absPath, err := filepath.Abs(targetDir)
			if err != nil {
				return fmt.Errorf("failed to resolve target directory: %w", err)
			}

			// Check if directory exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return fmt.Errorf("target directory does not exist: %s", absPath)
			}

			// Install hooks
			if force {
				return hooks.InstallWithForce(absPath)
			}
			return hooks.Install(absPath)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing hook files")
	return cmd
}

func newHooksConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Print settings.json configuration",
		Long: `Print the Claude Code settings.json configuration for wt hooks.

You can use this to manually add hooks to your project:
  wt hooks config > .claude/settings.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return hooks.PrintConfig()
		},
	}
}

func newHooksListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available hook scripts",
		Long:  `List all available Claude Code hook scripts that can be installed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			scripts, err := hooks.ListScripts()
			if err != nil {
				return err
			}

			fmt.Println("Available hook scripts:")
			for _, script := range scripts {
				fmt.Printf("  - %s\n", script)
			}
			return nil
		},
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
