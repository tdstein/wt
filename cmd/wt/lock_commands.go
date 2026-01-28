package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tdstein/wt/internal/locking"
)

// ============================================================================
// Task-Scoped Lock Commands
// ============================================================================

func newTaskLockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Task lock operations",
		Long:  "Manage locks for the specified task",
	}

	cmd.AddCommand(newTaskLockClaimCmd())
	cmd.AddCommand(newTaskLockReleaseCmd())
	cmd.AddCommand(newTaskLockListCmd())
	cmd.AddCommand(newTaskLockCleanCmd())

	return cmd
}

func newTaskLockClaimCmd() *cobra.Command {
	var pid int

	cmd := &cobra.Command{
		Use:   "claim <agent-name>",
		Short: "Claim a lock for this task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			targetPath := getTargetPath(ctx)
			taskID := getTaskID(ctx)
			agentName := args[0]

			mgr := locking.NewManager(targetPath)

			logInfo("Claiming lock for task: %s", taskID)
			if err := mgr.Claim(taskID, agentName, pid); err != nil {
				return err
			}

			logSuccess("Lock claimed: %s by %s", taskID, agentName)
			return nil
		},
	}

	cmd.Flags().IntVar(&pid, "pid", 0, "Process ID of the agent")
	return cmd
}

func newTaskLockReleaseCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "release <agent-name>",
		Short: "Release a lock for this task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			targetPath := getTargetPath(ctx)
			taskID := getTaskID(ctx)
			agentName := args[0]

			mgr := locking.NewManager(targetPath)

			if force {
				logInfo("Force releasing lock: %s", taskID)
				if err := mgr.ForceRelease(taskID); err != nil {
					return err
				}
				logSuccess("Lock force released: %s", taskID)
				return nil
			}

			logInfo("Releasing lock: %s", taskID)
			if err := mgr.Release(taskID, agentName); err != nil {
				return err
			}

			logSuccess("Lock released: %s by %s", taskID, agentName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force release without agent verification")
	return cmd
}

func newTaskLockListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List locks for this task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			targetPath := getTargetPath(ctx)
			taskID := getTaskID(ctx)

			mgr := locking.NewManager(targetPath)

			locks, err := mgr.ListAll()
			if err != nil {
				return err
			}

			// Filter to only this task
			var taskLocks []*locking.Lock
			for _, lock := range locks {
				if lock.TaskID == taskID {
					taskLocks = append(taskLocks, lock)
				}
			}

			if len(taskLocks) == 0 {
				fmt.Printf("No active locks for task: %s\n", taskID)
				return nil
			}

			// Print table header
			fmt.Printf("%-15s %-20s %-10s\n", "AGENT", "CLAIMED", "PID")
			fmt.Println(repeatString("-", 50))

			// Print each lock
			for _, lock := range taskLocks {
				age := time.Since(lock.ClaimedAt)
				ageStr := formatDuration(age)

				pidStr := "-"
				if lock.PID != 0 {
					pidStr = fmt.Sprintf("%d", lock.PID)
				}

				fmt.Printf("%-15s %-20s %-10s\n", lock.AgentName, ageStr, pidStr)
			}

			fmt.Printf("\nTotal: %d lock(s) for task %s\n", len(taskLocks), taskID)
			return nil
		},
	}

	return cmd
}

func newTaskLockCleanCmd() *cobra.Command {
	var timeout string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean stale locks for this task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			targetPath := getTargetPath(ctx)
			taskID := getTaskID(ctx)

			mgr := locking.NewManager(targetPath)

			// Parse timeout
			duration, err := time.ParseDuration(timeout)
			if err != nil {
				return fmt.Errorf("invalid timeout format: %w", err)
			}

			if dryRun {
				// List stale locks without removing
				staleLocks, err := mgr.ListStale(duration)
				if err != nil {
					return err
				}

				// Filter to this task
				var taskStaleLocks []*locking.Lock
				for _, lock := range staleLocks {
					if lock.TaskID == taskID {
						taskStaleLocks = append(taskStaleLocks, lock)
					}
				}

				if len(taskStaleLocks) == 0 {
					fmt.Printf("No stale locks found for task: %s\n", taskID)
					return nil
				}

				fmt.Printf("Found %d stale lock(s) for task %s (dry run):\n", len(taskStaleLocks), taskID)
				for _, lock := range taskStaleLocks {
					age := time.Since(lock.LastActive)
					fmt.Printf("  - %s (age: %s)\n", lock.AgentName, formatDuration(age))
				}
				return nil
			}

			// Clean stale locks
			logInfo("Cleaning stale locks for task %s (timeout: %s)...", taskID, duration)
			removed, err := mgr.CleanStale(duration)
			if err != nil {
				return err
			}

			// Filter to this task
			var taskRemoved []string
			for _, id := range removed {
				if id == taskID {
					taskRemoved = append(taskRemoved, id)
				}
			}

			if len(taskRemoved) == 0 {
				fmt.Printf("No stale locks found for task: %s\n", taskID)
				return nil
			}

			logSuccess("Removed %d stale lock(s) for task %s", len(taskRemoved), taskID)
			return nil
		},
	}

	cmd.Flags().StringVar(&timeout, "timeout", "1h", "Lock timeout duration (e.g., 30m, 1h, 2h)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be cleaned without actually cleaning")

	return cmd
}
