package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tdstein/wt/internal/agent"
	"github.com/tdstein/wt/internal/conflict"
	"github.com/tdstein/wt/internal/parse"
	"github.com/tdstein/wt/internal/repo"
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
  wt check <name>                      Check for conflicts
  wt sync <name>                       Sync with base branch
  wt prune [--older-than 7d]          Remove stale worktrees`,
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
	if err := conflict.CheckGitAvailable(); err != nil {
		return err
	}

	repoURL := args[0]
	if !parse.IsURL(repoURL) {
		return fmt.Errorf("invalid repository URL: %s", repoURL)
	}

	// Parse arguments
	result, err := parse.ParseArgs(args)
	if err != nil {
		return err
	}

	logInfo("Target: %s", result.TargetPath)

	// Create repository manager
	mgr := repo.NewManager(result.TargetPath, result.RepoURL)

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
	fmt.Println("Create agent worktrees:")
	fmt.Printf("  cd %s\n", result.TargetPath)
	fmt.Println("  wt add <agent-name> [base-branch]")

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
	if err := conflict.CheckGitAvailable(); err != nil {
		return err
	}

	targetPath := args[0]

	// Expand to absolute path
	result, err := parse.ParseArgs([]string{targetPath})
	if err != nil {
		return err
	}

	logInfo("Target: %s", result.TargetPath)

	// Create repository manager
	mgr := repo.NewManager(result.TargetPath, "")

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
	fmt.Println("Create agent worktrees:")
	fmt.Printf("  cd %s\n", result.TargetPath)
	fmt.Println("  wt add <agent-name> [base-branch]")

	return nil
}

// ============================================================================
// Add Command - Create Agent Worktree
// ============================================================================

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> [base-branch]",
		Short: "Create a new agent worktree",
		Long: `Create a new agent worktree with branch work/<name>.

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

	mgr := agent.NewAgentManager(targetPath)
	agentName := args[0]
	baseBranch := "main"
	if len(args) > 1 {
		baseBranch = args[1]
	}

	opts := agent.CreateOptions{
		AgentName:  agentName,
		BaseBranch: baseBranch,
	}

	logInfo("Creating agent worktree: %s", agentName)
	if err := mgr.Create(opts); err != nil {
		return err
	}

	logSuccess("Agent worktree created: %s", agentName)
	fmt.Printf("Branch: work/%s\n", agentName)
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

			mgr := agent.NewAgentManager(targetPath)
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

			mgr := agent.NewAgentManager(targetPath)
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

			mgr := agent.NewAgentManager(targetPath)
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

			mgr := agent.NewAgentManager(targetPath)
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
			fmt.Printf("Divergence: %s\n", conflict.FormatDivergence(result.Divergence))
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
		Use:   "sync <name>",
		Short: "Synchronize agent with base branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}

			mgr := agent.NewAgentManager(targetPath)
			opts := agent.SyncOptions{
				AgentName:  args[0],
				AutoRebase: autoRebase,
			}

			logInfo("Syncing agent: %s", args[0])

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

			fmt.Printf("Divergence: %s\n", conflict.FormatDivergence(result.Divergence))
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

			mgr := agent.NewAgentManager(targetPath)

			days := 7
			if olderThan != "" {
				var err error
				days, err = agent.ParseOlderThan(olderThan)
				if err != nil {
					return err
				}
			}

			opts := agent.PruneOptions{
				OlderThanDays: days,
				DryRun:        dryRun,
				Interactive:   interactive,
			}

			logInfo("Pruning stale agents (older than %d days)...", days)

			result, err := mgr.Prune(opts)
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
