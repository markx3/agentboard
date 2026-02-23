package agent

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

// CursorRunner implements AgentRunner for Cursor CLI ("agent" binary).
type CursorRunner struct{}

func (c *CursorRunner) ID() string     { return "cursor" }
func (c *CursorRunner) Name() string   { return "Cursor" }
func (c *CursorRunner) Binary() string { return "agent" }

func (c *CursorRunner) Available() bool {
	path, err := exec.LookPath("agent")
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), "cursor")
}

func (c *CursorRunner) BuildCommand(opts SpawnOpts) string {
	prompt := buildCursorPrompt(opts)
	return fmt.Sprintf("agent %s", shellQuote(prompt))
}

func buildCursorPrompt(opts SpawnOpts) string {
	task := opts.Task
	shortID := task.ID[:8]

	var b strings.Builder
	b.WriteString("You are working on an agentboard task.\n")
	fmt.Fprintf(&b, "Task: %s  |  ID: %s\n", task.Title, shortID)
	if task.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", task.Description)
	}
	b.WriteString("\n")

	switch task.Status {
	case db.StatusBacklog:
		b.WriteString("STAGE: Backlog — Project Ideation\n")
		b.WriteString("Explore ideas and brainstorm approaches for this task.\n")
		b.WriteString("When brainstorming is complete, move to planning:\n")
		fmt.Fprintf(&b, "  agentboard task move %s planning\n", shortID)
	case db.StatusPlanning:
		b.WriteString("STAGE: Planning — Implementation Design\n")
		b.WriteString("Create a detailed implementation plan for this task.\n")
		b.WriteString("When the plan is ready, move to in progress:\n")
		fmt.Fprintf(&b, "  agentboard task move %s in_progress\n", shortID)
	case db.StatusInProgress:
		b.WriteString("STAGE: In Progress — Implementation\n")
		b.WriteString("Implement this task based on the plan.\n")
		b.WriteString("When implementation is complete and a PR is opened, move to done:\n")
		fmt.Fprintf(&b, "  agentboard task move %s done\n", shortID)
	case db.StatusDone:
		b.WriteString("STAGE: Done — Verification & Cleanup\n")
		b.WriteString("Verify that the pull request has been opened and merged to main.\n")
		b.WriteString("Then clean up the git worktree for this task.\n")
	default:
		b.WriteString("Begin working on this task.\n")
		b.WriteString("When you are done, move the task to the next column using the agentboard CLI:\n")
		fmt.Fprintf(&b, "  agentboard task move %s <status>\n", shortID)
	}

	return b.String()
}
