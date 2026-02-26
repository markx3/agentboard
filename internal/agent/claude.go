package agent

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/markx3/agentboard/internal/db"
)

// ClaudeRunner implements AgentRunner for Claude Code CLI.
type ClaudeRunner struct{}

func (c *ClaudeRunner) ID() string     { return "claude" }
func (c *ClaudeRunner) Name() string   { return "Claude Code" }
func (c *ClaudeRunner) Binary() string { return "claude" }

func (c *ClaudeRunner) Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (c *ClaudeRunner) BuildCommand(opts SpawnOpts) string {
	sysPrompt := buildClaudeSystemPrompt(opts)
	initialPrompt := buildClaudeInitialPrompt(opts)
	skipFlag := ""
	if opts.Task.SkipPermissions {
		skipFlag = "--dangerously-skip-permissions "
	}
	return fmt.Sprintf("claude %s-w %s --append-system-prompt %s %s",
		skipFlag,
		shellQuote(opts.WorkDir),
		shellQuote(sysPrompt),
		shellQuote(initialPrompt),
	)
}

func buildClaudeSystemPrompt(opts SpawnOpts) string {
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
		b.WriteString("STAGE: Backlog — Unplanned\n")
		b.WriteString("Move to brainstorm to begin work:\n")
		fmt.Fprintf(&b, "  agentboard task move %s brainstorm\n", shortID)
	case db.StatusBrainstorm:
		b.WriteString("STAGE: Brainstorm — Exploring Ideas\n")
		b.WriteString("When brainstorming is complete, move to planning:\n")
		fmt.Fprintf(&b, "  agentboard task move %s planning\n", shortID)
	case db.StatusPlanning:
		b.WriteString("STAGE: Planning — Implementation Design\n")
		b.WriteString("When the plan is ready, move to in progress:\n")
		fmt.Fprintf(&b, "  agentboard task move %s in_progress\n", shortID)
	case db.StatusInProgress:
		b.WriteString("STAGE: In Progress — Implementation\n")
		b.WriteString("When implementation is complete and a PR is opened, move to done:\n")
		fmt.Fprintf(&b, "  agentboard task move %s done\n", shortID)
	case db.StatusDone:
		b.WriteString("STAGE: Done — Verification & Cleanup\n")
		b.WriteString("Verify that the pull request has been opened and merged to main.\n")
		b.WriteString("Then clean up the git worktree for this task.\n")
	default:
		b.WriteString("When you are done, move the task to the next column using the agentboard CLI:\n")
		fmt.Fprintf(&b, "  agentboard task move %s <status>\n", shortID)
	}

	b.WriteString("\nTASK METADATA:\n")
	b.WriteString("Update task fields as you work:\n")
	fmt.Fprintf(&b, "  agentboard task update %s --branch \"<branch-name>\"\n", shortID)
	fmt.Fprintf(&b, "  agentboard task update %s --pr-url \"<url>\"\n", shortID)
	fmt.Fprintf(&b, "  agentboard task update %s --assignee \"<name>\"\n", shortID)

	b.WriteString("\nDEPENDENCIES:\n")
	b.WriteString("Mark task dependencies:\n")
	fmt.Fprintf(&b, "  agentboard task block %s <blocker-id>   # this task is blocked by another\n", shortID)
	fmt.Fprintf(&b, "  agentboard task unblock %s <blocker-id> # remove a dependency\n", shortID)

	b.WriteString("\nACTIVITY REPORTING:\n")
	b.WriteString("Update your activity status so the board shows what you're doing:\n")
	fmt.Fprintf(&b, "  agentboard agent status %s \"<brief description>\"\n", shortID)
	b.WriteString("Update when starting each major step (reading code, writing implementation, running tests, creating PR).\n")

	return b.String()
}

func buildClaudeInitialPrompt(opts SpawnOpts) string {
	switch opts.Task.Status {
	case db.StatusBacklog:
		return "This task is in backlog. Move it to brainstorm to begin work."
	case db.StatusBrainstorm:
		return "Run /workflows:brainstorm to explore ideas for this task."
	case db.StatusPlanning:
		return "Run /workflows:plan to create a detailed implementation plan for this task."
	case db.StatusInProgress:
		return "Run /workflows:work to implement this task based on the plan."
	case db.StatusDone:
		return "Verify the pull request is merged and clean up the git worktree."
	default:
		return "Begin working on this task."
	}
}

func (c *ClaudeRunner) BuildEnrichmentCommand(opts SpawnOpts) string {
	task := opts.Task
	shortID := task.ID[:8]
	prompt := fmt.Sprintf(
		"Enrich task %q (%s): run `git log --oneline -10` and `agentboard task list --json`, "+
			"then update the description and leave a comment. "+
			"Commands: `agentboard task update %s --description \"<enriched>\"` "+
			"and `agentboard task comment %s --author enrichment --body \"<one sentence>\"`.",
		task.Title, shortID, shortID, shortID,
	)
	return fmt.Sprintf("claude --dangerously-skip-permissions -w %s --print %s",
		shellQuote(opts.WorkDir), shellQuote(prompt))
}
