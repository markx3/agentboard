package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/marcosfelipeeipper/agentboard/internal/agent"
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

var agentStartCmd = &cobra.Command{
	Use:   "start <task-id>",
	Short: "Spawn an agent for a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentStart,
}

var agentKillCmd = &cobra.Command{
	Use:   "kill <task-id>",
	Short: "Kill a running agent for a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentKill,
}

var (
	agentStartRunner     string
	agentSkipPermissions bool
	agentOutputJSON      bool
)

func init() {
	agentStartCmd.Flags().StringVar(&agentStartRunner, "runner", "", "agent runner (claude, cursor)")
	agentStartCmd.Flags().BoolVar(&agentSkipPermissions, "skip-permissions", false, "skip permission prompts")
	agentStartCmd.Flags().BoolVar(&agentOutputJSON, "json", false, "output as JSON")
	agentKillCmd.Flags().BoolVar(&agentOutputJSON, "json", false, "output as JSON")

	agentCmd.AddCommand(requestResetCmd, agentStartCmd, agentKillCmd)
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

func runAgentStart(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
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

	if task.AgentStatus == db.AgentActive {
		return fmt.Errorf("agent already running on task %s", task.ID[:8])
	}

	// Determine runner
	var runner agent.AgentRunner
	if agentStartRunner != "" {
		runner = agent.GetRunner(agentStartRunner)
		if runner == nil {
			return fmt.Errorf("unknown runner: %s", agentStartRunner)
		}
		if !runner.Available() {
			return fmt.Errorf("runner %s not available", agentStartRunner)
		}
	} else {
		available := agent.AvailableRunners()
		if len(available) == 0 {
			return fmt.Errorf("no agent runners available")
		}
		runner = available[0]
	}

	if agentSkipPermissions {
		task.SkipPermissions = true
		if err := svc.UpdateTask(ctx, task); err != nil {
			return fmt.Errorf("updating skip_permissions: %w", err)
		}
	}

	if err := agent.Spawn(ctx, svc, *task, runner); err != nil {
		return fmt.Errorf("spawning agent: %w", err)
	}

	if agentOutputJSON {
		task, _ = svc.GetTask(ctx, fullID)
		if task != nil {
			return json.NewEncoder(os.Stdout).Encode(task)
		}
	}

	fmt.Printf("Agent %s spawned for task %s (%s)\n", runner.Name(), task.ID[:8], task.Title)
	return nil
}

func runAgentKill(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
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
		return fmt.Errorf("no active agent on task %s", task.ID[:8])
	}

	if err := agent.Kill(ctx, svc, *task); err != nil {
		return fmt.Errorf("killing agent: %w", err)
	}

	if agentOutputJSON {
		task, _ = svc.GetTask(ctx, fullID)
		if task != nil {
			return json.NewEncoder(os.Stdout).Encode(task)
		}
	}

	fmt.Printf("Agent killed for task %s (%s)\n", task.ID[:8], task.Title)
	return nil
}
