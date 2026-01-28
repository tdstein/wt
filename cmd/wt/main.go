package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tdstein/wt/internal/agent"
	"github.com/tdstein/wt/internal/conflict"
	"github.com/tdstein/wt/internal/locking"
	"github.com/tdstein/wt/internal/parse"
	"github.com/tdstein/wt/internal/queue"
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

Setup a new repository:
  wt https://github.com/user/repo       Clone from remote
  wt https://github.com/user/repo proj  Clone with custom name
  wt myproject                          Initialize local project

Create and manage agents:
  wt create <name> <task-id> [base]     Create agent worktree
  wt <task-id> lock claim <agent>       Claim lock for task
  wt <task-id> queue add                Add task to queue`,
		Version: version,
		RunE:    runRootOrSetup,
		Args:    cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newTaskCmd())

	return cmd
}

func runRootOrSetup(cmd *cobra.Command, args []string) error {
	// Check if this is a setup command (URL or local init)
	firstArg := args[0]

	// If it's a URL, do setup
	if parse.IsURL(firstArg) {
		return runSetup(cmd, args)
	}

	// If not in a wt directory and not a URL, treat as local setup
	_, err := findWtRoot()
	if err != nil {
		// Not in wt directory, try local setup
		return runSetup(cmd, args)
	}

	// In a wt directory but no recognized command
	return fmt.Errorf("unknown command or operation")
}

// ============================================================================
// Setup Command (Unchanged)
// ============================================================================

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
	fmt.Println("  wt create <agent-name> <task-id> [base-branch]")
	fmt.Println()
	fmt.Println("Manage tasks:")
	fmt.Println("  wt <task-id> lock claim <agent-name>")
	fmt.Println("  wt <task-id> queue add")

	return nil
}

// ============================================================================
// Create Command (Promoted from agent create)
// ============================================================================

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name> <task-id> [base-branch]",
		Short: "Create a new agent worktree",
		Args:  cobra.RangeArgs(2, 3),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}
			cmd.SetContext(withTargetPath(cmd.Context(), targetPath))
			return nil
		},
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

// ============================================================================
// Task Command (Dynamic Router)
// ============================================================================

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "<task-id> <operation> [args...]",
		Short: "Task-specific operations",
		Long:  "Perform operations scoped to a specific task (lock, queue, agent management)",
		Args:  cobra.MinimumNArgs(2),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			targetPath, err := findWtRoot()
			if err != nil {
				return err
			}
			taskID := args[0]
			ctx := withTargetPath(cmd.Context(), targetPath)
			ctx = withTaskID(ctx, taskID)
			cmd.SetContext(ctx)
			return nil
		},
	}

	cmd.AddCommand(newTaskLockCmd())
	cmd.AddCommand(newTaskQueueCmd())
	cmd.AddCommand(newTaskAgentCmd())

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
