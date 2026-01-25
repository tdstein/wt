package main

import (
	"fmt"
	"os"

	"github.com/posit-dev/wt/internal/agent"
	"github.com/posit-dev/wt/internal/conflict"
	"github.com/posit-dev/wt/internal/parse"
	"github.com/posit-dev/wt/internal/repo"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wt [repo-url|name] [name]",
		Short: "Git worktree setup for parallel Claude Code agents",
		Long: `wt creates a bare git repository structure optimized for multiple
agents working in parallel worktrees.

Setup a new repository:
  wt <repo-url>           Clone from remote
  wt <repo-url> <name>    Clone from remote with custom name
  wt <name>               Initialize local project`,
		Version: version,
		RunE:    runSetup,
		Args:    cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(newAgentCmd())

	return cmd
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Check prerequisites
	if err := conflict.CheckGitAvailable(); err != nil {
		return err
	}

	// Parse arguments
	result, err := parse.ParseArgs(args)
	if err != nil {
		return err
	}

	logInfo("Mode: %s", result.Mode)
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

	// Setup based on mode
	if result.Mode == "local" {
		logInfo("Initializing local repository...")
		if err := mgr.SetupLocal(); err != nil {
			return fmt.Errorf("failed to setup local repository: %w", err)
		}
	} else {
		logInfo("Cloning from %s...", result.RepoURL)
		if err := mgr.SetupRemote(); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	// Report success
	fmt.Println()
	logSuccess("Worktree ready: %s", result.TargetPath)
	fmt.Println()
	fmt.Println("Create agent worktrees:")
	fmt.Printf("  cd %s\n", result.TargetPath)
	fmt.Println("  wt agent create <agent-name> <task-id> [base-branch]")
	fmt.Println()
	fmt.Println("Manage agents:")
	fmt.Println("  wt agent list    # List all agents")
	fmt.Println("  wt agent status  # Show dashboard")

	return nil
}

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agent worktrees",
		Long:  "Commands for creating and managing agent worktrees",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Find WT_TARGET_PATH by looking for .bare directory
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}
			cmd.SetContext(withTargetPath(cmd.Context(), targetPath))
			return nil
		},
	}

	cmd.AddCommand(newAgentCreateCmd())
	cmd.AddCommand(newAgentRemoveCmd())
	cmd.AddCommand(newAgentListCmd())
	cmd.AddCommand(newAgentCheckCmd())
	cmd.AddCommand(newAgentSyncCmd())
	cmd.AddCommand(newAgentPruneCmd())
	cmd.AddCommand(newAgentStatusCmd())

	return cmd
}

func newAgentCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name> <task-id> [base-branch]",
		Short: "Create a new agent worktree",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := getTargetPath(cmd.Context())
			mgr := agent.NewAgentManager(targetPath)

			agentName := args[0]
			taskID := args[1]
			baseBranch := "main"
			if len(args) > 2 {
				baseBranch = args[2]
			}

			opts := agent.CreateOptions{
				AgentName:  agentName,
				TaskID:     taskID,
				BaseBranch: baseBranch,
			}

			logInfo("Creating agent worktree: %s", agentName)
			if err := mgr.Create(opts); err != nil {
				return err
			}

			logSuccess("Agent worktree created: %s", agentName)
			fmt.Printf("Branch: task/%s/%s\n", taskID, agentName)
			fmt.Printf("Path: %s/%s\n", targetPath, agentName)
			return nil
		},
	}

	return cmd
}

func newAgentRemoveCmd() *cobra.Command {
	var deleteBranch bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an agent worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := getTargetPath(cmd.Context())
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

func newAgentListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all agent worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := getTargetPath(cmd.Context())
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
			fmt.Printf("%-20s %-15s %-30s %-10s %-10s\n",
				"AGENT", "TASK", "BRANCH", "AGE", "STATUS")
			fmt.Println(repeatString("-", 95))

			// Print each agent
			for _, a := range agents {
				fmt.Printf("%-20s %-15s %-30s %-10s %-10s\n",
					a.Agent, a.TaskID, a.Branch, a.AgeHuman, a.Status)
			}

			return nil
		},
	}

	return cmd
}

func newAgentCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check <name>",
		Short: "Check for merge conflicts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := getTargetPath(cmd.Context())
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

func newAgentSyncCmd() *cobra.Command {
	var autoRebase bool

	cmd := &cobra.Command{
		Use:   "sync <name>",
		Short: "Synchronize agent with base branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := getTargetPath(cmd.Context())
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

func newAgentPruneCmd() *cobra.Command {
	var olderThan string
	var dryRun bool
	var interactive bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove stale agent worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := getTargetPath(cmd.Context())
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

func newAgentStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show agent dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := getTargetPath(cmd.Context())
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
			fmt.Printf("%-20s %-15s %-10s %-10s\n",
				"AGENT", "TASK", "AGE", "STATUS")
			fmt.Println(repeatString("-", 60))

			for _, a := range status.Agents {
				statusColor := ""
				if a.Status == "active" {
					statusColor = colorGreen
				} else {
					statusColor = colorYellow
				}
				fmt.Printf("%-20s %-15s %-10s %s%-10s%s\n",
					a.Agent, a.TaskID, a.AgeHuman, statusColor, a.Status, colorReset)
			}

			return nil
		},
	}

	return cmd
}
