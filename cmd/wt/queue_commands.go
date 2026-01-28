package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tdstein/wt/internal/queue"
)

// ============================================================================
// Task-Scoped Queue Commands
// ============================================================================

func newTaskQueueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Task queue operations",
		Long:  "Manage queue operations for the specified task",
	}

	cmd.AddCommand(newTaskQueueAddCmd())
	cmd.AddCommand(newTaskQueueListCmd())
	cmd.AddCommand(newTaskQueueGetCmd())
	cmd.AddCommand(newTaskQueueRemoveCmd())

	return cmd
}

func newTaskQueueAddCmd() *cobra.Command {
	var priority string
	var description string
	var dependencies []string
	var mergeAfter []string
	var baseBranch string
	var tags []string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add this task to the queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			targetPath := getTargetPath(ctx)
			taskID := getTaskID(ctx)

			mgr := queue.NewManager(targetPath)

			// Parse priority
			var p queue.Priority
			switch priority {
			case "high":
				p = queue.PriorityHigh
			case "normal":
				p = queue.PriorityNormal
			case "low":
				p = queue.PriorityLow
			default:
				return fmt.Errorf("invalid priority: %s (must be high, normal, or low)", priority)
			}

			opts := queue.AddOptions{
				TaskID:       taskID,
				Description:  description,
				Priority:     p,
				Dependencies: dependencies,
				MergeAfter:   mergeAfter,
				BaseBranch:   baseBranch,
				Tags:         tags,
			}

			logInfo("Adding task to queue: %s", taskID)
			if err := mgr.Add(opts); err != nil {
				return err
			}

			logSuccess("Task added: %s", taskID)
			fmt.Printf("Priority: %s\n", priority)
			if len(dependencies) > 0 {
				fmt.Printf("Dependencies: %v\n", dependencies)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&priority, "priority", "normal", "Task priority (high, normal, low)")
	cmd.Flags().StringVar(&description, "description", "", "Task description")
	cmd.Flags().StringSliceVar(&dependencies, "depends-on", []string{}, "Task dependencies")
	cmd.Flags().StringSliceVar(&mergeAfter, "merge-after", []string{}, "Tasks to merge after")
	cmd.Flags().StringVar(&baseBranch, "base-branch", "main", "Base branch")
	cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Task tags")

	return cmd
}

func newTaskQueueListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List queue operations for this task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			targetPath := getTargetPath(ctx)
			taskID := getTaskID(ctx)

			mgr := queue.NewManager(targetPath)

			task, err := mgr.Get(taskID)
			if err != nil {
				return fmt.Errorf("task not found: %s", taskID)
			}

			// Print task details
			fmt.Printf("Task ID:      %s\n", task.TaskID)
			fmt.Printf("Description:  %s\n", task.Description)
			fmt.Printf("Priority:     %s\n", task.Priority)
			fmt.Printf("State:        %s\n", task.State)
			fmt.Printf("Base Branch:  %s\n", task.BaseBranch)
			fmt.Printf("Created:      %s\n", task.Created.Format("2006-01-02 15:04:05"))

			if len(task.Dependencies) > 0 {
				fmt.Printf("Dependencies: %v\n", task.Dependencies)
			}

			if len(task.MergeAfter) > 0 {
				fmt.Printf("Merge After:  %v\n", task.MergeAfter)
			}

			if len(task.Tags) > 0 {
				fmt.Printf("Tags:         %v\n", task.Tags)
			}

			if task.ClaimedBy != "" {
				fmt.Printf("Claimed By:   %s\n", task.ClaimedBy)
				fmt.Printf("Claimed At:   %s\n", task.ClaimedAt.Format("2006-01-02 15:04:05"))
			}

			return nil
		},
	}

	return cmd
}

func newTaskQueueGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get details for this task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			targetPath := getTargetPath(ctx)
			taskID := getTaskID(ctx)

			mgr := queue.NewManager(targetPath)

			task, err := mgr.Get(taskID)
			if err != nil {
				return fmt.Errorf("task not found: %s", taskID)
			}

			// Print task details
			fmt.Printf("Task ID:      %s\n", task.TaskID)
			fmt.Printf("Description:  %s\n", task.Description)
			fmt.Printf("Priority:     %s\n", task.Priority)
			fmt.Printf("State:        %s\n", task.State)
			fmt.Printf("Base Branch:  %s\n", task.BaseBranch)
			fmt.Printf("Created:      %s\n", task.Created.Format("2006-01-02 15:04:05"))

			if len(task.Dependencies) > 0 {
				fmt.Printf("Dependencies: %v\n", task.Dependencies)
			}

			if len(task.MergeAfter) > 0 {
				fmt.Printf("Merge After:  %v\n", task.MergeAfter)
			}

			if len(task.Tags) > 0 {
				fmt.Printf("Tags:         %v\n", task.Tags)
			}

			if task.ClaimedBy != "" {
				fmt.Printf("Claimed By:   %s\n", task.ClaimedBy)
				fmt.Printf("Claimed At:   %s\n", task.ClaimedAt.Format("2006-01-02 15:04:05"))
			}

			return nil
		},
	}

	return cmd
}

func newTaskQueueRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove this task from the queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			targetPath := getTargetPath(ctx)
			taskID := getTaskID(ctx)

			mgr := queue.NewManager(targetPath)

			logInfo("Removing task: %s", taskID)
			if err := mgr.Remove(taskID); err != nil {
				return err
			}

			logSuccess("Task removed: %s", taskID)
			return nil
		},
	}

	return cmd
}
