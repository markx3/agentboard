package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent lifecycle commands",
}

var requestResetCmd = &cobra.Command{
	Use:   "request-reset <task-id>",
	Short: "Request fresh context for next stage",
	Long:  "Sets a reset flag on the task. When the agent exits and the TUI detects the window is gone, it will respawn a fresh agent with the next-stage prompt.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRequestReset,
}

func init() {
	agentCmd.AddCommand(requestResetCmd)
	rootCmd.AddCommand(agentCmd)
}

func runRequestReset(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()

	// Find task by prefix
	tasks, err := svc.ListTasks(ctx)
	if err != nil {
		return err
	}
	fullID := findByPrefix(tasks, args[0])
	if fullID == "" {
		return fmt.Errorf("task not found: %s", args[0])
	}

	task, err := svc.GetTask(ctx, fullID)
	if err != nil {
		return err
	}

	if task.AgentStatus != db.AgentActive {
		fmt.Printf("Warning: task %s has no active agent (status: %s)\n", task.ID[:8], task.AgentStatus)
	}

	task.ResetRequested = true
	if err := svc.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("setting reset flag: %w", err)
	}

	fmt.Printf("Reset requested for task %s (%s)\n", task.ID[:8], task.Title)
	return nil
}
